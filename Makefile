default: build

build:
	go build -o terraform-provider-appwrite

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/appwrite/appwrite/0.1.0/$$(go env GOOS)_$$(go env GOARCH)
	cp terraform-provider-appwrite ~/.terraform.d/plugins/registry.terraform.io/appwrite/appwrite/0.1.0/$$(go env GOOS)_$$(go env GOARCH)/

test:
	go test ./... -v -count=1 -timeout 10m

acceptance-test:
	TF_ACC=$${TF_ACC:-1} go test ./... -v -count=1 $(TESTARGS) -timeout 120m

sweep:
	@echo "WARNING: This will destroy infrastructure. Use only in development."
	go test ./... -v -sweep=all $(SWEEPARGS) -timeout 60m

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

lint: vet fmt-check

clean:
	rm -f terraform-provider-appwrite

docs:
	go generate ./...

# Self-hosted Appwrite for testing
appwrite-up:
	./testing/bootstrap.sh --up

appwrite-down:
	./testing/bootstrap.sh --down

appwrite-test:
	./testing/bootstrap.sh

.PHONY: build install test acceptance-test sweep test-compile vet fmt fmt-check lint clean docs appwrite-up appwrite-down appwrite-test
