# cantcp build system.

GO      ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/burn-lab-dev/cantcp/internal/version.Version=$(VERSION)
GOFLAGS := -trimpath
BINDIR  := bin
BINS    := cantcpd cantcp-cli
# Target name to GOARCH pairs published in releases. GOARM is set for armv7.
TARGETS := amd64:amd64 arm64:arm64 armv7:arm
# Debian architecture to GOARCH pairs used by the deb target.
DEBTARGETS := amd64:amd64 arm64:arm64 armhf:arm

.PHONY: all build test vet fmt cross deb cover clean version

all: build

build:
	@mkdir -p $(BINDIR)
	@for bin in $(BINS); do \
		echo "build $$bin ($(VERSION))"; \
		CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BINDIR)/$$bin ./cmd/$$bin || exit 1; \
	done

test:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

fmt:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

cross:
	@mkdir -p $(BINDIR)
	@for target in $(TARGETS); do \
		name=$${target%%:*}; goarch=$${target##*:}; \
		for bin in $(BINS); do \
			echo "build $$bin linux/$$name"; \
			GOARM=7 CGO_ENABLED=0 GOOS=linux GOARCH=$$goarch \
				$(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' \
				-o $(BINDIR)/$$bin-linux-$$name ./cmd/$$bin || exit 1; \
		done; \
	done

# Build the .deb packages for all published architectures into dist/.
# Requires dpkg-deb; the version is taken from git describe without the
# leading "v".
deb:
	@for target in $(DEBTARGETS); do \
		./scripts/build-deb.sh $(VERSION:v%=%) $${target%%:*} $${target##*:} || exit 1; \
	done

cover:
	$(GO) test -race -covermode=atomic -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

clean:
	rm -rf $(BINDIR) coverage.out

version:
	@echo $(VERSION)
