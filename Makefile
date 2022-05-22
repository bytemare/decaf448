PACKAGES    := $(shell go list ./...)
COMMIT      := $(shell git rev-parse HEAD)

GH_ACTIONS = .github/workflows

.PHONY: fmt
fmt:
	@echo "Formatting ..."
	@go mod tidy
	@golines -m 120 -t 4 -w .
	@gofumpt -w -extra .
	@gci write --Section Standard --Section Default --Section "Prefix($(shell go list -m))" .

.PHONY: license
license:
	@echo "Checking License headers ..."
	@if addlicense -check -v -f .github/licence-header.tmpl *; then echo "License headers OK"; else return 1; fi;

.PHONY: test
test:
	@echo "Running all tests ..."
	@go test -v ./...

.PHONY: vectors
vectors:
	@echo "Testing vectors ..."
	@go test -v decaf448_hash_test.go

.PHONY: cover
cover:
	@echo "Testing with coverage ..."
	@go test -v -race -covermode=atomic -coverpkg=./... -coverprofile=./coverage.out ./...
