package jsonexporter

import (
	"fmt"
	"math"
	"strconv"
	"math"

	"github.com/kawamuray/jsonpath" // Originally: "github.com/NickSardo/jsonpath"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/superq/prometheus-json-exporter/harness"
)

type JsonScraper interface {
	Scrape(data []byte, reg *harness.MetricRegistry) error
}

type ValueScraper struct {
	*Mapping
	valueJsonPath *jsonpath.Path
}

func NewValueScraper(mapping *Mapping) (JsonScraper, error) {
	valuepath, err := compilePath(mapping.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path;path:<%s>,err:<%s>", mapping.Path, err)
	}

	scraper := &ValueScraper{
		Mapping:       mapping,
		valueJsonPath: valuepath,
	}
	return scraper, nil
}

func (vs *ValueScraper) parseValue(bytes []byte) (float64, error) {
	value, err := strconv.ParseFloat(string(bytes), 64)
	if err != nil {
		return -1.0, fmt.Errorf("failed to parse value as float;value:<%s>", bytes)
	}
	return value, nil
}

func (vs *ValueScraper) forTargetValue(data []byte, handle func(*jsonpath.Result)) error {
	eval, err := jsonpath.EvalPathsInBytes(data, []*jsonpath.Path{vs.valueJsonPath})
	if err != nil {
		return fmt.Errorf("failed to eval jsonpath;path:<%s>,json:<%s>", vs.valueJsonPath, data)
	}

	for {
		result, ok := eval.Next()
		if !ok {
			break
		}
		handle(result)
	}
	return nil
}

func (vs *ValueScraper) Scrape(data []byte, reg *harness.MetricRegistry) error {
	isFirst := true
	return vs.forTargetValue(data, func(result *jsonpath.Result) {
		if !isFirst {
			log.Infof("ignoring non-first value;path:<%s>", vs.valueJsonPath)
			return
		}
		isFirst = false

		if result.Type != jsonpath.JsonNumber {
			log.Warnf("skipping not numerical result;path:<%s>,value:<%s>",
				vs.valueJsonPath, result.Value)
			return
		}

		value, err := vs.parseValue(result.Value)
		if err != nil {
			// Should never happen.
			log.Errorf("could not parse numerical value as float;path:<%s>,value:<%s>",
				vs.valueJsonPath, result.Value)
			return
		}

		log.Debugf("metric updated;name:<%s>,labels:<%s>,value:<%.2f>", vs.Name, vs.Labels, value)
		reg.Get(vs.Name).(*prometheus.GaugeVec).With(vs.Labels).Set(value)
	})
}

type SingleObjectScraper struct {
	*ValueScraper
	labelJsonPaths map[string]*jsonpath.Path
}

func NewSingleObjectScraper(mapping *Mapping) (JsonScraper, error) {
	valueScraper, err := NewValueScraper(mapping)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path;path:<%s>,err:<%s>", mapping.Path, err)
	}

	labelPaths, err := compilePaths(mapping.Labels)
	if err != nil {
		return nil, err
	}	

	scraper := &SingleObjectScraper{
		ValueScraper:   valueScraper.(*ValueScraper),
		labelJsonPaths: labelPaths,
	}
	return scraper, nil
}

func (sobsc *SingleObjectScraper) newLabels() map[string]string {
	labels := make(map[string]string)
	for name, value := range sobsc.Labels {
		if _, ok := sobsc.labelJsonPaths[name]; !ok {
			// Static label value.
			labels[name] = value
		}
	}
	return labels
}

func (sobsc *SingleObjectScraper) parseValue(bytes []byte) (float64, error) {
	value, err := strconv.ParseFloat(string(bytes), 64)
	if err != nil {
		return -1.0, fmt.Errorf("failed to parse value as float;value:<%s>", bytes)
	}
	return value, nil
}

func (sobsc *SingleObjectScraper) forTargetValue(data []byte, handle func(*jsonpath.Result)) error {
	eval, err := jsonpath.EvalPathsInBytes(data, []*jsonpath.Path{sobsc.valueJsonPath})
	if err != nil {
		return fmt.Errorf("failed to eval jsonpath;path:<%s>,json:<%s>", sobsc.valueJsonPath, data)
	}

	for {
		result, ok := eval.Next()
		if !ok {
			break
		}
		handle(result)
	}
	return nil
}

func (sobsc *SingleObjectScraper) extractFirstValue(data []byte, path *jsonpath.Path) (*jsonpath.Result, error) {
	eval, err := jsonpath.EvalPathsInBytes(data, []*jsonpath.Path{path})
	if err != nil {
		return nil, fmt.Errorf("failed to eval jsonpath;err:<%s>", err)
	}

	result, ok := eval.Next()
	if !ok {
		return nil, fmt.Errorf("no value found for path")
	}
	return result, nil
}


func (sobsc *SingleObjectScraper) Scrape(data []byte, reg *harness.MetricRegistry) error {
	isFirst := true
	return sobsc.forTargetValue(data, func(result *jsonpath.Result) {
		if !isFirst {
			log.Infof("ignoring non-first value;path:<%s>", sobsc.valueJsonPath)
			return
		}
		isFirst = false

		if result.Type != jsonpath.JsonNumber {
			log.Warnf("skipping not numerical result;path:<%s>,value:<%s>",
				sobsc.valueJsonPath, result.Value)
			return
		}

		value, err := sobsc.parseValue(result.Value)
		if err != nil {
			// Should never happen.
			log.Errorf("could not parse numerical value as float;path:<%s>,value:<%s>",
				sobsc.valueJsonPath, result.Value)
			return
		}

		labels := sobsc.newLabels()
		for name, path := range sobsc.labelJsonPaths {
			firstResult, err := sobsc.extractFirstValue(data, path)
			if err != nil {
				log.Warnf("could not find value for label path;path:<%s>,json:<%s>,err:<%s>", path, data, err)
				continue
			}
			value := firstResult.Value
			if firstResult.Type == jsonpath.JsonString {
				// Strip quotes
				value = value[1 : len(value)-1]
			}
			labels[name] = string(value)
		}
		log.Debugf("metric updated;name:<%s>,labels:<%s>,value:<%.2f>", sobsc.Name, labels, value)
		reg.Get(sobsc.Name).(*prometheus.GaugeVec).With(labels).Set(value)
	})
}

type ObjectScraper struct {
	*ValueScraper
	labelJsonPaths map[string]*jsonpath.Path
	valueJsonPaths map[string]*jsonpath.Path
}

func NewObjectScraper(mapping *Mapping) (JsonScraper, error) {
	valueScraper, err := NewValueScraper(mapping)
	if err != nil {
		return nil, err
	}

	labelPaths, err := compilePaths(mapping.Labels)
	if err != nil {
		return nil, err
	}
	valuePaths, err := compilePaths(mapping.Values)
	if err != nil {
		return nil, err
	}
	scraper := &ObjectScraper{
		ValueScraper:   valueScraper.(*ValueScraper),
		labelJsonPaths: labelPaths,
		valueJsonPaths: valuePaths,
	}
	return scraper, nil
}

func (obsc *ObjectScraper) newLabels() map[string]string {
	labels := make(map[string]string)
	for name, value := range obsc.Labels {
		if _, ok := obsc.labelJsonPaths[name]; !ok {
			// Static label value.
			labels[name] = value
		}
	}
	return labels
}

func (obsc *ObjectScraper) extractFirstValue(data []byte, path *jsonpath.Path) (*jsonpath.Result, error) {
	eval, err := jsonpath.EvalPathsInBytes(data, []*jsonpath.Path{path})
	if err != nil {
		return nil, fmt.Errorf("failed to eval jsonpath;err:<%s>", err)
	}

	result, ok := eval.Next()
	if !ok {
		return nil, fmt.Errorf("no value found for path")
	}
	return result, nil
}

func (obsc *ObjectScraper) Scrape(data []byte, reg *harness.MetricRegistry) error {
	return obsc.forTargetValue(data, func(result *jsonpath.Result) {
		if result.Type != jsonpath.JsonObject && result.Type != jsonpath.JsonArray {
			log.Warnf("skipping not structual result;path:<%s>,value:<%s>",
				obsc.valueJsonPath, result.Value)
			return
		}

		labels := obsc.newLabels()
		for name, path := range obsc.labelJsonPaths {
			firstResult, err := obsc.extractFirstValue(result.Value, path)
			if err != nil {
				log.Warnf("could not find value for label path;path:<%s>,json:<%s>,err:<%s>", path, result.Value, err)
				continue
			}
			value := firstResult.Value
			if firstResult.Type == jsonpath.JsonString {
				// Strip quotes
				value = value[1 : len(value)-1]
			}
			labels[name] = string(value)
		}

		for name, mappingValue := range obsc.Values {
			var metricValue float64
			path := obsc.valueJsonPaths[name]

			if path == nil {
				// Static value
				value, err := obsc.parseValue([]byte(mappingValue))
				if err != nil {
					log.Errorf("could not use configured value as float number;name:<%s>,err:<%s>", err)
					continue
				}
				metricValue = value
			} else {
				// Dynamic value
				firstResult, err := obsc.extractFirstValue(result.Value, path)
				if err != nil {
					log.Warnf("could not find value for value path;path:<%s>,json:<%s>,err:<%s>", path, result.Value, err)
					continue
				}

				//if firstResult.Type != jsonpath.JsonNumber {
				//	log.Warnf("skipping not numerical result;path:<%s>,value:<%s>",
				//		obsc.valueJsonPath, result.Value)
				//	continue
				//}
				//
				//value, err := obsc.parseValue(firstResult.Value)

				var value float64
				var boolValue bool
				var err error
				switch firstResult.Type {
				case jsonpath.JsonNumber:
					value, err = obsc.parseValue(firstResult.Value)
				case jsonpath.JsonNull:
					value = math.NaN()
				case jsonpath.JsonBool:
					if boolValue, err = strconv.ParseBool(string(firstResult.Value)); boolValue {
						value = 1
					} else {
						value = 0
					}
				default:
					log.Warnf("skipping not numerical result;path:<%v>,value:<%s>",
						obsc.valueJsonPath, result.Value)
					continue
				}


				if err != nil {
					// Should never happen.
					log.Errorf("could not parse numerical value as float;path:<%s>,value:<%s>",
						obsc.valueJsonPath, firstResult.Value)
					continue
				}
				metricValue = value
			}

			fqn := harness.MakeMetricName(obsc.Name, name)
			log.Debugf("metric updated;name:<%s>,labels:<%s>,value:<%.2f>", fqn, labels, metricValue)
			reg.Get(fqn).(*prometheus.GaugeVec).With(labels).Set(metricValue)
		}
	})
}

