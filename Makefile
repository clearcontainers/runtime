DESTDIR :=
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin
QEMUBINDIR := $(BINDIR)
SYSCONFDIR := $(PREFIX)/etc
LIBEXECDIR := $(PREFIX)/libexec
LOCALSTATEDIR := $(PREFIX)/var
SHAREDIR := $(PREFIX)/share

CCDIR := clear-containers
PKGDATADIR := $(SHAREDIR)/$(CCDIR)
PKGLIBEXECDIR := $(LIBEXECDIR)/$(CCDIR)

SED = sed

SOURCES := $(shell find . 2>&1 | grep -E '.*\.(c|h|go)$$')
VERSION := ${shell cat ./VERSION}
COMMIT_NO := $(shell git rev-parse HEAD 2> /dev/null || true)
COMMIT := $(if $(shell git status --porcelain --untracked-files=no),"${COMMIT_NO}-dirty","${COMMIT_NO}")

TARGET = cc-runtime
CONFIG_FILE = configuration.toml
CONFIG = config/$(CONFIG_FILE)
CONFIG_IN = $(CONFIG).in


V            = @
Q            = $(V:1=)
QUIET_BUILD  = $(Q:@=@echo    '     BUILD   '$@;)
QUIET_CHECK  = $(Q:@=@echo    '     CHECK   '$@;)
QUIET_CLEAN  = $(Q:@=@echo    '     CLEAN   '$@;)
QUIET_CONFIG = $(Q:@=@echo    '     CONFIG  '$@;)
QUIET_INST   = $(Q:@=@echo    '     INSTALL '$@;)
QUIET_TEST   = $(Q:@=@echo    '     TEST    '$@;)

.DEFAULT: $(TARGET)
$(TARGET): $(SOURCES) Makefile
	$(QUIET_BUILD)go build -i -ldflags "-X main.commit=${COMMIT} -X main.version=${VERSION} -X main.libExecDir=${LIBEXECDIR}" -o $@ .

.PHONY: check check-go-static check-go-test coverage
$(TARGET).coverage: $(SOURCES) Makefile
	$(QUIET_TEST)go test -o $@ -covermode count

$(CONFIG): $(CONFIG_IN)
	$(QUIET_CONFIG)$(SED) \
		-e "s|@CCDIR@|$(CCDIR)|g" \
		-e "s|@CONFIG_IN@|$(CONFIG_IN)|g" \
		-e "s|@PKGLIBEXECDIR@|$(PKGLIBEXECDIR)|g" \
		-e "s|@PKGDATADIR@|$(PKGDATADIR)|g" \
		-e "s|@QEMUBINDIR@|$(QEMUBINDIR)|g" \
		-e "s|@LOCALSTATEDIR@|$(LOCALSTATEDIR)|g" \
		$< > $@

generate-config: $(CONFIG)

check: check-go-static check-go-test

check-go-test:
	$(QUIET_TEST).ci/go-test.sh

check-go-static:
	$(QUIET_CHECK).ci/go-static-checks.sh $(GO_STATIC_CHECKS_ARGS)
	$(QUIET_CHECK).ci/go-no-os-exit.sh

coverage:
	$(QUIET_TEST).ci/go-test.sh html-coverage

install: $(TARGET) $(CONFIG)
	$(QUIET_INST)install -D $(TARGET) $(DESTDIR)$(BINDIR)/$(TARGET)
	$(QUIET_INST)install -D $(CONFIG) $(DESTDIR)$(SYSCONFDIR)/$(CCDIR)/$(CONFIG_FILE)

clean:
	$(QUIET_CLEAN)rm -f $(TARGET) $(CONFIG)
