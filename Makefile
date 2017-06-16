DESTDIR :=
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin
SYSCONFDIR := $(PREFIX)/etc
LIBEXECDIR := $(PREFIX)/libexec

CCDIR := "clear-containers"

SOURCES := $(shell find . 2>&1 | grep -E '.*\.(c|h|go)$$')
VERSION := ${shell cat ./VERSION}
COMMIT_NO := $(shell git rev-parse HEAD 2> /dev/null || true)
COMMIT := $(if $(shell git status --porcelain --untracked-files=no),"${COMMIT_NO}-dirty","${COMMIT_NO}")

TARGET = cc-runtime
CONFIG = configuration.toml

V           = @
Q           = $(V:1=)
QUIET_BUILD = $(Q:@=@echo    '     BUILD   '$@;)
QUIET_CHECK = $(Q:@=@echo    '     CHECK   '$@;)
QUIET_CLEAN = $(Q:@=@echo    '     CLEAN   '$@;)
QUIET_INST  = $(Q:@=@echo    '     INSTALL '$@;)
QUIET_TEST  = $(Q:@=@echo    '     TEST    '$@;)

.DEFAULT: $(TARGET)
$(TARGET): $(SOURCES) Makefile
	$(QUIET_BUILD)go build -i -ldflags "-X main.commit=${COMMIT} -X main.version=${VERSION} -X main.libExecDir=${LIBEXECDIR}" -o $@ .

.PHONY: check check-go-static check-go-test coverage
$(TARGET).coverage: $(SOURCES) Makefile
	$(QUIET_TEST)go test -o $@ -covermode count

check: check-go-static check-go-test

check-go-test:
	$(QUIET_TEST).ci/go-test.sh

check-go-static:
	$(QUIET_CHECK).ci/go-static-checks.sh $(GO_STATIC_CHECKS_ARGS)
	$(QUIET_CHECK).ci/go-no-os-exit.sh

coverage:
	$(QUIET_TEST).ci/go-test.sh html-coverage

install:
	$(QUIET_INST)install -D $(TARGET) $(DESTDIR)$(BINDIR)/$(TARGET)
	$(QUIET_INST)install -D config/$(CONFIG) $(DESTDIR)$(SYSCONFDIR)/$(CCDIR)/$(CONFIG)

clean:
	$(QUIET_CLEAN)rm -f $(TARGET)
