FROM registry.js.design/base/golang:1.18-alpine3.16 AS build

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://goproxy.cn,direct

WORKDIR /root

COPY . /root

RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o process ./process.go && \
    chmod 777 process

FROM registry.js.design/base/alpine:3.16

COPY --from=build /root/process /app/process

WORKDIR /app

ENTRYPOINT ["/app/process"]
