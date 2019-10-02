# Dunder - geeky message API

Dunder the message JSON API provides gateway for sharing messages directly out of 
you console. For best experience you may want to write your own tool for
sharing messages or just use curl :)

# Status - under heavy development

Don't consider Dunder as stable, everything may change. Beware software may also
contain bugs!

# Local development setup

Some tips how to get service up and running. Please familiarize yourself
with `Makefile` it may provide some helpful commands. Note that `go.mod` file
requires Go 1.13, it should be safe to build service with older versions.

## Setup tls certs

TLS certs for development are stored under `tls` directory. Use 
[mkcert](https://github.com/FiloSottile/mkcert) to generate valid local
certificates.
```bash
$ mkcert -install
$ mkcert -cert-file ./tls/crt.pem -key-file ./tls/key.pem dunder.io "*.dunder.io" dunder.test localhost 127.0.0.1 ::1
```

## Setup local database

Currently Dunder provides only `CockroachDB` backend. However implementation
is based on [gorm](https://gorm.io/) so potentially it's possible to port
it to other databases. Note that some inline SQL may be compatible only with Postgres.

Run `docker-compose up` to setup local CockroachDB cluster. Setup provides useful
web interface at port `8080`. Please consult `docker-compose.yaml` as some folder may
need to be created to mount cluster data volumes.

Next you need to create database could be set later with `--cockroach_db.database`
flag. For example using `postgresql-client`:

```bash
$ createdb -p 26257 -h localhost -U root -e live_database
```

You could achieve same result by using `CockroachDB` repl [client](https://www.cockroachlabs.com/docs/stable/install-cockroachdb-linux.html):
```bash
$ cockroach sql --insecure
```

And running create database SQL statement.

## Running service

Service main code is placed under `cmd` directory. You need to build
binary and run service.

```bash
$ go build -o bin/dunder cmd/*.go
$ ./bin/dunder --config_file config.yaml
```

You can also get information about available command line options by
running:

```bash
$ ./bin/dunder -h
Usage of dunder:
      --tls                            Connection uses TLS if true, else plain TCP
      --cert_file string               TLS certificate file path
      --key_file string                TLS key file path
      --port int                       GRPC port
      --debug                          
      --cockroach_db.host string       
      --cockroach_db.should_migrate    
      --cockroach_db.debug             
      --cockroach_db.database string   
      --cockroach_db.user string       
      --config_file string             provide a config file path
  -h, --help                           print this help menu
```

Related flags could be set with config file `config.yaml`:

```yaml
tls: true
cert_file: tls/crt.pem
key_file: tls/key.pem
port: 8081
debug: true

cockroach_db:
  host: localhost
  should_migrate: true
  debug: true
  database: live_database
  user: root
```

## Send some messages

Post some messges:

```bash
$ TOKEN=`echo -n $USER | base64 -w0`
$ curl -d '{"text": "some text", "hashtags":["tag1", "tag2"]}' -H"Authorization: Bearer ${TOKEN}" https://localhost:8081/message
$ curl -d '{"text": "some other text", "hashtags":["tag3", "tag4"]}' -H"Authorization: Bearer ${TOKEN}" https://localhost:8081/message
```

User header is required and users are dynamically created as Dunder don't provide yet
any endpoints for user management.

## Get messages with filtering

Get some messages using eg. tag query:

```bash
$ curl https://localhost:8081/message?hashtag=tag1
```

Filter options:
```text
- from_date - from date range
- to_date - to date range
- limit - response limit (default 100)
- cursor - pagination cursor
- user_name - filter by user name
- hashtag - filter by hashtag
```

## Trends

Trends provides at smallest minute granularity statistics of messages occurrence with option
to filter with `hashtag`. Aggregation option would accept [time.Duration](https://golang.org/pkg/time/#ParseDuration) format and
it's minimal value is `1m`.

```bash
$ curl "https://localhost:8081/trend?to_date=2019-09-23&aggregation=1m&from_date=2019-09-22&hashtag=dummy3"
```

Trends options:
```text
- from_date - from date range
- to_date - to date range
- hashtag - filter by hashtag
- aggregation - aggregation period
```

# Further development

This section describes some further development steps to release Dunder to public.

## Deployment

Standard method is to build helm chart and scale service using Kubernetes cluster.
Some reference implementation could be look up [here](https://github.com/jozuenoon/message_bus/tree/master/deployment).
CockroachDB could be also easily setup and maintained in K8S [cluster](https://github.com/helm/charts/tree/master/stable/cockroachdb).
Usually it's good choice to use `ingress-nginx` controller with TLS termination.

## Kubernetes and TLS

Simplest approach is to use `LetEncrypt` and `cert-manager` to manage TLS along with DNS resolver. 
Certificates are stored directly in K8S secrets and can be mounted to service or used by ingress 
controller if TLS termination is made on LB level.

## Monitoring and health checks

Kubernetes rolling update and load balancing requires decently working health checks for readiness and liveness
probes. This enable load balancing and rolling update to happen properly with no down times or missed requests.

With [OpenCensus](https://opencensus.io/) it's easy to implement tracing and monitoring layers, to gain insights on usage and performance
of service. By using `OpenCensus` one would have a lot of options for exporting tracing and metrics. For even deeper dive 
if application is deployed on GKE there is very nice tool for [profiling with pprof](https://cloud.google.com/profiler/docs/profiling-go)
so flame charts are directly available on Stackdriver page. Some additional [reference](https://medium.com/google-cloud/continuous-profiling-of-go-programs-96d4416af77b).

## Architecture expansion and performance

Current implementation may need some further optimization around SQL statements to get best
efficiency at scale. Probably some geo partitioning tricks would be useful along with
distributing app servers in different locations around world.

The nature of instant message sharing systems is that user would usually expect to see
some recent messages. Leveraging this would evict old messages to other storage where
they can be searched eg. using only batch jobs, so live database would be efficient in 
querying most recent messages.

Tweaking efficiency even further I would recommend to look at CQRS and Event Sourcing techniques
eg. using SQL database as hard storage (command part) and piping commands down to eg. 
Redis cluster where query part would occur.

## Testing

In current state Dunder is missing a lot of tests. Most critical path repository implementation
have same parts covered (and may be improved). As this repository is just POC it would
require some extra effort to cover all paths. However design of service allows for
easy testing with eg. mockery since service layers depend on interface implementations
rather then on concrete structures.
