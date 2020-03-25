FROM golang:alpine as builder

ADD . /opt/json_exporter
WORKDIR /opt/json_exporter
ENV GO111MODULE=on

RUN apk add --no-cache gcc musl-dev git && go build -o json_exporter .

FROM alpine:3.11

RUN  mkdir -p /opt/json_exporter

WORKDIR /opt/json_exporter

COPY --from=builder /opt/json_exporter/json_exporter .
#COPY --from=builder /opt/json_exporter/entrypoint.sh .

#RUN chmod +x ./json_exporter && chmod +x ./entrypoint.sh
RUN chmod +x ./json_exporter

EXPOSE 7979
#ENTRYPOINT ["./entrypoint.sh"]

CMD  ./json_exporter  ${ENDPOINT_JSON} /opt/json_exporter/config.yml