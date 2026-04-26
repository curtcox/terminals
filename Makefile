ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
LOCAL_BIN := $(ROOT_DIR)/.bin
LOCAL_FLUTTER_BIN := $(ROOT_DIR)/.sdk/flutter/bin
CLIENT_WEB_PORT ?= 60739
CLIENT_WEB_HOST ?= 0.0.0.0
BUILD_SHA ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
export PATH := $(LOCAL_BIN):$(LOCAL_FLUTTER_BIN):$(PATH)

.PHONY: server-build server-test server-test-sandbox server-test-network-probe server-lint server-coverage \
	client-build client-build-web client-build-android client-build-ios client-build-linux client-build-windows client-build-macos client-build-all \
	client-test client-lint client-coverage \
	proto-lint proto-breaking proto-generate \
	skills-validate development-docs-test plans-index pick-next-work \
	all-lint all-test all-check stop-server stop-server-test run-server run-client-web \
	run-local run-local-test run-local-smoke-test run-mac mac-e2e-test usecase-validate \
	ui-inspect-test

server-build:
	cd terminal_server && go build ./...

server-test:
	cd terminal_server && go test ./...

server-test-network-probe:
	go run ./scripts/probe_server_test_network.go

server-test-sandbox:
	./scripts/server-test-sandbox.sh

server-lint:
	cd terminal_server && golangci-lint run ./...

server-coverage:
	cd terminal_server && go test ./... -coverprofile=coverage.out

client-build:
	$(MAKE) client-build-web

client-build-web:
	cd terminal_client && flutter build web --no-wasm-dry-run --dart-define=TERMINALS_BUILD_SHA=$(BUILD_SHA) --dart-define=TERMINALS_BUILD_DATE=$(BUILD_DATE)

client-build-android:
	@if [ -n "$$ANDROID_SDK_ROOT" ] || [ -n "$$ANDROID_HOME" ]; then \
		cd terminal_client && flutter build apk; \
	else \
		echo "Skipping Android build: Android SDK path is not configured (ANDROID_SDK_ROOT/ANDROID_HOME)."; \
	fi

client-build-ios:
	@if [ "$$(uname -s)" = "Darwin" ] && xcodebuild -version >/dev/null 2>&1; then \
		tmp="$$(mktemp)"; \
		if cd terminal_client && flutter build ios --no-codesign >"$$tmp" 2>&1; then \
			cat "$$tmp"; \
			rm -f "$$tmp"; \
		else \
			if grep -Eq "iOS [0-9]+\\.[0-9]+ is not installed|Unable to find a destination matching the provided destination specifier" "$$tmp"; then \
				echo "Skipping iOS build: required iOS platform components are not installed in Xcode."; \
				tail -n 20 "$$tmp"; \
				rm -f "$$tmp"; \
			else \
				cat "$$tmp"; \
				rm -f "$$tmp"; \
				exit 1; \
			fi; \
		fi; \
	else \
		echo "Skipping iOS build: requires macOS with Xcode command line tools."; \
	fi

client-build-linux:
	@if [ "$$(uname -s)" = "Linux" ]; then \
		cd terminal_client && flutter build linux; \
	else \
		echo "Skipping Linux build: only supported on Linux hosts."; \
	fi

client-build-windows:
	@if [ "$$(uname -s)" = "MINGW64_NT" ] || [ "$$(uname -s)" = "MSYS_NT" ] || [ "$$(uname -s)" = "CYGWIN_NT" ]; then \
		cd terminal_client && flutter build windows; \
	else \
		echo "Skipping Windows build: only supported on Windows hosts."; \
	fi

client-build-macos:
	@if [ "$$(uname -s)" = "Darwin" ] && xcodebuild -version >/dev/null 2>&1; then \
		cd terminal_client && flutter build macos; \
	else \
		echo "Skipping macOS build: requires macOS with Xcode command line tools."; \
	fi

client-build-all: client-build-web client-build-android client-build-ios client-build-linux client-build-windows client-build-macos

client-test:
	cd terminal_client && flutter test

client-lint:
	cd terminal_client && flutter analyze && dart format --set-exit-if-changed .

client-coverage:
	cd terminal_client && flutter test --coverage

proto-lint:
	cd api && buf lint
	cd terminal_server && go test ./internal/transport -run 'TestProtoRoundTrip' -count=1

proto-breaking:
	cd api && buf breaking --against '../.git#branch=main,subdir=api'

proto-generate:
	cd api && buf generate

skills-validate:
	./scripts/validate-skills.sh

development-docs-test:
	./scripts/test-development-environment-docs.sh

plans-index:
	python3 ./scripts/generate-plans-index.py

pick-next-work:
	@python3 ./scripts/pick-next-work.py

all-lint: server-lint client-lint proto-lint

all-test: server-test client-test

all-check: all-lint all-test proto-breaking client-build-all development-docs-test

stop-server:
	./scripts/stop-server.sh

stop-server-test:
	./scripts/test-stop-server-target.sh
	./scripts/test-stop-server-safety.sh

run-server:
	cd terminal_server && \
		TERMINALS_GRPC_HOST=0.0.0.0 \
		TERMINALS_CONTROL_WS_HOST=0.0.0.0 \
		TERMINALS_CONTROL_TCP_HOST=0.0.0.0 \
		TERMINALS_CONTROL_HTTP_HOST=0.0.0.0 \
		TERMINALS_ADMIN_HTTP_HOST=0.0.0.0 \
		TERMINALS_BUILD_SHA=$(BUILD_SHA) \
		TERMINALS_BUILD_DATE=$(BUILD_DATE) \
		go run ./cmd/server

run-client-web:
	cd terminal_client && flutter build web --no-wasm-dry-run --pwa-strategy=none --dart-define=TERMINALS_BUILD_SHA=$(BUILD_SHA) --dart-define=TERMINALS_BUILD_DATE=$(BUILD_DATE)
	cd terminal_client && python3 -m http.server $(CLIENT_WEB_PORT) --bind $(CLIENT_WEB_HOST) --directory build/web

run-local:
	./scripts/run-local.sh

run-local-test:
	./scripts/test-run-local.sh

run-local-smoke-test:
	./scripts/test-run-local.sh --real-smoke

run-mac:
	./scripts/run-mac.sh

mac-e2e-test:
	./scripts/test-mac-e2e.sh

usecase-validate:
	INFO="$(INFO)" ./scripts/usecase-validate.sh "$(USECASE)"

ui-inspect-test:
	./scripts/test-ui-inspect-run.sh
