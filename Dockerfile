FROM scratch
COPY bin/dunder .

ADD https://github.com/golang/go/raw/master/lib/time/zoneinfo.zip /zoneinfo.zip
ENV ZONEINFO /zoneinfo.zip

ADD https://curl.haxx.se/ca/cacert.pem /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/dunder"]