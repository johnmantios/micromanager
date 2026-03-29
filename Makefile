REPO ?=github.com/johnmantios/micromanager
BUILD_DIR ?= $(CURDIR)/out
BINARY_NAME?=micromanager
BINARY_SRC=${REPO}/cmd
GO_LINKER_FLAGS=-ldflags "-s"

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