.DEFAULT_GOAL := help
.PHONY: help run build test vet lint check dist clean

help: ## Show the available targets
	@grep -E '^[a-z]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-8s %s\n", $$1, $$2}'

run: ## Run the game (flags: make run ARGS="-lan -silent")
	go run ./cmd/mineparty $(ARGS)

build: ## Build for this machine into bin/
	go build -o bin/mineparty ./cmd/mineparty

test: ## Run the tests (one test: make test RUN=TestName)
	go test -race -run '$(RUN)' ./internal/...

vet: ## Vet for Linux and Windows
	go vet ./...
	GOOS=windows go vet ./...

lint: ## Run golangci-lint (advisory: existing issues make this fail)
	golangci-lint run ./...

check: vet test ## Run vet and the tests

dist: ## Wipe dist/ and build every platform
	scripts/build.sh

clean: ## Remove bin/
	rm -rf bin
