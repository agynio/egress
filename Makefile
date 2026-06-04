SHELL := /bin/bash

.PHONY: proto build test lint fmt clean

proto:
	buf generate buf.build/agynio/api --path agynio/api/egress/v1
	buf generate buf.build/agynio/api --path agynio/api/authorization/v1
	buf generate buf.build/agynio/api --path agynio/api/secrets/v1
	buf generate buf.build/agynio/api --path agynio/api/notifications/v1
	buf generate buf.build/agynio/api --path agynio/api/identity/v1
	buf generate https://github.com/agynio/api.git#branch=noa/issue-60-api --path proto/agynio/api/ziti_management/v1/ziti_management.proto

build:
	go build ./...

test:
	go test ./...

lint:
	go vet ./...

fmt:
	gofmt -w $$(find . -type f -name '*.go')

clean:
	rm -rf .gen
