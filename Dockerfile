FROM golang:alpine

WORKDIR /opt/json_exporter
COPY . /opt/json_exporter
ENV GO111MODULE=on

#RUN echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.4/main/" > /etc/apk/repositories

RUN apk add --no-cache gcc musl-dev git && go build -I -v -o json_exporter 
RUN chmod +x ./json_exporter && chmod +x ./entrypoint.sh

EXPOSE 7979
ENTRYPOINT ["./entrypoint.sh"]
