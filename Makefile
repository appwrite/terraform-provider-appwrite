default: build

# Pinned rather than @latest. A floating linter silently changes what passes
# review: a release adds a check and every open PR goes red for reasons nobody
# introduced, or drops one and a rule stops being enforced without a commit.
# Note that CI installed the v1 module path, whose @latest is the end-of-life
# v1.64.8 -- unpinned and stale at the same time.
GOLANGCI_LINT_VERSION ?= v2.13.2

BINARY := terraform-provider-appwrite

build:
	go build -o $(BINARY)

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/appwrite/appwrite/0.1.0/$$(go env GOOS)_$$(go env GOARCH)
	cp $(BINARY) ~/.terraform.d/plugins/registry.terraform.io/appwrite/appwrite/0.1.0/$$(go env GOOS)_$$(go env GOARCH)/

# Unit tests get a short timeout on purpose. Nothing here may touch the network,
# and a deadline is the only way to enforce that structurally -- a test that
# reaches for an API either fails fast or hangs until it trips this, and either
# way the author finds out immediately rather than in the nightly.
test:
	go test ./... -count=1 -timeout 60s

acceptance-test:
	TF_ACC=1 go test ./... -v -count=1 $(TESTARGS) -timeout 120m

# Deletes resources an interrupted acceptance run left behind. Needs
# APPWRITE_SWEEP_PROJECT_ID naming the project to sweep; see internal/sweep.
sweep:
	@echo "WARNING: this deletes tf-acc-test-prefixed resources in APPWRITE_SWEEP_PROJECT_ID."
	go test ./internal/sweep/ -v -sweep=appwrite -timeout 60m $(SWEEPARGS)

# The sweepers delete infrastructure with no plan in front of them, so where
# that code is linked is a safety property rather than a detail. These two
# targets assert it from both sides: present in what the tests run, absent from
# what users install. They fail the moment a sweeper is moved into non-test code
# the provider imports.
sweeper-linked:
	@go test -c -o /tmp/appwrite-sweep.test ./internal/sweep/
	@count=$$(go tool nm /tmp/appwrite-sweep.test 2>/dev/null | grep -c 'internal/sweep\.sweep' || true); \
	rm -f /tmp/appwrite-sweep.test; \
	if [ "$$count" -eq 0 ]; then \
		echo "FAIL: no sweeper symbols in the test binary; the sweepers are not reachable from tests"; \
		exit 1; \
	fi; \
	echo "OK: $$count sweeper symbols linked into the test binary."

sweeper-unlinked:
	@echo "==> Checking that no shipped package imports internal/sweep..."
	@if go list -deps . | grep -q 'internal/sweep$$'; then \
		echo "FAIL: internal/sweep is in the release binary's import graph:"; \
		go list -deps . | grep 'internal/sweep$$'; \
		echo "Sweeper code deletes infrastructure without a plan and must not ship to users."; \
		echo "Keep it reachable only from _test.go files."; \
		exit 1; \
	fi
	@echo "OK: internal/sweep is not reachable from the provider binary."

test-compile:
	go test -c ./... -o /dev/null

vet:
	go vet ./...

fmt:
	gofmt -s -w .

fmt-check:
	@gofmt_output=$$(gofmt -l .); \
	if [ -n "$$gofmt_output" ]; then \
		echo "Files not formatted:"; \
		echo "$$gofmt_output"; \
		exit 1; \
	fi

golangci-lint:
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run

# `lint` was vet + fmt-check while CI additionally ran golangci-lint and the
# docs check, so a clean local run could still fail the PR. These now agree.
lint: vet fmt-check golangci-lint

# Guards that need no Go toolchain opinion. Each is a shell script so CI and a
# developer run exactly the same thing.
checks:
	@scripts/checks/no-direct-resource-test.sh

# go.mod and go.sum drifting from what the code imports is a supply-chain
# problem rather than a tidiness one: it decides what a release builds from.
depscheck:
	@echo "==> Checking source code with go mod tidy..."
	@go mod tidy
	@git diff --exit-code -- go.mod go.sum || \
		(echo; echo "Unexpected difference in go.mod/go.sum. Run 'go mod tidy' and commit."; exit 1)

docs-check: docs
	@git diff --exit-code -- docs/ || \
		(echo; echo "Documentation is out of date. Run 'make docs' and commit the changes."; exit 1)

# What CI runs, in one target, so it can be reproduced locally before pushing.
ci: lint checks depscheck test sweeper-linked sweeper-unlinked docs-check

clean:
	rm -f $(BINARY)

docs:
	go generate ./...

.PHONY: build install test acceptance-test sweep sweeper-linked sweeper-unlinked \
	test-compile vet fmt fmt-check golangci-lint lint checks depscheck docs-check ci clean docs
