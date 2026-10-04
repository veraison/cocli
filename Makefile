.DEFAULT_GOAL := help

SHELL := /bin/bash

GO111MODULE := on

GOPKG += github.com/veraison/cocli/cmd

MOCKGEN := $(shell go env GOPATH)/bin/mockgen
INTERFACES := cmd/isubmitter.go
MOCKPKG := mocks

GOLINT ?= golangci-lint

GOLINT_ARGS ?= run

.PHONY: lint
lint: _mocks; $(GOLINT) $(GOLINT_ARGS)

ifeq ($(MAKECMDGOALS),test)
GOTEST_ARGS ?= -v -race $(GOPKG)
else
  ifeq ($(MAKECMDGOALS),test-cover)
  GOTEST_ARGS ?= -short -cover $(GOPKG)
  endif
endif

COVER_THRESHOLD := $(shell sed -n "s/^ *min-coverage: '\(.*\)'/≥\1%/p" .github/workflows/ci-go-cover.yml)

define MOCK_template
cmd/mocks/$(notdir $(1)): $(1)
	$$(MOCKGEN) -source=$$< -destination=$$@ -package=$$(MOCKPKG)
endef

$(foreach m,$(INTERFACES),$(eval $(call MOCK_template,$(m))))
MOCK_FILES := $(foreach m,$(INTERFACES),$(join cmd/mocks/,$(notdir $(m))))
CLEANFILES := $(MOCK_FILES)

_mocks: $(MOCK_FILES)
.PHONY: _mocks

.PHONY: test test-cover
test test-cover: _mocks; go test $(GOTEST_ARGS)

realtest: _mocks; go test $(GOTEST_ARGS)
.PHONY: realtest

CLEANFILES += cmd/output.cbor

.PHONY: clean
clean: ; $(RM) $(CLEANFILES)

presubmit:
	@echo
	@echo ">>> Check that the reported coverage figures are $(COVER_THRESHOLD)"
	@echo
	$(MAKE) test-cover
	@echo
	@echo ">>> Fix any lint error"
	@echo
	$(MAKE) lint

.PHONY: licenses
licenses: ; @./scripts/licenses.sh

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  * test:       run unit tests for $(GOPKG)"
	@echo "  * test-cover: run unit tests and measure coverage for $(GOPKG)"
	@echo "  * lint:       lint sources using .golangci.yml"
	@echo "  * presubmit:  check you are ready to push your local branch to remote"
	@echo "  * help:       print this menu"
	@echo "  * licenses:   check licenses of dependent packages"
	@echo "  * clean:      remove auto-generated files"
