SHELL := /bin/bash

NAME := dunder
DOCKER_REGISTRY ?= jozuenoon

GIT_BRANCH := $(shell git branch | sed -n '/\* /s///p' 2>/dev/null)
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null)


all: bin build_docker push

test:
	go test ./... -v

.PHONY: bin
bin:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(NAME) cmd/*.go

build_docker:
	docker build -f Dockerfile -t $(NAME)\:$(GIT_BRANCH)_$(GIT_COMMIT) .

test_docker: bin
	docker build -f Dockerfile -t $(DOCKER_REGISTRY)/$(NAME):test .

test_docker_push:
	docker push $(DOCKER_REGISTRY)/$(NAME):test

push:
	docker tag $(NAME):$(GIT_BRANCH)_$(GIT_COMMIT) $(DOCKER_REGISTRY)/$(NAME):$(GIT_BRANCH)_$(GIT_COMMIT)
	docker push $(DOCKER_REGISTRY)/$(NAME):$(GIT_BRANCH)_$(GIT_COMMIT)

tls:
	mkcert dunder.io "*.dunder.io" dunder.test localhost 127.0.0.1 ::1 -cert-file ./tls/crt.pem -key-file ./tls/key.pem

create_live_db:
	createdb -p 26257 -h localhost -U root -e live_database

run: bin
	./bin/dunder --config_file config.yaml

new_message:
	curl -d '{"text": "some text", "hashtags":["dummy3"]}' -H"User: ${USER}" https://localhost:8081/message

get_message:
	curl https://localhost:8081/message?ulid=${ulid}

get_tag:
	curl https://localhost:8081/message?hashtag=dummy3

get_trends:
	curl "https://localhost:8081/trend?to_date=2019-09-23&aggregation=1m&from_date=2019-09-22&hashtag=dummy3"