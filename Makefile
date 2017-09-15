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

installing = $(findstring install,$(MAKECMDGOALS))

ifeq ($(cc_system_build),yes)
    # Configure the build for a standard Clear Containers system that is
    # using OBS-generated packages.
    PREFIX        := /usr
    BINDIR        := $(PREFIX)/bin
    DESTBINDIR    := /usr/local/bin
    QEMUBINDIR    := $(BINDIR)
    SYSCONFDIR    := /etc
    LOCALSTATEDIR := /var

    ifeq (,$(installing))
        # Force a rebuild to ensure version details are correct
        # (but only for a non-install build phase).
        EXTRA_DEPS = clean
    endif
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
DEFAULTSDIR := $(SHAREDIR)/defaults

PKGDATADIR := $(SHAREDIR)/$(CCDIR)
PKGLIBDIR := $(LOCALSTATEDIR)/lib/$(CCDIR)
PKGRUNDIR := $(LOCALSTATEDIR)/run/$(CCDIR)
PKGLIBEXECDIR := $(LIBEXECDIR)/$(CCDIR)

KERNELPATH := $(PKGDATADIR)/vmlinuz.container
IMAGEPATH := $(PKGDATADIR)/clear-containers.img

KERNELPARAMS :=

# The CentOS/RHEL hypervisor binary is not called qemu-lite
ifeq (,$(filter-out centos rhel,$(distro)))
QEMUCMD := qemu-system-x86_64
else
QEMUCMD := qemu-lite-system-x86_64
endif

QEMUPATH := $(QEMUBINDIR)/$(QEMUCMD)
MACHINETYPE := pc

SHIMCMD := cc-shim
SHIMPATH := $(PKGLIBEXECDIR)/$(SHIMCMD)

PROXYCMD := cc-proxy
PROXYURL := unix://$(PKGRUNDIR)/proxy.sock
PROXYPATH := $(PKGLIBEXECDIR)/$(PROXYCMD)

PAUSEROOTPATH := $(PKGLIBDIR)/runtime/bundles/pause_bundle
PAUSEBINRELPATH := bin/pause

GLOBALLOGPATH := $(PKGLIBDIR)/runtime/runtime.log

# Default number of vCPUs
DEFVCPUS := 1
# Default memory size in MiB
DEFMEMSZ := 2048

DEFDISABLEBLOCK := false

SED = sed

SOURCES := $(shell find . 2>&1 | grep -E '.*\.(c|h|go)$$')
VERSION := ${shell cat ./VERSION}
COMMIT_NO := $(shell git rev-parse HEAD 2> /dev/null || true)
COMMIT := $(if $(shell git status --porcelain --untracked-files=no),${COMMIT_NO}-dirty,${COMMIT_NO})

CONFIG_FILE = configuration.toml
CONFIG = config/$(CONFIG_FILE)
CONFIG_IN = $(CONFIG).in

DESTTARGET := $(abspath $(DESTBINDIR)/$(TARGET))

DESTCONFDIR := $(DESTDIR)/$(DEFAULTSDIR)/$(CCDIR)
DESTSYSCONFDIR := $(DESTDIR)/$(SYSCONFDIR)/$(CCDIR)

# Main configuration file location for stateless systems
DESTCONFIG := $(abspath $(DESTCONFDIR)/$(CONFIG_FILE))

# Secondary configuration file location. Note that this takes precedence
# over DESTCONFIG.
DESTSYSCONFIG := $(abspath $(DESTSYSCONFDIR)/$(CONFIG_FILE))

DESTSHAREDIR := $(DESTDIR)/$(SHAREDIR)

PAUSEDESTDIR := $(abspath $(DESTDIR)/$(PAUSEROOTPATH)/$(PAUSEBINRELPATH))

BASH_COMPLETIONS := data/completions/bash/cc-runtime
BASH_COMPLETIONSDIR := $(DESTSHAREDIR)/bash-completion/completions

# list of variables the user may wish to override
USER_VARS += BASH_COMPLETIONSDIR
USER_VARS += BINDIR
USER_VARS += CC_SYSTEM_BUILD
USER_VARS += DESTCONFIG
USER_VARS += DESTDIR
USER_VARS += DESTSYSCONFIG
USER_VARS += DESTTARGET
USER_VARS += GLOBALLOGPATH
USER_VARS += IMAGEPATH
USER_VARS += MACHINETYPE
USER_VARS += KERNELPATH
USER_VARS += KERNELPARAMS
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
USER_VARS += DEFVCPUS
USER_VARS += DEFMEMSZ
USER_VARS += DEFDISABLEBLOCK


V              = @
Q              = $(V:1=)
QUIET_BUILD    = $(Q:@=@echo    '     BUILD   '$@;)
QUIET_CHECK    = $(Q:@=@echo    '     CHECK   '$@;)
QUIET_CLEAN    = $(Q:@=@echo    '     CLEAN   '$@;)
QUIET_CONFIG   = $(Q:@=@echo    '     CONFIG  '$@;)
QUIET_GENERATE = $(Q:@=@echo    '     GENERATE '$@;)
QUIET_INST     = $(Q:@=@echo    '     INSTALL '$@;)
QUIET_TEST     = $(Q:@=@echo    '     TEST    '$@;)

default: $(TARGET) $(CONFIG) pause install-git-hooks
.DEFAULT: default

build: default
build-cc-system: default
install-cc-system: install

define GENERATED_CODE
// WARNING: This file is auto-generated - DO NOT EDIT!
//
// Note that some variables are "var" to allow them to be modified
// by the tests.
package main

// commit is the git commit the runtime is compiled from.
var commit = "$(COMMIT)"

// version is the runtime version.
var version = "$(VERSION)"

const defaultHypervisorPath = "$(QEMUPATH)"
const defaultImagePath = "$(IMAGEPATH)"
const defaultKernelPath = "$(KERNELPATH)"
const defaultKernelParams = "$(KERNELPARAMS)"
const defaultMachineType = "$(MACHINETYPE)"
const defaultPauseRootPath = "$(PAUSEROOTPATH)"
const defaultProxyURL = "$(PROXYURL)"
const defaultRootDirectory = "$(PKGRUNDIR)"
const defaultRuntimeLib = "$(PKGLIBDIR)"
const defaultRuntimeRun = "$(PKGRUNDIR)"
const defaultShimPath = "$(SHIMPATH)"
const pauseBinRelativePath = "$(PAUSEBINRELPATH)"

const defaultVCPUCount uint32 = $(DEFVCPUS)
const defaultMemSize uint32 = $(DEFMEMSZ) // MiB
const defaultDisableBlockDeviceUse bool = $(DEFDISABLEBLOCK)

// Default config file used by stateless systems.
var defaultRuntimeConfiguration = "$(DESTCONFIG)"

// Alternate config file that takes precedence over
// defaultRuntimeConfiguration.
var defaultSysConfRuntimeConfiguration = "$(DESTSYSCONFIG)"

var defaultProxyPath = "$(PROXYPATH)"
endef

export GENERATED_CODE


GENERATED_FILES += config-generated.go

config-generated.go: Makefile VERSION
	$(QUIET_GENERATE)echo "$$GENERATED_CODE" >$@

$(TARGET): $(EXTRA_DEPS) $(SOURCES) $(GENERATED_FILES) Makefile | show-summary
	$(QUIET_BUILD)go build -i -o $@ .

pause: pause/pause.go
	$(QUIET_BUILD)go build -o pause/pause $<

.PHONY: \
	check \
	check-go-static \
	check-go-test \
	coverage \
	default \
	install \
	install-git-hooks \
	pause \
	show-header \
	show-summary \
	show-variables

$(TARGET).coverage: $(SOURCES) $(GENERATED_FILES) Makefile
	$(QUIET_TEST)go test -o $@ -covermode count

$(CONFIG): $(CONFIG_IN) $(GENERATED_FILES)
	$(QUIET_CONFIG)$(SED) \
		-e "s|@CONFIG_IN@|$(CONFIG_IN)|g" \
		-e "s|@IMAGEPATH@|$(IMAGEPATH)|g" \
		-e "s|@KERNELPATH@|$(KERNELPATH)|g" \
		-e "s|@KERNELPARAMS@|$(KERNELPARAMS)|g" \
		-e "s|@LOCALSTATEDIR@|$(LOCALSTATEDIR)|g" \
		-e "s|@PAUSEROOTPATH@|$(PAUSEROOTPATH)|g" \
		-e "s|@PKGLIBEXECDIR@|$(PKGLIBEXECDIR)|g" \
		-e "s|@PROXYURL@|$(PROXYURL)|g" \
		-e "s|@QEMUPATH@|$(QEMUPATH)|g" \
		-e "s|@MACHINETYPE@|$(MACHINETYPE)|g" \
		-e "s|@SHIMPATH@|$(SHIMPATH)|g" \
		-e "s|@GLOBALLOGPATH@|$(GLOBALLOGPATH)|g" \
		-e "s|@DEFVCPUS@|$(DEFVCPUS)|g" \
		-e "s|@DEFMEMSZ@|$(DEFMEMSZ)|g" \
		-e "s|@DEFDISABLEBLOCK@|$(DEFDISABLEBLOCK)|g" \
		$< > $@

generate-config: $(CONFIG)

check: check-go-static check-go-test

check-go-test: $(GENERATED_FILES)
	$(QUIET_TEST).ci/go-test.sh

check-go-static:
	$(QUIET_CHECK).ci/go-static-checks.sh $(GO_STATIC_CHECKS_ARGS)
	$(QUIET_CHECK).ci/go-no-os-exit.sh

coverage:
	$(QUIET_TEST).ci/go-test.sh html-coverage

install: default install-completions
	$(QUIET_INST)install -D $(TARGET) $(DESTTARGET)
	$(QUIET_INST)install -D $(CONFIG) $(DESTCONFIG)
	@ if [ -e pause/pause ]; then \
		install -D pause/pause $(PAUSEDESTDIR); \
	fi

install-completions:
	$(QUIET_INST)install --mode 0644 -D $(BASH_COMPLETIONS) $(BASH_COMPLETIONSDIR)

clean:
	$(QUIET_CLEAN)rm -f $(TARGET) $(CONFIG) $(GENERATED_FILES)
	$(QUIET_CLEAN)rm -f pause/pause

show-usage: show-header
	@printf "• Overview:\n"
	@printf "\n"
	@printf "\tTo build $(TARGET), just run, \"make\".\n"
	@printf "\n"
	@printf "\tFor a verbose build, run \"make V=1\".\n"
	@printf "\n"
	@printf "• Additional targets:\n"
	@printf "\n"
	@printf "\tbuild             : standard build (equivalent to 'build-cc-system' if CC_SYSTEM_BUILD set)\n"
	@printf "\tdefault           : same as 'build'\n"
	@printf "\tbuild-cc-system   : build using standard Clear Containers system paths\n"
	@printf "\tcheck             : run tests\n"
	@printf "\tclean             : remove built files\n"
	@printf "\tcoverage          : run coverage tests\n"
	@printf "\tdefault           : same as just \"make\"\n"
	@printf "\tgenerate-config   : create configuration file\n"
	@printf "\tinstall           : install files (equivalent to 'install-cc-system' if CC_SYSTEM_BUILD set)\n"
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
	@printf "\tClear Containers system build         : $(cc_system_build)\n"
	@printf "\n"
	@printf "\tbinary install path (DESTTARGET)      : %s\n" $(DESTTARGET)
	@printf "\tconfig install path (DESTCONFIG)      : %s\n" $(DESTCONFIG)
	@printf "\talternate config path (DESTSYSCONFIG) : %s\n" $(DESTSYSCONFIG)
	@printf "\thypervisor path (QEMUPATH)            : %s\n" $(QEMUPATH)
	@printf "\tassets path (PKGDATADIR)              : %s\n" $(PKGDATADIR)
	@printf "\tproxy+shim path (PKGLIBEXECDIR)       : %s\n" $(PKGLIBEXECDIR)
	@printf "\tpause bundle path (PAUSEROOTPATH)     : %s\n" $(PAUSEROOTPATH)
	@printf "\n"


# The following git hooks handle HEAD changes:
# post-checkout <prev_head> <new_head> <file_or_branch_checkout>
# post-commit # no parameters
# post-merge <squash_or_not>
# post-rewrite <amend_or_rebase>
#
define GIT_HOOK_POST_CHECKOUT
#!/usr/bin/env bash
prev_head=$$1
new_head=$$2
[[ "$$prev_head" == "$$new_head" ]] && exit
printf "\nexecuting post-checkout git hook\n\n"
rm -f config-generated.go
endef
export GIT_HOOK_POST_CHECKOUT

define GIT_HOOK_POST_GENERIC
#!/usr/bin/env bash
printf "\n executing $$0 git hook\n\n"
rm -f config-generated.go
endef
export GIT_HOOK_POST_GENERIC

# This git-hook is executed after every checkout git operation
.git/hooks/post-checkout: Makefile
	@ mkdir -p .git/hooks/
	$(QUIET_INST)echo "$$GIT_HOOK_POST_CHECKOUT" >$@
	@ chmod +x $@

# This git-hook is executed after every commit, merge, amend or rebase git
# operation
.git/hooks/post-commit .git/hooks/post-merge .git/hooks/post-rewrite: Makefile
	@ mkdir -p .git/hooks/
	$(QUIET_INST)echo "$$GIT_HOOK_POST_GENERIC" >$@
	@ chmod +x $@

install-git-hooks: .git/hooks/post-checkout .git/hooks/post-commit \
    .git/hooks/post-merge .git/hooks/post-rewrite
