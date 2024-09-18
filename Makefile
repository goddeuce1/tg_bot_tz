################################################################################
#
# Makefile
#
################################################################################

.PHONY: run
run:
	@go run cmd/main.go

.PHONY: build
build:
	@go build cmd/main.go

.PHONY: lint
lint:
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run --print-issued-lines=false

.PHONY: test
test:
	@go test -v ./...