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
	android-client-build android-client-test android-client-lint android-client-deps android-client-dependency-check android-client-compile-android-test android-client-connected-test android-client-gradle-stop android-client-boundary android-client-boundary-test \
	web-client-build web-client-test web-client-lint web-client-boundary web-client-proto-check web-client-smoke-test run-web-client \
	proto-lint proto-breaking proto-generate proto-flex-check proto-contract-generate proto-contract-test proto-contract-verify \
	skills-validate development-docs-test server-test-network-probe-test plans-index validation-matrix usecases-index usecases-site usecases-site-check usecase-wiring-audit pick-next-work next bug-resolved-check bug-resolve-test \
	all-lint all-test all-check ci-local stop-server stop-server-test run-server run-client-web run-web-client \
	run-local run-local-test run-local-smoke-test run-mac mac-e2e-test usecase-validate \
	ui-inspect-test format help help-all

server-build: ## Build the Go server
	cd terminal_server && go build ./...

server-test: ## Run Go server tests
	cd terminal_server && go test ./...

server-test-network-probe:
	go run ./scripts/probe_server_test_network.go

server-test-network-probe-assert:
	bash ./scripts/assert-server-test-network-probe.sh

server-test-network-probe-test:
	bash ./scripts/test-server-test-network-probe.sh

server-test-sandbox:
	./scripts/server-test-sandbox.sh

server-lint: ## Lint Go server (golangci-lint)
	cd terminal_server && golangci-lint run ./...

server-coverage:
	cd terminal_server && go test ./... -coverprofile=coverage.out

client-build: ## Build Flutter client (web)
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
			if grep -Eq "iOS [0-9]+\\.[0-9]+ is not installed|iOS [0-9]+\\.[0-9]+ Platform Not Installed|Unable to find a destination matching the provided destination specifier" "$$tmp"; then \
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

client-test: ## Run Flutter client tests
	cd terminal_client && flutter test

client-lint: ## Lint Flutter client (analyze + dart format check)
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

# Stops Gradle daemons (and JUnit worker JVMs) for this user. Prevents abandoned test runs from pinning CPU; safe when no daemons run.
android-client-gradle-stop:
	@cd android_client && \
	if [ -n "$(ANDROID_JAVA_HOME)" ] && [ -x "$(ANDROID_JAVA_HOME)/bin/java" ]; then \
		JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew --stop || true; \
	else \
		./gradlew --stop || true; \
	fi

android-client-test:
	@if [ -n "$$ANDROID_SDK_ROOT" ] || [ -n "$$ANDROID_HOME" ] || [ -f android_client/local.properties ]; then \
		if [ -z "$(ANDROID_JAVA_HOME)" ] || [ ! -x "$(ANDROID_JAVA_HOME)/bin/java" ]; then \
			echo "Skipping native Android tests: JDK not found (need a working Java 17+). Set JAVA_HOME, install openjdk@17, or rely on Android Studio's bundled JBR under Applications."; \
		else \
			python3 scripts/android-client-stop-stuck-workers.py --repo-root "$(CURDIR)" --android-root "$(CURDIR)/android_client" --java-home "$(ANDROID_JAVA_HOME)"; \
			cd android_client && JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew testDebugUnitTest; \
			ec=$$?; \
			JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew --stop || true; \
			python3 "$(CURDIR)/scripts/android-client-stop-stuck-workers.py" --repo-root "$(CURDIR)" --android-root "$(CURDIR)/android_client" --java-home "$(ANDROID_JAVA_HOME)" || true; \
			exit $$ec; \
		fi; \
	else \
		echo "Skipping native Android tests: Android SDK path is not configured (ANDROID_SDK_ROOT/ANDROID_HOME)."; \
	fi

android-client-lint:
	@if [ -n "$$ANDROID_SDK_ROOT" ] || [ -n "$$ANDROID_HOME" ] || [ -f android_client/local.properties ]; then \
		if [ -z "$(ANDROID_JAVA_HOME)" ] || [ ! -x "$(ANDROID_JAVA_HOME)/bin/java" ]; then \
			echo "Skipping native Android lint: JDK not found (need a working Java 17+). Set JAVA_HOME, install openjdk@17, or rely on Android Studio's bundled JBR under Applications."; \
		else \
			cd android_client && JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew lintDebug detektMain; \
		fi; \
	else \
		echo "Skipping native Android lint: Android SDK path is not configured (ANDROID_SDK_ROOT/ANDROID_HOME)."; \
	fi

# Reports dependency updates (does not modify build files). Optional; not part of all-lint.
android-client-deps:
	@if [ -n "$$ANDROID_SDK_ROOT" ] || [ -n "$$ANDROID_HOME" ] || [ -f android_client/local.properties ]; then \
		if [ -z "$(ANDROID_JAVA_HOME)" ] || [ ! -x "$(ANDROID_JAVA_HOME)/bin/java" ]; then \
			echo "Skipping Android dependency report: JDK not found (need a working Java 17+)."; \
		else \
			cd android_client && JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew dependencyUpdates --no-daemon; \
		fi; \
	else \
		echo "Skipping Android dependency report: Android SDK path is not configured (ANDROID_SDK_ROOT/ANDROID_HOME)."; \
	fi

# OWASP Dependency-Check (CVEs on runtime classpaths). Optional locally; set NVD_API_KEY for faster, more reliable NVD access.
android-client-dependency-check:
	@if [ -n "$$ANDROID_SDK_ROOT" ] || [ -n "$$ANDROID_HOME" ] || [ -f android_client/local.properties ]; then \
		if [ -z "$(ANDROID_JAVA_HOME)" ] || [ ! -x "$(ANDROID_JAVA_HOME)/bin/java" ]; then \
			echo "Skipping Android dependency check: JDK not found (need a working Java 17+)."; \
		else \
			cd android_client && JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew :app:dependencyCheckAnalyze --no-daemon; \
		fi; \
	else \
		echo "Skipping Android dependency check: Android SDK path is not configured (ANDROID_SDK_ROOT/ANDROID_HOME)."; \
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
			python3 scripts/android-client-stop-stuck-workers.py --repo-root "$(CURDIR)" --android-root "$(CURDIR)/android_client" --java-home "$(ANDROID_JAVA_HOME)"; \
			cd android_client && JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew connectedDebugAndroidTest; \
			ec=$$?; \
			JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew --stop || true; \
			python3 "$(CURDIR)/scripts/android-client-stop-stuck-workers.py" --repo-root "$(CURDIR)" --android-root "$(CURDIR)/android_client" --java-home "$(ANDROID_JAVA_HOME)" || true; \
			exit $$ec; \
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

proto-lint: ## Lint protobuf definitions (buf lint + round-trip test)
	cd api && buf lint
	cd terminal_server && GOCACHE="$(LOCAL_GO_CACHE)" go test ./internal/transport -run 'TestProtoRoundTrip' -count=1

proto-breaking: ## Check for breaking protobuf changes against main
	cd api && buf breaking --against '../.git#branch=main,subdir=api'

proto-generate: ## Regenerate Go + Dart code from .proto files
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

usecases-site:
	python3 ./scripts/build-usecase-site.py

usecases-site-with-results:
	python3 ./scripts/build-usecase-site.py --include-results --include-validation-runs --include-bugs

usecases-site-check:
	python3 ./scripts/test_build_usecase_site.py
	python3 ./scripts/build-usecase-site.py --check

usecase-wiring-audit:
	bash terminal_server/internal/scenario/audit/verify_terminal_ui_usecases.sh
	python3 scripts/audit-usecase-wiring.py

ci-status: ## Probe CI gates and write scripts/ci-status.json
	@scripts/check-ci-gates.sh; true

pick-next-work:
	@python3 ./scripts/pick-next-work.py

next: ## Print the recommended next work item (reads ci-status + plan frontmatter)
	@python3 ./scripts/next.py

quality-check: ## Flag oversized files and other quality checks
	@python3 ./scripts/find-oversized-files.py --check

oversized-toc-audit:
	@python3 ./scripts/find-oversized-files.py --json \
	  | python3 -c "\
import json, sys, pathlib; \
data = json.load(sys.stdin); \
missing = [f for f in data if int(f.get('lines',0)) > 800 \
  and '// CONTENTS:' not in pathlib.Path(f['path']).read_text()]; \
[print(f['path'], 'missing // CONTENTS: block') for f in missing]; \
sys.exit(1 if missing else 0)"

bug-resolved-check:
	@python3 ./scripts/check-resolved-bugs.py

bug-resolve-test:
	@python3 ./scripts/test_bug_resolve.py

all-lint: server-lint client-lint client-boundary android-client-boundary android-client-lint web-client-lint proto-lint

all-test: server-test client-test client-boundary-test android-client-boundary-test android-client-test android-client-compile-android-test web-client-test

all-check: quality-check bug-resolved-check bug-resolve-test all-lint all-test proto-breaking proto-contract-test web-client-proto-check client-build-all android-client-build web-client-build development-docs-test usecases-index usecases-site-check usecase-wiring-audit validation-matrix

# Fast subset: lint + cheap checks only.  Targets the inner-loop signal an
# agent typically needs after editing one or two files — no full builds, no
# Flutter/Android build, no integration tests.
check-fast: quality-check all-lint proto-breaking usecase-wiring-audit ## Lint + cheap checks only (no builds, no integration tests)

# Same prerequisites as all-check, but with -k so a single failure does not
# mask downstream gates.  Use when you want every failure surfaced in one
# pass (e.g. before opening a PR).
check-all-keep-going: ## Full gate with -k so all failures surface in one pass
	@$(MAKE) -k all-check

ci-local: all-check

stop-server: ## Kill running server process(es)
	./scripts/stop-server.sh

stop-server-test:
	./scripts/test-stop-server-target.sh
	./scripts/test-stop-server-safety.sh

run-server: ## Start the Go server (ports 50051–50056)
	cd terminal_server && \
		TERMINALS_GRPC_HOST=0.0.0.0 \
		TERMINALS_CONTROL_WS_HOST=0.0.0.0 \
		TERMINALS_CONTROL_TCP_HOST=0.0.0.0 \
		TERMINALS_CONTROL_HTTP_HOST=0.0.0.0 \
		TERMINALS_ADMIN_HTTP_HOST=0.0.0.0 \
		TERMINALS_BUILD_SHA=$(BUILD_SHA) \
		TERMINALS_BUILD_DATE=$(BUILD_DATE) \
		go run ./cmd/server

run-client-web: ## Build Flutter web client and serve on port 8080
	cd terminal_client && flutter build web --no-wasm-dry-run --pwa-strategy=none --dart-define=TERMINALS_BUILD_SHA=$(BUILD_SHA) --dart-define=TERMINALS_BUILD_DATE=$(BUILD_DATE)
	cd terminal_client && python3 -m http.server $(CLIENT_WEB_PORT) --bind $(CLIENT_WEB_HOST) --directory build/web

run-web-client:
	cd web_client && WEB_CLIENT_PORT=$(CLIENT_WEB_PORT) WEB_CLIENT_HOST=$(CLIENT_WEB_HOST) npm run serve

run-local: ## Start server + Flutter web client (writes .tmp/run-local-*.log)
	./scripts/run-local.sh

run-local-test:
	./scripts/test-run-local.sh

run-local-smoke-test:
	./scripts/test-run-local.sh --real-smoke

run-mac:
	./scripts/run-mac.sh

mac-e2e-test:
	./scripts/test-mac-e2e.sh

usecase-validate: ## Run automated validation for a use-case ID  (e.g. make usecase-validate USECASE=C1)
	INFO="$(INFO)" ./scripts/usecase-validate.sh "$(USECASE)"

ui-inspect-test:
	./scripts/test-ui-inspect-run.sh

format: ## Auto-format all code (Go: gofumpt, Flutter: dart format)
	cd terminal_server && gofumpt -w .
	cd terminal_client && dart format .

help: ## Show annotated targets; all targets listed via make help-all
	@printf "Annotated targets (## description):\n\n"
	@grep -E '^[a-zA-Z_-]+:.*## ' Makefile \
	  | sed 's/:.*## /\t/' \
	  | awk -F'\t' '{ printf "  %-35s %s\n", $$1, $$2 }' \
	  | sort
	@printf "\nRun 'make help-all' for the full target list.\n"

help-all: ## List every target name (unsorted, no descriptions)
	@grep -E '^[a-zA-Z_-]+:' Makefile \
	  | grep -v '^\.PHONY' \
	  | sed 's/:.*$$//' \
	  | sort \
	  | pr -3 -t -w 80
