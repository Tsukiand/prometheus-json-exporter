FROM golang:alpine as builder

ADD . /opt/json_exporter
WORKDIR /opt/json_exporter
ENV GO111MODULE=on

RUN apk add --no-cache gcc musl-dev git && go build -o json_exporter -v

FROM alpine:3.11

WORKDIR /opt/

COPY --from=builder /opt/json_exporter .

RUN chmod +x ./json_exporter && chmod +x ./entrypoint.sh


EXPOSE 7979
ENTRYPOINT ["./entrypoint.sh"]
