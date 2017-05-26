DESTDIR :=
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin
SYSCONFDIR := $(PREFIX)/etc

SOURCES := $(shell find . 2>&1 | grep -E '.*\.(c|h|go)$$')
VERSION := ${shell cat ./VERSION}
COMMIT_NO := $(shell git rev-parse HEAD 2> /dev/null || true)
COMMIT := $(if $(shell git status --porcelain --untracked-files=no),"${COMMIT_NO}-dirty","${COMMIT_NO}")

TARGET = cc-runtime
CONFIG = configuration.toml

.DEFAULT: $(TARGET)
$(TARGET): $(SOURCES) Makefile
	go build -i -ldflags "-X main.commit=${COMMIT} -X main.version=${VERSION}" -o $@ .

.PHONY: check check-go-static check-go-test
check: check-go-static check-go-test

check-go-test:
	.ci/go-test.sh

check-go-static:
	.ci/go-static-checks.sh $(GO_STATIC_CHECKS_ARGS)

install:
	$(QUIET_INST)install -D $(TARGET) $(DESTDIR)$(BINDIR)/$(TARGET)
	$(QUIET_INST)install -D config/$(CONFIG) $(DESTDIR)$(SYSCONFDIR)/clear-containers/$(CONFIG)

clean:
	rm -f $(TARGET)
