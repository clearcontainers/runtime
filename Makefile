# Copyright (c) 2017 Intel Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

TARGET = cc-runtime
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
COMMIT := $(if $(shell git status --porcelain --untracked-files=no),${COMMIT_NO}-dirty,${COMMIT_NO})

CONFIG_FILE = configuration.toml
CONFIG = config/$(CONFIG_FILE)
CONFIG_IN = $(CONFIG).in

DESTBINDIR := $(DESTDIR)/$(BINDIR)
DESTTARGET := $(abspath $(DESTBINDIR)/$(TARGET))

DESTCONFDIR := $(DESTDIR)/$(SYSCONFDIR)/$(CCDIR)
DESTCONFIG := $(abspath $(DESTCONFDIR)/$(CONFIG_FILE))

# list of variables the user may wish to override
USER_VARS += BINDIR
USER_VARS += DESTCONFIG
USER_VARS += DESTDIR
USER_VARS += DESTTARGET
USER_VARS += LIBEXECDIR
USER_VARS += LOCALSTATEDIR
USER_VARS += PKGDATADIR
USER_VARS += PKGLIBEXECDIR
USER_VARS += PREFIX
USER_VARS += QEMUBINDIR
USER_VARS += SHAREDIR
USER_VARS += SYSCONFDIR

V            = @
Q            = $(V:1=)
QUIET_BUILD  = $(Q:@=@echo    '     BUILD   '$@;)
QUIET_CHECK  = $(Q:@=@echo    '     CHECK   '$@;)
QUIET_CLEAN  = $(Q:@=@echo    '     CLEAN   '$@;)
QUIET_CONFIG = $(Q:@=@echo    '     CONFIG  '$@;)
QUIET_INST   = $(Q:@=@echo    '     INSTALL '$@;)
QUIET_TEST   = $(Q:@=@echo    '     TEST    '$@;)

.DEFAULT: $(TARGET)
$(TARGET): $(SOURCES) Makefile show-summary
	$(QUIET_BUILD)go build -i -ldflags "-X main.commit=${COMMIT} -X main.version=${VERSION} -X main.libExecDir=${LIBEXECDIR}" -o $@ .

.PHONY: \
	check \
	check-go-static \
	check-go-test \
	coverage \
	show-header \
	show-summary \
	show-variables

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
	$(QUIET_INST)install -D $(TARGET) $(DESTTARGET)
	$(QUIET_INST)install -D $(CONFIG) $(DESTCONFIG)

clean:
	$(QUIET_CLEAN)rm -f $(TARGET) $(CONFIG)

show-usage: show-header
	@printf "• Overview:\n"
	@printf "\n"
	@printf "  To build $(TARGET), just run, \"make\".\n"
	@printf "\n"
	@printf "  For a verbose build, run \"make V=1\".\n"
	@printf "\n"
	@printf "• Additional targets:\n"
	@printf "\n"
	@printf "\tcheck           : run tests\n"
	@printf "\tclean           : remove built files\n"
	@printf "\tcoverage        : run coverage tests\n"
	@printf "\tgenerate-config : create configuration file\n"
	@printf "\tinstall         : install files\n"
	@printf "\tshow-summary    : show install locations\n"
	@printf "\n"

handle_help: show-usage show-variables show-footer

usage: handle_help
help: handle_help

show-variables:
	@printf "• Variables affecting the build:\n\n"
	@printf \
          "$(sort $(foreach v,$(USER_VARS),\t$(v)=$(value $(v))\n))"
	@printf "\n"

show-header:
	@printf "%s - version %s (commit %s)\n\n" $(TARGET) $(VERSION) $(COMMIT)

show-footer:
	@printf "• Project home: https://github.com/clearcontainers/runtime\n\n"

show-summary: show-header
	@printf "• Summary:\n"
	@printf "\n"
	@printf "  binary install path (DESTTARGET) : %s\n" $(DESTTARGET)
	@printf "  config install path (DESTCONFIG) : %s\n" $(DESTCONFIG)
	@printf "\n"
