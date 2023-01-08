# Makefile
REPORTS_DIR=reports
BUILD_DIR=bin/
COVERFILE=$(REPORTS_DIR)/coverage.out
TEST_DIRS=$(shell echo ./...)

.DEFAULT_GOAL: test

.PHONY: dep
dep: ## fetch go dependencies
	@go mod download

.PHONY: clean
clean: ## clean repo
	find . -type f -iname "*~" -print0 | xargs -0r rm -f
	rm -rfv $(REPORTS_DIR) *.out *~ *.bak $(BUILD_DIR)

.PHONY: help
help: Makefile
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: update-assets
update-assets: ## update the embedded assets
	wget -O assets/Leap_Second.dat "https://hpiers.obspm.fr/iers/bul/bulc/Leap_Second.dat"

.PHONY: test
test: ## run the unit tests
	go test -timeout=5s -v $(TEST_DIRS)

.PHONY: race
race: ## run the unit tests and check for race conditions
	go test -race -timeout=5s -v $(TEST_DIRS)

.PHONY: benchmark
benchmark: ## run any benchmark tests
	go test -bench=. -run=^# $(TEST_DIRS)

.PHONY: vet
vet: ## run go vet
	go vet -all ./...

.PHONY: fmt
fmt: ## format the code using go fmt
	go fmt ./...

.PHONY: lintci
lintci:	## run the linters and security checks
	golangci-lint run ./...

.PHONY:	coverage
coverage:	## get the code coverage
	@mkdir -vp $(REPORTS_DIR)
	go test -v -covermode=count -coverprofile=$(COVERFILE)
