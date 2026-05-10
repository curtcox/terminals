ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
LOCAL_BIN := $(ROOT_DIR)/.bin
LOCAL_FLUTTER_BIN := $(ROOT_DIR)/.sdk/flutter/bin
LOCAL_GO_CACHE := $(ROOT_DIR)/.cache/go-build
CLIENT_WEB_PORT ?= 60739
CLIENT_WEB_HOST ?= 0.0.0.0
BUILD_SHA ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
# Prefer explicit JAVA_HOME, then common JDK 17 installs (Homebrew, Android Studio JBR, Linux packages), then macOS java_home.
ANDROID_JAVA_HOME ?= $(shell \
	jh=""; \
	if [ -n "$$JAVA_HOME" ] && [ -x "$$JAVA_HOME/bin/java" ] && "$$JAVA_HOME/bin/java" -version >/dev/null 2>&1; then jh="$$JAVA_HOME"; \
	elif [ -d /opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ]; then jh=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home; \
	elif [ -d /usr/local/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ]; then jh=/usr/local/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home; \
	elif [ -d "$$HOME/Applications/Android Studio.app/Contents/jbr/Contents/Home" ]; then jh="$$HOME/Applications/Android Studio.app/Contents/jbr/Contents/Home"; \
	elif [ -d "/Applications/Android Studio.app/Contents/jbr/Contents/Home" ]; then jh="/Applications/Android Studio.app/Contents/jbr/Contents/Home"; \
	elif [ -d "/Applications/Android Studio Preview.app/Contents/jbr/Contents/Home" ]; then jh="/Applications/Android Studio Preview.app/Contents/jbr/Contents/Home"; \
	elif [ -d /usr/lib/jvm/java-17-openjdk-amd64 ]; then jh=/usr/lib/jvm/java-17-openjdk-amd64; \
	elif [ -d /usr/lib/jvm/java-17-openjdk ]; then jh=/usr/lib/jvm/java-17-openjdk; \
	elif [ -x /usr/libexec/java_home ]; then jh="$$(/usr/libexec/java_home -v 17 2>/dev/null || /usr/libexec/java_home -v 21 2>/dev/null || /usr/libexec/java_home 2>/dev/null || true)"; \
	fi; \
	printf '%s' "$$jh")
export PATH := $(LOCAL_BIN):$(LOCAL_FLUTTER_BIN):$(PATH)

.PHONY: server-build server-test server-test-sandbox server-test-network-probe server-test-network-probe-assert server-lint server-coverage \
	client-build client-build-web client-build-android client-build-ios client-build-linux client-build-windows client-build-macos client-build-all \
	client-test client-lint client-boundary client-boundary-test client-coverage \
	android-client-build android-client-test android-client-lint android-client-compile-android-test android-client-connected-test android-client-boundary android-client-boundary-test \
	web-client-build web-client-test web-client-lint web-client-boundary web-client-proto-check web-client-smoke-test run-web-client \
	proto-lint proto-breaking proto-generate proto-flex-check proto-contract-generate proto-contract-test proto-contract-verify \
	skills-validate development-docs-test server-test-network-probe-test plans-index validation-matrix usecases-index pick-next-work next \
	all-lint all-test all-check ci-local stop-server stop-server-test run-server run-client-web run-web-client \
	run-local run-local-test run-local-smoke-test run-mac mac-e2e-test usecase-validate \
	ui-inspect-test

server-build:
	cd terminal_server && go build ./...

server-test:
	cd terminal_server && go test ./...

server-test-network-probe:
	go run ./scripts/probe_server_test_network.go

server-test-network-probe-assert:
	bash ./scripts/assert-server-test-network-probe.sh

server-test-network-probe-test:
	bash ./scripts/test-server-test-network-probe.sh

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

client-boundary:
	./scripts/check-client-boundary.sh

client-boundary-test:
	./scripts/test-check-client-boundary.sh

client-coverage:
	cd terminal_client && flutter test --coverage

android-client-build:
	@if [ -n "$$ANDROID_SDK_ROOT" ] || [ -n "$$ANDROID_HOME" ] || [ -f android_client/local.properties ]; then \
		if [ -z "$(ANDROID_JAVA_HOME)" ] || [ ! -x "$(ANDROID_JAVA_HOME)/bin/java" ]; then \
			echo "Skipping native Android build: JDK not found (need a working Java 17+). Set JAVA_HOME, install openjdk@17, or rely on Android Studio's bundled JBR under Applications."; \
		else \
			cd android_client && JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew assembleDebug -PTERMINALS_BUILD_SHA=$(BUILD_SHA) -PTERMINALS_BUILD_DATE=$(BUILD_DATE); \
		fi; \
	else \
		echo "Skipping native Android build: Android SDK path is not configured (ANDROID_SDK_ROOT/ANDROID_HOME)."; \
	fi

android-client-test:
	@if [ -n "$$ANDROID_SDK_ROOT" ] || [ -n "$$ANDROID_HOME" ] || [ -f android_client/local.properties ]; then \
		if [ -z "$(ANDROID_JAVA_HOME)" ] || [ ! -x "$(ANDROID_JAVA_HOME)/bin/java" ]; then \
			echo "Skipping native Android tests: JDK not found (need a working Java 17+). Set JAVA_HOME, install openjdk@17, or rely on Android Studio's bundled JBR under Applications."; \
		else \
			cd android_client && JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew testDebugUnitTest; \
		fi; \
	else \
		echo "Skipping native Android tests: Android SDK path is not configured (ANDROID_SDK_ROOT/ANDROID_HOME)."; \
	fi

android-client-lint:
	@if [ -n "$$ANDROID_SDK_ROOT" ] || [ -n "$$ANDROID_HOME" ] || [ -f android_client/local.properties ]; then \
		if [ -z "$(ANDROID_JAVA_HOME)" ] || [ ! -x "$(ANDROID_JAVA_HOME)/bin/java" ]; then \
			echo "Skipping native Android lint: JDK not found (need a working Java 17+). Set JAVA_HOME, install openjdk@17, or rely on Android Studio's bundled JBR under Applications."; \
		else \
			cd android_client && JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew lintDebug; \
		fi; \
	else \
		echo "Skipping native Android lint: Android SDK path is not configured (ANDROID_SDK_ROOT/ANDROID_HOME)."; \
	fi

android-client-compile-android-test:
	@if [ -n "$$ANDROID_SDK_ROOT" ] || [ -n "$$ANDROID_HOME" ] || [ -f android_client/local.properties ]; then \
		if [ -z "$(ANDROID_JAVA_HOME)" ] || [ ! -x "$(ANDROID_JAVA_HOME)/bin/java" ]; then \
			echo "Skipping native Android instrumentation compile: JDK not found (need a working Java 17+). Set JAVA_HOME, install openjdk@17, or rely on Android Studio's bundled JBR under Applications."; \
		else \
			cd android_client && JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew compileDebugAndroidTestKotlin; \
		fi; \
	else \
		echo "Skipping native Android instrumentation compile: Android SDK path is not configured (ANDROID_SDK_ROOT/ANDROID_HOME)."; \
	fi

android-client-connected-test:
	@if [ -n "$$ANDROID_SDK_ROOT" ] || [ -n "$$ANDROID_HOME" ] || [ -f android_client/local.properties ]; then \
		if [ -z "$(ANDROID_JAVA_HOME)" ] || [ ! -x "$(ANDROID_JAVA_HOME)/bin/java" ]; then \
			echo "Skipping native Android connected tests: JDK not found (need a working Java 17+). Set JAVA_HOME, install openjdk@17, or rely on Android Studio's bundled JBR under Applications."; \
		elif ! command -v adb >/dev/null 2>&1; then \
			echo "Skipping native Android connected tests: adb is not available in PATH."; \
		elif ! adb devices | awk 'NR>1 && $$2=="device" {found=1} END {exit found ? 0 : 1}'; then \
			echo "Skipping native Android connected tests: no connected Android device/emulator found."; \
		else \
			cd android_client && JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew connectedDebugAndroidTest; \
		fi; \
	else \
		echo "Skipping native Android connected tests: Android SDK path is not configured (ANDROID_SDK_ROOT/ANDROID_HOME)."; \
	fi

android-client-boundary:
	./scripts/check-android-client-boundary.sh

android-client-boundary-test:
	./scripts/test-android-client-boundary.sh

web-client-build:
	cd web_client && npm run build

web-client-test:
	cd web_client && npm test

web-client-lint:
	cd web_client && npm run lint

web-client-boundary:
	cd web_client && npm run boundary

web-client-proto-check:
	cd web_client && npm run proto:check

web-client-smoke-test:
	./scripts/test-web-client-smoke.sh

proto-lint:
	cd api && buf lint
	cd terminal_server && GOCACHE="$(LOCAL_GO_CACHE)" go test ./internal/transport -run 'TestProtoRoundTrip' -count=1

proto-breaking:
	cd api && buf breaking --against '../.git#branch=main,subdir=api'

proto-generate:
	cd api && buf generate

proto-flex-check:
	python3 ./scripts/check-proto-flex-fields.py --enforce

proto-contract-generate:
	./scripts/proto-contract-generate.sh

proto-contract-test:
	$(MAKE) proto-lint
	$(MAKE) proto-flex-check
	./scripts/proto-contract-test.sh
	cd terminal_server && GOCACHE="$(LOCAL_GO_CACHE)" go test ./internal/protocolcontract
	cd terminal_server && GOCACHE="$(LOCAL_GO_CACHE)" go test ./internal/transport -run 'TestProto|TestGenerated' -count=1
	cd terminal_client && HOME="$(ROOT_DIR)/.home" PUB_CACHE="$(ROOT_DIR)/.home/.pub-cache" dart test/protocol_contract_test.dart

proto-contract-verify:
	./scripts/proto-contract-verify.sh

skills-validate:
	./scripts/validate-skills.sh

development-docs-test:
	./scripts/test-development-environment-docs.sh

plans-index:
	python3 ./scripts/generate-plans-index.py

validation-matrix:
	python3 ./scripts/generate-validation-matrix.py

usecases-index:
	python3 ./scripts/generate-usecases-index.py

pick-next-work:
	@python3 ./scripts/pick-next-work.py

next:
	@python3 ./scripts/next.py

all-lint: server-lint client-lint client-boundary android-client-boundary android-client-lint web-client-lint proto-lint

all-test: server-test client-test client-boundary-test android-client-boundary-test android-client-test android-client-compile-android-test web-client-test

all-check: all-lint all-test proto-breaking proto-contract-test web-client-proto-check client-build-all android-client-build web-client-build development-docs-test usecases-index validation-matrix

ci-local: all-check

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

run-web-client:
	cd web_client && WEB_CLIENT_PORT=$(CLIENT_WEB_PORT) WEB_CLIENT_HOST=$(CLIENT_WEB_HOST) npm run serve

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
