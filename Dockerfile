FROM golang:1.13-alpine as builder
COPY . /build
WORKDIR /build

ENV GO111MODULE=on
ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

RUN apk add git \
    && go mod vendor

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/dunder cmd/dunder.go

FROM scratch
COPY --from=builder /build/bin/dunder .

ADD https://github.com/golang/go/raw/master/lib/time/zoneinfo.zip /zoneinfo.zip
ENV ZONEINFO /zoneinfo.zip

ADD https://curl.haxx.se/ca/cacert.pem /etc/ssl/certs/ca-certificates.crt

EXPOSE 9000

ENTRYPOINT ["/dunder"]
