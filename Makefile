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

# A "CC system build" is either triggered by specifying
# a special target or via an environment variable.
cc_system_build_requested := $(foreach f,\
    build-cc-system install-cc-system,\
    $(findstring $(f),$(MAKECMDGOALS)))
ifneq (,$(strip $(cc_system_build_requested)))
    cc_system_build = yes
else
    cc_system_build = no
endif

# If this environment variable is set to any value,
# enable the CC system build.
ifneq (,$(CC_SYSTEM_BUILD))
    cc_system_build = yes
endif

TARGET = cc-runtime
DESTDIR :=
CCDIR := clear-containers

ifeq ($(cc_system_build),yes)
    # Configure the build for a standard Clear Containers system that is
    # using OBS-generated packages.
    PREFIX        := /usr
    BINDIR        := $(PREFIX)/bin
    DESTBINDIR    := /usr/local/bin
    QEMUBINDIR    := $(BINDIR)
    SYSCONFDIR    := /etc
    LOCALSTATEDIR := /var
else
    PREFIX        := /usr/local
    BINDIR        := $(PREFIX)/bin
    DESTBINDIR    := $(DESTDIR)/$(BINDIR)
    QEMUBINDIR    := $(BINDIR)
    SYSCONFDIR    := $(PREFIX)/etc
    LOCALSTATEDIR := $(PREFIX)/var
endif

LIBEXECDIR := $(PREFIX)/libexec
SHAREDIR := $(PREFIX)/share

PKGDATADIR := $(SHAREDIR)/$(CCDIR)
PKGLIBDIR := $(LOCALSTATEDIR)/lib/$(CCDIR)
PKGRUNDIR := $(LOCALSTATEDIR)/run/$(CCDIR)
PKGLIBEXECDIR := $(LIBEXECDIR)/$(CCDIR)

KERNELPATH := $(PKGDATADIR)/vmlinux.container
IMAGEPATH := $(PKGDATADIR)/clear-containers.img

# The CentOS/RHEL hypervisor binary is not called qemu-lite
ifeq (,$(filter-out centos rhel,$(distro)))
QEMUCMD := qemu-system-x86_64
else
QEMUCMD := qemu-lite-system-x86_64
endif

QEMUPATH := $(QEMUBINDIR)/$(QEMUCMD)

SHIMCMD := cc-shim
SHIMPATH := $(PKGLIBEXECDIR)/$(SHIMCMD)

PROXYCMD := cc-proxy
PROXYURL := unix://$(PKGRUNDIR)/proxy.sock
PROXYPATH := $(PKGLIBEXECDIR)/$(PROXYCMD)

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

DESTTARGET := $(abspath $(DESTBINDIR)/$(TARGET))

DESTCONFDIR := $(DESTDIR)/$(SYSCONFDIR)/$(CCDIR)
DESTCONFIG := $(abspath $(DESTCONFDIR)/$(CONFIG_FILE))

PAUSEDESTDIR := $(abspath $(DESTDIR)/$(PAUSEROOTPATH)/$(PAUSEBINRELPATH))

override LIBS +=
override CFLAGS += -Os -Wall -Wextra -static

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
USER_VARS += PROXYPATH
USER_VARS += PROXYURL
USER_VARS += QEMUBINDIR
USER_VARS += QEMUCMD
USER_VARS += QEMUPATH
USER_VARS += SHAREDIR
USER_VARS += SHIMPATH
USER_VARS += SYSCONFDIR
USER_VARS += PAUSEDESTDIR


V              = @
Q              = $(V:1=)
QUIET_BUILD    = $(Q:@=@echo    '     BUILD   '$@;)
QUIET_CHECK    = $(Q:@=@echo    '     CHECK   '$@;)
QUIET_CLEAN    = $(Q:@=@echo    '     CLEAN   '$@;)
QUIET_CONFIG   = $(Q:@=@echo    '     CONFIG  '$@;)
QUIET_GENERATE = $(Q:@=@echo    '     GENERATE '$@;)
QUIET_INST     = $(Q:@=@echo    '     INSTALL '$@;)
QUIET_TEST     = $(Q:@=@echo    '     TEST    '$@;)

default: $(TARGET) $(CONFIG) pause
.DEFAULT: default

build-cc-system: default
install-cc-system: install

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
var defaultProxyPath = "$(PROXYPATH)"
endef

export GENERATED_CODE


GENERATED_FILES += config-generated.go

config-generated.go: Makefile VERSION
	$(QUIET_GENERATE)echo "$$GENERATED_CODE" >$@

$(TARGET): $(SOURCES) $(GENERATED_FILES) Makefile | show-summary
	$(QUIET_BUILD)go build -i -o $@ .

pause: pause/pause.o
	$(QUIET_BUILD)$(CC) -o pause/pause pause/*.o $(CFLAGS) $(LIBS)

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
	@ if [ -e pause/pause ]; then \
		install -D pause/pause $(PAUSEDESTDIR); \
	fi

clean:
	$(QUIET_CLEAN)rm -f $(TARGET) $(CONFIG) $(GENERATED_FILES)
	$(QUIET_CLEAN)rm -f pause/*.o pause/pause

show-usage: show-header
	@printf "• Overview:\n"
	@printf "\n"
	@printf "\tTo build $(TARGET), just run, \"make\".\n"
	@printf "\n"
	@printf "\tFor a verbose build, run \"make V=1\".\n"
	@printf "\n"
	@printf "• Additional targets:\n"
	@printf "\n"
	@printf "\tbuild-cc-system   : build using standard Clear Containers system paths\n"
	@printf "\tcheck             : run tests\n"
	@printf "\tclean             : remove built files\n"
	@printf "\tcoverage          : run coverage tests\n"
	@printf "\tdefault           : same as just \"make\"\n"
	@printf "\tgenerate-config   : create configuration file\n"
	@printf "\tinstall           : install files\n"
	@printf "\tinstall-cc-system : install using standard Clear Containers system paths\n"
	@printf "\tpause             : build pause binary\n"
	@printf "\tshow-summary      : show install locations\n"
	@printf "\n"

handle_help: show-usage show-summary show-variables show-footer

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
	@printf "\tClear Containers system build     : $(cc_system_build)\n"
	@printf "\n"
	@printf "\tbinary install path (DESTTARGET)  : %s\n" $(DESTTARGET)
	@printf "\tconfig install path (DESTCONFIG)  : %s\n" $(DESTCONFIG)
	@printf "\thypervisor path (QEMUPATH)        : %s\n" $(QEMUPATH)
	@printf "\tassets path (PKGDATADIR)          : %s\n" $(PKGDATADIR)
	@printf "\tproxy+shim path (PKGLIBEXECDIR)   : %s\n" $(PKGLIBEXECDIR)
	@printf "\tpause bundle path (PAUSEROOTPATH) : %s\n" $(PAUSEROOTPATH)
	@printf "\n"
