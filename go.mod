module github.com/superq/prometheus-json-exporter

go 1.13

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.2

require (
	github.com/Sirupsen/logrus v0.0.0-00010101000000-000000000000 // indirect
	github.com/kawamuray/jsonpath v0.0.0-20160208140654-5c448ebf9735
	github.com/kawamuray/prometheus-exporter-harness v0.0.0-20160809123908-97eeea7b8b06
	github.com/kawamuray/prometheus-json-exporter v0.0.0-20171006005928-51e3dc02a30a
	github.com/prometheus/client_golang v1.3.0
	github.com/sirupsen/logrus v1.4.2
	github.com/urfave/cli v1.22.2
	gopkg.in/yaml.v2 v2.2.8
)
