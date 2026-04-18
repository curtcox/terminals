ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
LOCAL_BIN := $(ROOT_DIR)/.bin
LOCAL_FLUTTER_BIN := $(ROOT_DIR)/.sdk/flutter/bin
CLIENT_WEB_PORT ?= 60739
CLIENT_WEB_HOST ?= 0.0.0.0
export PATH := $(LOCAL_BIN):$(LOCAL_FLUTTER_BIN):$(PATH)

.PHONY: server-build server-test server-lint server-coverage \
	client-build client-build-web client-build-android client-build-ios client-build-linux client-build-windows client-build-macos client-build-all \
	client-test client-lint client-coverage \
	proto-lint proto-breaking proto-generate \
	all-lint all-test all-check run-server run-client-web \
	run-local run-local-test run-local-smoke-test run-mac mac-e2e-test usecase-validate

server-build:
	cd terminal_server && go build ./...

server-test:
	cd terminal_server && go test ./...

server-lint:
	cd terminal_server && golangci-lint run ./...

server-coverage:
	cd terminal_server && go test ./... -coverprofile=coverage.out

client-build:
	$(MAKE) client-build-web

client-build-web:
	cd terminal_client && flutter build web --no-wasm-dry-run

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

proto-breaking:
	cd api && buf breaking --against '../.git#branch=main,subdir=api'

proto-generate:
	cd api && buf generate

all-lint: server-lint client-lint proto-lint

all-test: server-test client-test

all-check: all-lint all-test proto-breaking client-build-all

run-server:
	cd terminal_server && go run ./cmd/server

run-client-web:
	cd terminal_client && flutter run -d web-server --web-port=$(CLIENT_WEB_PORT) --web-hostname=$(CLIENT_WEB_HOST)

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
	./scripts/usecase-validate.sh "$(USECASE)"
