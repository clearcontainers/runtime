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

# Determine the lower-case name of the distro
distro := $(shell \
for file in /etc/os-release /usr/lib/os-release; do \
    if [ -e $$file ]; then \
        grep ^ID= $$file|cut -d= -f2-|tr -d '"'; \
        break; \
    fi \
done)

TARGET = cc-runtime
DESTDIR :=
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin
SYSCONFDIR := $(PREFIX)/etc
LIBEXECDIR := $(PREFIX)/libexec
LOCALSTATEDIR := $(PREFIX)/var
SHAREDIR := $(PREFIX)/share

CCDIR := clear-containers

PKGDATADIR := $(SHAREDIR)/$(CCDIR)
PKGLIBDIR := $(LOCALSTATEDIR)/lib/$(CCDIR)
PKGRUNDIR := $(LOCALSTATEDIR)/run/$(CCDIR)
PKGLIBEXECDIR := $(LIBEXECDIR)/$(CCDIR)

KERNELPATH := $(PKGDATADIR)/vmlinux.container
IMAGEPATH := $(PKGDATADIR)/clear-containers.img

QEMUBINDIR := $(BINDIR)

# The CentOS/RHEL hypervisor binary is not called qemu-lite
ifeq (,$(filter-out centos rhel,$(distro)))
QEMUCMD := qemu-system-x86_64
else
QEMUCMD := qemu-lite-system-x86_64
endif

QEMUPATH := $(QEMUBINDIR)/$(QEMUCMD)

SHIMCMD := cc-shim
SHIMPATH := $(PKGLIBEXECDIR)/$(SHIMCMD)

PROXYURL := unix://$(PKGRUNDIR)/proxy.sock

PAUSEROOTPATH := $(PKGLIBDIR)/runtime/bundles/pause_bundle
PAUSEBINRELPATH := bin/pause

GLOBALLOGPATH := $(PKGLIBDIR)/runtime/runtime.log

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
USER_VARS += GLOBALLOGPATH
USER_VARS += IMAGEPATH
USER_VARS += KERNELPATH
USER_VARS += LIBEXECDIR
USER_VARS += LOCALSTATEDIR
USER_VARS += PAUSEBINRELPATH
USER_VARS += PAUSEROOTPATH
USER_VARS += PKGDATADIR
USER_VARS += PKGLIBDIR
USER_VARS += PKGLIBEXECDIR
USER_VARS += PKGRUNDIR
USER_VARS += PREFIX
USER_VARS += PROXYURL
USER_VARS += QEMUBINDIR
USER_VARS += QEMUCMD
USER_VARS += QEMUPATH
USER_VARS += SHAREDIR
USER_VARS += SHIMPATH
USER_VARS += SYSCONFDIR

V              = @
Q              = $(V:1=)
QUIET_BUILD    = $(Q:@=@echo    '     BUILD   '$@;)
QUIET_CHECK    = $(Q:@=@echo    '     CHECK   '$@;)
QUIET_CLEAN    = $(Q:@=@echo    '     CLEAN   '$@;)
QUIET_CONFIG   = $(Q:@=@echo    '     CONFIG  '$@;)
QUIET_GENERATE = $(Q:@=@echo    '     GENERATE '$@;)
QUIET_INST     = $(Q:@=@echo    '     INSTALL '$@;)
QUIET_TEST     = $(Q:@=@echo    '     TEST    '$@;)

default: $(TARGET) $(CONFIG)
.DEFAULT: default

define GENERATED_CODE
// WARNING: This file is auto-generated - DO NOT EDIT!
package main

// commit is the git commit the runtime is compiled from.
const commit = "$(COMMIT)"

// version is the runtime version.
const version = "$(VERSION)"

const defaultHypervisorPath = "$(QEMUPATH)"
const defaultImagePath = "$(IMAGEPATH)"
const defaultKernelPath = "$(KERNELPATH)"
const defaultPauseRootPath = "$(PAUSEROOTPATH)"
const defaultProxyURL = "$(PROXYURL)"
const defaultRootDirectory = "$(PKGRUNDIR)"
const defaultRuntimeLib = "$(PKGLIBDIR)"
const defaultRuntimeRun = "$(PKGRUNDIR)"
const defaultShimPath = "$(SHIMPATH)"
const pauseBinRelativePath = "$(PAUSEBINRELPATH)"

// Required to be modifiable (for the tests)
var defaultRuntimeConfiguration = "$(DESTCONFIG)"
endef

export GENERATED_CODE


GENERATED_FILES += config-generated.go

config-generated.go:
	$(QUIET_GENERATE)echo "$$GENERATED_CODE" >$@

$(TARGET): $(SOURCES) $(GENERATED_FILES) Makefile | show-summary
	$(QUIET_BUILD)go build -i -o $@ .

.PHONY: \
	check \
	check-go-static \
	check-go-test \
	coverage \
	default \
	install \
	show-header \
	show-summary \
	show-variables

$(TARGET).coverage: $(SOURCES) $(GENERATED_FILES) Makefile
	$(QUIET_TEST)go test -o $@ -covermode count

$(CONFIG): $(CONFIG_IN)
	$(QUIET_CONFIG)$(SED) \
		-e "s|@CONFIG_IN@|$(CONFIG_IN)|g" \
		-e "s|@IMAGEPATH@|$(IMAGEPATH)|g" \
		-e "s|@KERNELPATH@|$(KERNELPATH)|g" \
		-e "s|@LOCALSTATEDIR@|$(LOCALSTATEDIR)|g" \
		-e "s|@PAUSEROOTPATH@|$(PAUSEROOTPATH)|g" \
		-e "s|@PKGLIBEXECDIR@|$(PKGLIBEXECDIR)|g" \
		-e "s|@PROXYURL@|$(PROXYURL)|g" \
		-e "s|@QEMUPATH@|$(QEMUPATH)|g" \
		-e "s|@SHIMPATH@|$(SHIMPATH)|g" \
		-e "s|@GLOBALLOGPATH@|$(GLOBALLOGPATH)|g" \
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

install: default
	$(QUIET_INST)install -D $(TARGET) $(DESTTARGET)
	$(QUIET_INST)install -D $(CONFIG) $(DESTCONFIG)

clean:
	$(QUIET_CLEAN)rm -f $(TARGET) $(CONFIG) $(GENERATED_FILES)

show-usage: show-header
	@printf "• Overview:\n"
	@printf "\n"
	@printf "\tTo build $(TARGET), just run, \"make\".\n"
	@printf "\n"
	@printf "\tFor a verbose build, run \"make V=1\".\n"
	@printf "\n"
	@printf "• Additional targets:\n"
	@printf "\n"
	@printf "\tcheck           : run tests\n"
	@printf "\tclean           : remove built files\n"
	@printf "\tcoverage        : run coverage tests\n"
	@printf "\tdefault         : same as just \"make\"\n"
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
	@printf "\tbinary install path (DESTTARGET) : %s\n" $(DESTTARGET)
	@printf "\tconfig install path (DESTCONFIG) : %s\n" $(DESTCONFIG)
	@printf "\n"
