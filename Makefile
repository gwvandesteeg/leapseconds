# Makefile

.DEFAULT_GOAL: update-assets

.PHONY: help
help: Makefile
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: update-assets
update-assets: ## update the embedded assets
	wget -O assets/Leap_Second.dat "https://hpiers.obspm.fr/iers/bul/bulc/Leap_Second.dat"

.PHONY: test
test: ## run the unit tests
	go test -timeout=5s -v ./...

.PHONY: benchmark
benchmark: ## run any benchmark tests
	go test -bench=. -run=^# ./...

.PHONY: vet
vet: ## run go vet
	go vet -all ./...

.PHONY: fmt
fmt: ## format the code using go fmt
	go fmt ./...

.PHONY: lintci
lintci:
	golangci-lint run ./...
