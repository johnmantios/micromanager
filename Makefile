REPO ?=github.com/johnmantios/micromanager
BUILD_DIR ?= $(CURDIR)/out
BINARY_NAME?=micromanager
BINARY_SRC=${REPO}/cmd
GO_LINKER_FLAGS=-ldflags "-s"
TESTCONTAINERS_RYUK_DISABLED ?= true

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

.PHONY: build
build:
	@printf "$(OK_COLOR)==> Building binary$(NO_COLOR)\n"
	@go build -o ${BUILD_DIR}/${BINARY_NAME} ${GO_LINKER_FLAGS} ${BINARY_SRC}

# ==================================================================================== #
# TESTING
# ==================================================================================== #

## test-unit: run the table-driven unit tests (no external services required)
.PHONY: test-unit
test-unit:
	@printf "$(OK_COLOR)==> Running unit tests$(NO_COLOR)\n"
	@go test -tags unit -count=1 -race ./...

## test-integration: run the table-driven integration tests backed by testcontainers
.PHONY: test-integration
test-integration:
	@printf "$(OK_COLOR)==> Running integration tests$(NO_COLOR)\n"
	@TESTCONTAINERS_RYUK_DISABLED=$(TESTCONTAINERS_RYUK_DISABLED) go test -tags integration -count=1 -timeout 15m ./...

## test-e2e: run the table-driven end-to-end API tests backed by testcontainers
.PHONY: test-e2e
test-e2e:
	@printf "$(OK_COLOR)==> Running end-to-end tests$(NO_COLOR)\n"
	@TESTCONTAINERS_RYUK_DISABLED=$(TESTCONTAINERS_RYUK_DISABLED) go test -tags e2e -count=1 -timeout 15m ./...

## test: run the unit, integration and end-to-end test suites
.PHONY: test
test: test-unit test-integration test-e2e
