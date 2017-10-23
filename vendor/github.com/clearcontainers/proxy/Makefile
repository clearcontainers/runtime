# This is the version string used when it isn't possible to do a git describe,
# which happens with a git-archive tarball for instance. A '+' is appended to
# the version string to mean that it's a development version eg. 3.0.0+ means
# somewhere between 3.0.0 and 3.0.1.
#
# The version should be bumped and the '+' sign removed just before tagging a
# new release.
# A '+' sign should be added in the commit just after tagging a new release.
VERSION := 3.0.1+

DESTDIR :=
PREFIX := /usr
BINDIR=$(PREFIX)/bin
LIBEXECDIR := $(PREFIX)/libexec
LOCALSTATEDIR := /var

SOURCES := $(shell find . 2>&1 | grep -E '.*\.(c|h|go)$$')
PROXY_SOCKET := $(LOCALSTATEDIR)/run/clear-containers/proxy.sock

DESCRIBE := $(shell git describe 2> /dev/null || true)
DESCRIBE_DIRTY := $(if $(shell git status --porcelain --untracked-files=no 2> /dev/null),${DESCRIBE}-dirty,${DESCRIBE})
ifneq ($(DESCRIBE_DIRTY),)
VERSION := $(DESCRIBE_DIRTY)
endif

#
# systemd files
#

HAVE_SYSTEMD := $(shell pkg-config --exists systemd 2>/dev/null && echo 'yes')

ifeq ($(HAVE_SYSTEMD),yes)
UNIT_DIR := $(shell pkg-config --variable=systemdsystemunitdir systemd)
UNIT_FILES = cc-proxy.service cc-proxy.socket
GENERATED_FILES += $(UNIT_FILES)
endif

#
# Pretty printing
#

V	      = @
Q	      = $(V:1=)
QUIET_GOBUILD = $(Q:@=@echo    '     GOBUILD  '$@;)
QUIET_GEN     = $(Q:@=@echo    '     GEN      '$@;)

# Entry point
all: cc-proxy $(UNIT_FILES)

#
# proxy
#

cc-proxy: $(SOURCES) Makefile
	$(QUIET_GOBUILD)go build -i -o $@ -ldflags \
		"-X main.DefaultSocketPath=$(PROXY_SOCKET) -X main.Version=$(VERSION)"

#
# Tests
#

.PHONY: check check-go-static check-go-test
check: check-go-static check-go-test

check-go-static:
	.ci/go-lint.sh

check-go-test:
	.ci/go-test.sh

coverage:
	.ci/go-test.sh html-coverage

#
# Documentation
#

doc:
	$(Q).ci/go-doc.sh || true

#
# install
#

define INSTALL_EXEC
	$(QUIET_INST)install -D $1 $(DESTDIR)$2/$1 || exit 1;

endef
define INSTALL_FILE
	$(QUIET_INST)install -D -m 644 $1 $(DESTDIR)$2/$1 || exit 1;

endef

all-installable: cc-proxy $(UNIT_FILES)

install: all-installable
	$(call INSTALL_EXEC,cc-proxy,$(LIBEXECDIR)/clear-containers)
	$(foreach f,$(UNIT_FILES),$(call INSTALL_FILE,$f,$(UNIT_DIR)))

clean:
	rm -f cc-proxy $(GENERATED_FILES)

$(GENERATED_FILES): %: %.in Makefile
	@mkdir -p `dirname $@`
	$(QUIET_GEN)sed \
		-e 's|[@]bindir[@]|$(BINDIR)|g' \
		-e 's|[@]libexecdir[@]|$(LIBEXECDIR)|' \
		-e "s|[@]localstatedir[@]|$(LOCALSTATEDIR)|" \
		"$<" > "$@"

#
# dist
#

dist:
	git archive --format=tar --prefix=cc-proxy-$(VERSION)/ HEAD | xz -c > cc-proxy-$(VERSION).tar.xz
