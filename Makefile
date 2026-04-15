ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
LOCAL_BIN := $(ROOT_DIR)/.bin
LOCAL_FLUTTER_BIN := $(ROOT_DIR)/.sdk/flutter/bin
export PATH := $(LOCAL_BIN):$(LOCAL_FLUTTER_BIN):$(PATH)

.PHONY: server-build server-test server-lint server-coverage \
	client-build client-test client-lint client-coverage \
	proto-lint proto-breaking proto-generate \
	all-lint all-test all-check run-server run-client-web \
	run-mac mac-e2e-test

server-build:
	cd terminal_server && go build ./...

server-test:
	cd terminal_server && go test ./...

server-lint:
	cd terminal_server && golangci-lint run ./...

server-coverage:
	cd terminal_server && go test ./... -coverprofile=coverage.out

client-build:
	cd terminal_client && flutter build web

client-test:
	cd terminal_client && flutter test

client-lint:
	cd terminal_client && flutter analyze && dart format --set-exit-if-changed .

client-coverage:
	cd terminal_client && flutter test --coverage

proto-lint:
	cd api && buf lint

proto-breaking:
	cd api && buf breaking --against '../.git#branch=main,subdir=api'

proto-generate:
	cd api && buf generate

all-lint: server-lint client-lint proto-lint

all-test: server-test client-test

all-check: all-lint all-test proto-breaking

run-server:
	cd terminal_server && go run ./cmd/server

run-client-web:
	cd terminal_client && flutter run -d web-server

run-mac:
	./scripts/run-mac.sh

mac-e2e-test:
	./scripts/test-mac-e2e.sh
