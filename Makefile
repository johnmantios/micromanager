BUILD_DIR ?= build

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

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	go run ./cmd/api

## build-dir: creates a build directory based on the argument given
build-dir:
	@echo "${BUILD_DIR}"
	@mkdir -p "${BUILD_DIR}"
# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: tidy and vendor dependencies and format, vet and test all code
.PHONY: audit
audit: vendor
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

## vendor: tidy and vendor dependencies
.PHONY: vendor
vendor:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor

# ==================================================================================== #
# BUILD
# ==================================================================================== #



# ==================================================================================== #
# BUILD
# ==================================================================================== #

## test-unit: run unit tests
.PHONY: test-unit
test-unit:
	printf "$(OK_COLOR)==> Unit tests$(NO_COLOR)\n"
	go test -tags unit ./... -v


## test-unit-coverage: run unit tests, create build directory and save inside the unit test coverage
.PHONY: test-unit-coverage
test-unit-coverage: build-dir
	go install github.com/jstemmer/go-junit-report/v2@latest
	printf "==> Unit tests with code coverage\n"
	mkdir -p ${BUILD_DIR}
	go test -v -tags unit -covermode=atomic -coverprofile=$(BUILD_DIR)/unit-coverage.txt ./... -v
