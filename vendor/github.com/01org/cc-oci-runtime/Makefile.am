#  This file is part of cc-oci-runtime.
#
#  Copyright (C) 2016 Intel Corporation
#
#  This program is free software; you can redistribute it and/or
#  modify it under the terms of the GNU General Public License
#  as published by the Free Software Foundation; either version 2
#  of the License, or (at your option) any later version.
#
#  This program is distributed in the hope that it will be useful,
#  but WITHOUT ANY WARRANTY; without even the implied warranty of
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#  GNU General Public License for more details.
#
#  You should have received a copy of the GNU General Public License
#  along with this program; if not, write to the Free Software
#  Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
#

CLEANFILES = *.log

ACLOCAL_AMFLAGS = -I m4

AUTOMAKE_OPTIONS = parallel-tests

CHECK_DEPS =
FUNCTIONAL_TESTS_DEPS =

# Has a value if building in a git tree
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null)

# Has a value if building in a git-archive(1)-generated tree
# (see .gitattributes).
GENERATED_COMMIT = $(shell cat commit_id.fmt 2>/dev/null \| grep \'^[a-f0-9][a-f0-9]*\$\')

commit_id:
	$(AM_V_GEN)if [ "x$(GIT_COMMIT)" != "x" ]; then \
		echo "Determining commit id from git"; \
		echo "$(GIT_COMMIT)" > commit_id; \
	elif [ "x$(GENERATED_COMMIT)" != "x" ]; then \
		echo "Determining commit id from generated value"; \
		echo "$(GENERATED_COMMIT)" > commit_id; \
	else \
		echo "WARNING: cannot determine actual commit id"; \
		echo "0" > commit_id; \
	fi

STORED_COMMIT = $(shell cat $(builddir)/commit_id 2>/dev/null || cat $(srcdir)/commit_id 2>/dev/null)

SYSCONFDIR = $(sysconfdir)/$(PACKAGE_NAME)
DEFAULTSDIR = $(datadir)/defaults/$(PACKAGE_NAME)

src/commands/cc_oci_runtime-version.$(OBJEXT): \
	 commit_id

GENERATED_FILES = \
	data/cc-oci-runtime.sh \
	data/config.json \
	data/run-bats.sh \
	data/dockerFiles/Dockerfile.fedora \
	data/dockerFiles/Dockerfile.ubuntu \
	data/vm.json \
	tests/functional/common.bash \
	tests/functional/data/config-minimal-cc-oci.json \
	tests/metrics/density/docker_cpu_usage.sh \
	tests/metrics/density/docker_memory_usage.sh \
	tests/metrics/workload_time/cor_create_time.sh

$(GENERATED_FILES): %: %.in Makefile
	@mkdir -p `dirname $@`
	$(AM_V_GEN)$(SED) \
		-e 's|[@]bindir[@]|$(bindir)|g' \
		-e 's|[@]srcdir[@]|$(srcdir)|g' \
		-e 's|[@]libexecdir[@]|$(libexecdir)|' \
		-e "s|[@]localstatedir[@]|$(localstatedir)|" \
		-e 's|[@]BUNDLE_TEST_PATH[@]|$(BUNDLE_TEST_PATH)|g' \
		-e 's|[@]CMDLINE[@]|$(CMDLINE)|g' \
		-e 's|[@]CONTAINER_KERNEL[@]|$(CONTAINER_KERNEL)|g' \
		-e 's|[@]CONTAINERS_IMG[@]|$(CONTAINERS_IMG)|g' \
		-e 's|[@]DEFAULTSDIR[@]|$(DEFAULTSDIR)|g' \
		-e 's|[@]PACKAGE_NAME[@]|$(PACKAGE_NAME)|g' \
		-e 's|[@]QEMU_PATH[@]|$(QEMU_PATH)|g' \
		-e 's|[@]BATS_PATH[@]|$(BATS_PATH)|g' \
		-e 's|[@]ROOTFS_PATH[@]|$(BUNDLE_TEST_PATH)/rootfs|g' \
		-e 's|[@]SYSCONFDIR[@]|$(SYSCONFDIR)|g' \
		-e 's|[@]DOCKER_FEDORA_VERSION[@]|$(DOCKER_FEDORA_VERSION)|g' \
		-e 's|[@]DOCKER_ENGINE_FEDORA_VERSION[@]|$(DOCKER_ENGINE_FEDORA_VERSION)|g' \
		-e 's|[@]DOCKER_UBUNTU_VERSION[@]|$(DOCKER_UBUNTU_VERSION)|g' \
		-e 's|[@]DOCKER_ENGINE_UBUNTU_VERSION[@]|$(DOCKER_ENGINE_UBUNTU_VERSION)|g' \
		"$<" > "$@"

if FUNCTIONAL_TESTS
if AUTO_BUNDLE_CREATION

FUNCTIONAL_TESTS_DEPS += $(BUNDLE_TEST_PATH)

# XXX: don't fail if the bundle cannot be generated - docker might not
# be running for example. Those tests that need a bundle will detect
# that it doesn't exist and they will be skipped.
$(BUNDLE_TEST_PATH):
	$(AM_V_GEN)$(top_srcdir)/data/make-bundle-dir.sh $@ || true

endif
endif

CLEANFILES += $(GENERATED_FILES)

AM_CPPFLAGS = -I $(top_srcdir)/src -DG_LOG_DOMAIN=\"$(PACKAGE_NAME)\" \
	-DSYSCONFDIR=\"$(SYSCONFDIR)\" \
	-DDEFAULTSDIR=\"$(DEFAULTSDIR)\" \
	-DLOCALSTATEDIR=\"$(localstatedir)\" \
	-DGIT_COMMIT=\"$(STORED_COMMIT)\" \
	-DLIBEXECDIR=\"$(libexecdir)\"

defaultsdir = $(datadir)/defaults/$(PACKAGE_NAME)
defaults_DATA = data/vm.json data/hypervisor.args data/kernel-cmdline

cc_image_systemd_files = \
	data/cc-agent.target \
	data/cc-agent.service \
	data/container.target \
	data/opt-rootfs.mount \
	data/opt-rootfs-proc.mount \
	data/opt-rootfs-sys.mount \
	data/container-workload.service

if CC_IMAGE_SYSTEMDSYSTEMUNIT
ccimage_systemdfilesdir= @CC_IMAGE_SYSTEMDSYSTEMUNIT_PATH@
ccimage_systemdfiles_DATA = $(cc_image_systemd_files)
endif

AM_CFLAGS = -std=gnu99 -fstack-protector -Wall -pedantic \
	-Wstrict-prototypes -Wundef -fno-common \
	-Werror-implicit-function-declaration \
	-Wformat -Wformat-security -Werror=format-security \
	-Wconversion -Wunreachable-code

# We set --with-systemdunitdir here so make distcheck can run make install as a
# normal user and not fail.
# This directory doesn't have to be unique and doing several make distcheck
# will not end up clobbering /tmp this way.
DISTCHECK_CONFIGURE_FLAGS = \
	--enable-debug \
	--enable-tests \
	--enable-cppcheck \
	--enable-valgrind \
	--disable-valgrind-helgrind \
	--disable-valgrind-drd \
	--enable-code-coverage \
	--enable-functional-tests \
	--with-systemdsystemunitdir=/tmp/cc-distcheck/systemdunitdir/

bin_PROGRAMS = cc-oci-runtime
dist_bin_SCRIPTS = data/cc-oci-runtime.sh

common_sources = \
	src/util.c src/util.h \
	src/logging.c src/logging.h \
	src/oci.c src/oci.h \
	src/process.c src/process.h \
	src/mount.c src/mount.h \
	src/network.c src/network.h \
	src/networking.c src/networking.h \
	src/netlink.c src/netlink.h \
	src/state.c src/state.h \
	src/events.c src/events.h \
	src/runtime.c src/runtime.h \
	src/semver.c src/semver.h \
	src/annotation.c src/annotation.h \
	src/namespace.c src/namespace.h \
	src/priv.c src/priv.h \
	src/oci-config.c src/oci-config.h \
	src/hypervisor.c src/hypervisor.h \
	src/json.c src/json.h \
	src/proxy.c src/proxy.h \
	src/spec_handler.c src/spec_handler.h \
	src/pod.c src/pod.h \
	src/common.h \
	src/command.c src/command.h \
	src/commands/create.c \
	src/commands/delete.c \
	src/commands/exec.c \
	src/commands/events.c \
	src/commands/help.c \
	src/commands/kill.c \
	src/commands/list.c \
	src/commands/run.c \
	src/commands/start.c \
	src/commands/state.c \
	src/commands/stop.c \
	src/commands/pause.c \
	src/commands/ps.c \
	src/commands/resume.c \
	src/commands/version.c \
	src/commands/checkpoint.c \
	src/commands/restore.c \
	src/commands/update.c \
	src/spec_handlers/hooks.c \
	src/spec_handlers/platform.c \
	src/spec_handlers/process.c \
	src/spec_handlers/root.c \
	src/spec_handlers/vm.c \
	src/spec_handlers/linux.c \
	src/spec_handlers/mounts.c \
	src/spec_handlers/annotations.c

cc_oci_runtime_SOURCES = \
	src/main.c \
	$(common_sources)

cc_oci_runtime_LDADD = \
	$(GLIB_LIBS) \
	$(GIO_LIBS) \
	$(JSON_GLIB_LIBS) \
	$(LIBMNL_LIBS) \
	$(UUID_LIBS)

cc_oci_runtime_CFLAGS = \
	$(AM_CFLAGS) \
	$(GLIB_CFLAGS) \
	$(GIO_CFLAGS) \
	$(JSON_GLIB_CFLAGS) \
	$(LIBMNL_CFLAGS) \
	$(UUID_CFLAGS)

libexec_SCRIPTS = cc-proxy

CLEANFILES += cc-proxy

AM_V_GO    = $(am__v_GO_@AM_V@)
am__v_GO_  = $(am__v_GO_@AM_DEFAULT_V@)
am__v_GO_0 = @echo "  GOBUILD " $@;
am__v_GO_1 =

systemdservice_in_files =		\
	proxy/cc-proxy.service.in	\
	proxy/cc-proxy.socket.in

systemdservice_files = 		\
	proxy/cc-proxy.service	\
	proxy/cc-proxy.socket

CLEANFILES += $(systemdservice_files)

if HAVE_SYSTEMD
GENERATED_FILES += $(systemdservice_files)
systemdservicedir   = $(systemdsystemunitdir)
systemdservice_DATA = $(systemdservice_files)
endif

proxy_ldflags = "-X main.DefaultSocketPath=$(localstatedir)/run/cc-oci-runtime/proxy.sock"
cc-proxy: $(cc_proxy_sources)
	$(AM_V_GO)go build -o $@ -ldflags=$(proxy_ldflags) $(srcdir)/proxy

cc_proxy_sources =			\
	proxy/api/api.go		\
	proxy/api/client.go		\
	proxy/api/common_test.go	\
	proxy/api/fdpassing.go		\
	proxy/api/fdpassing_test.go	\
	proxy/api/protocol.go		\
	proxy/fdleak_test.go		\
	proxy/protocol.go		\
	proxy/protocol_test.go		\
	proxy/proxy.go			\
	proxy/proxy_test.go		\
	proxy/socket_activation.go	\
	proxy/syscall.go		\
	proxy/vm.go

cc_proxy_extra_dist =			\
	proxy/README.md			\
	proxy/COPYING

CHECK_DEPS += check-proxy

check-proxy:
	go test -v -race -timeout 2s $(srcdir)/proxy

check-go:
	@$(top_srcdir)/.ci/ci-go-static-checks.sh

libexec_PROGRAMS = cc-shim

cc_shim_SOURCES = \
	shim/shim.c \
	shim/shim.h \
	shim/utils.c \
	shim/utils.h \
	shim/log.c \
	shim/log.h

cc_shim_CFLAGS = \
	$(AM_CFLAGS)

bats_test_sources = \
	tests/functional/common.bash.in \
	data/run-bats.sh.in \
	tests/functional/data/config-minimal-cc-oci.json.in \
	tests/functional/help.bats \
	tests/functional/kill.bats \
	tests/functional/pause.bats \
	tests/functional/README \
	tests/functional/README.rst \
	tests/functional/start.bats \
	tests/functional/state.bats \
	tests/functional/version.bats

documentation_extra_dist =	\
	documentation/high-level-overview.png

EXTRA_DIST = \
	.ci \
	LICENSE \
	CONTRIBUTING.md \
	DESIGN.rst \
	Doxyfile \
	data/config.json.in \
	data/vm.json.in \
	data/make-bundle-dir.sh \
	tests/data \
	commit_id \
	versions.txt \
	$(documentation_extra_dist) \
	$(defaults_DATA) \
	$(cc_image_systemd_files) \
	$(cc_proxy_sources) \
	$(cc_proxy_extra_dist) \
	$(systemdservice_in_files) \
	data/cc-oci-runtime.sh.in \
	data/dockerFiles/Dockerfile.fedora.in \
	data/dockerFiles/Dockerfile.ubuntu.in \
	data/dockerFiles/Dockerfile.clearlinux \
	$(bats_test_sources) \
	tests/integration \
	tests/lib \
	tests/metrics/density/docker_cpu_usage.sh.in \
	tests/metrics/density/docker_memory_usage.sh.in \
	tests/metrics/workload_time/cor_create_time.sh.in

if CPPCHECK
CHECK_DEPS += cppcheck
cppcheck:
	@$(CPPCHECK_PATH) --enable=performance,unusedFunction,missingInclude \
		--language=c --std=c99 --std=posix \
		--error-exitcode=1 -I$(srcdir)/src $(srcdir)/src
endif

if FUNCTIONAL_TESTS
CHECK_DEPS += functional-tests
FUNCTIONAL_TESTS_DEPS += cc-oci-runtime cc-proxy cc-shim $(GENERATED_FILES)

functional-tests: $(FUNCTIONAL_TESTS_DEPS)
		$(AM_V_GEN)test -n "$(BUNDLE_TEST_PATH)" && \
			echo "Using bundle '$(BUNDLE_TEST_PATH)'" || true
		@if [ "$(builddir)" != "$(srcdir)" ]; then \
			rm -f $(builddir)/tests/functional/*.bats ; \
			for f in $(abs_top_srcdir)/tests/functional/*.bats; do \
				ln -s $$f $(builddir)/tests/functional/ ; \
			done; \
		fi
		@echo -e \
		"export PROXY_SOCKET_PATH=\"$(abs_top_srcdir)/tests/functional/proxy.sock\" \n" \
		"export SHIM_PATH=\"$(abs_top_srcdir)/cc-shim\" \n" \
		"echo 'Running a new instance of cc-proxy' \n" \
		"$(abs_top_srcdir)/cc-proxy -socket-path=\"\$${PROXY_SOCKET_PATH}\" & \n" \
		"p=\$$! \n" \
		"sleep 1 \n" \
		"bash $(builddir)/data/run-bats.sh $(builddir)/tests/functional/ \n" \
		"echo 'killing own cc-proxy instance' \n" \
		"kill -9 \$$p \n" \
		"rm -f \$${PROXY_SOCKET_PATH} \n" > $(abs_top_srcdir)/tests/functional/run_functional_tests.sh
		@bash -f $(abs_top_srcdir)/tests/functional/run_functional_tests.sh
		@rm -f $(abs_top_srcdir)/tests/functional/run_functional_tests.sh
endif

if DOCKER_TESTS
CHECK_DEPS += docker-tests
docker-tests: cc-oci-runtime
	@$(BATS_PATH) $(srcdir)/tests/integration/docker
endif

#### tests ####
if BUILD_TESTS

if VALGRIND_ENABLED
@VALGRIND_CHECK_RULES@
VALGRIND_memcheck_FLAGS = --tool=memcheck --error-exitcode=1 --leak-check=full --errors-for-leak-kinds=definite,indirect --show-possibly-lost=no
CHECK_DEPS += check-valgrind
endif

if CODE_COVERAGE_ENABLED
@CODE_COVERAGE_RULES@
CHECK_DEPS += code-coverage-capture
endif

check_LTLIBRARIES = libtest.la

libtest_la_SOURCES = \
	$(common_sources)

libtest_la_CFLAGS = \
	$(AM_CFLAGS) \
	$(CODE_COVERAGE_CFLAGS) \
	$(GLIB_CFLAGS) \
	$(GIO_CFLAGS) \
	$(JSON_GLIB_CFLAGS) \
	$(LIBMNL_CFLAGS) \
	$(UUID_CFLAGS) \
	-DTEST_DATA_DIR=\"$(abs_srcdir)/tests/data\" \
	-DUNIT_TESTING

libtest_la_LIBS = \
	$(CODE_COVERAGE_LDFLAGS)

TEST_COMMON_CFLAGS = \
	$(AM_CFLAGS) \
	$(CODE_COVERAGE_CFLAGS) \
	$(JSON_GLIB_CFLAGS) \
	$(GLIB_CFLAGS) \
	$(CHECK_CFLAGS) \
	-std=gnu99 \
	-DTEST_DATA_DIR=\"$(abs_srcdir)/tests/data\" \
	-DUNIT_TESTING

TEST_COMMON_LDADD = \
	libtest.la \
	$(CODE_COVERAGE_LDFLAGS) \
	$(JSON_GLIB_LIBS) \
	$(GLIB_LIBS) \
	$(CHECK_LIBS) \
	$(LIBMNL_LIBS) \
	$(UUID_LIBS)

TEST_COMMON_SOURCES = \
	tests/test_common.c \
	tests/test_common.h

TESTS = \
	hypervisor_test \
	json_test \
	logging_test \
	namespace_test \
	oci_config_test \
	oci_test \
	pod_test \
	priv_test \
	proxy_test \
	process_test \
	runtime_test \
	semver_test \
	state_test \
	util_test \
	mount_test \
	annotation_test \
	network_test \
	spec_handler_test \
	sh_annotations_test \
	sh_linux_test \
	sh_vm_test \
	sh_platform_test \
	sh_process_test \
	sh_root_test \
	sh_mounts_test \
	sh_hooks_test

noinst_PROGRAMS = \
	$(TESTS)

check_PROGRAMS = \
	$(TESTS)

## hypervisor.c test ##
hypervisor_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/hypervisor_test.c

hypervisor_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

hypervisor_test_LDADD = \
	$(TEST_COMMON_LDADD)

## json.c test ##
json_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/json_test.c

json_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

json_test_LDADD = \
	$(TEST_COMMON_LDADD)

## logging.c test ##
logging_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/logging_test.c

logging_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

logging_test_LDADD = \
	$(TEST_COMMON_LDADD)

## namespace.c test ##
namespace_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/namespace_test.c

namespace_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

namespace_test_LDADD = \
	$(TEST_COMMON_LDADD)

## oci-config.c test ##
oci_config_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/oci-config_test.c

oci_config_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

oci_config_test_LDADD = \
	$(TEST_COMMON_LDADD)

## oci.c test ##
oci_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/oci_test.c

oci_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

oci_test_LDADD = \
	$(TEST_COMMON_LDADD)

## process.c test ##
process_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/process_test.c

process_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

process_test_LDADD = \
	$(TEST_COMMON_LDADD)

## runtime.c test ##
runtime_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/runtime_test.c

runtime_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

runtime_test_LDADD = \
	$(TEST_COMMON_LDADD)

## semver.c test ##
semver_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/semver_test.c

semver_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

semver_test_LDADD = \
	$(TEST_COMMON_LDADD)

## state.c test ##
state_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/state_test.c

state_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

state_test_LDADD = \
	$(TEST_COMMON_LDADD)

## util.c test ##
util_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/util_test.c

util_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

util_test_LDADD = \
	$(TEST_COMMON_LDADD)

## priv.c test ##
priv_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/priv_test.c

priv_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

priv_test_LDADD = \
	$(TEST_COMMON_LDADD)

## proxy.c test ##
proxy_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/proxy_test.c

proxy_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

proxy_test_LDADD = \
	$(TEST_COMMON_LDADD)

## mount.c test ##
mount_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/mount_test.c

mount_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

mount_test_LDADD = \
	$(TEST_COMMON_LDADD)

## annotation.c test ##
annotation_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/annotation_test.c

annotation_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

annotation_test_LDADD = \
	$(TEST_COMMON_LDADD)

## network.c test ##
network_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/network_test.c

network_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

network_test_LDADD = \
	$(TEST_COMMON_LDADD)

## spec_handler.c ##
spec_handler_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/spec_handler_test.c

spec_handler_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

spec_handler_test_LDADD = \
	$(TEST_COMMON_LDADD)

## spec_handlers/annotations.c test ##
sh_annotations_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/spec_handlers/annotations_test.c

sh_annotations_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

sh_annotations_test_LDADD = \
	$(TEST_COMMON_LDADD)

## spec_handlers/linux.c test ##
sh_linux_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/spec_handlers/linux_test.c

sh_linux_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

sh_linux_test_LDADD = \
	$(TEST_COMMON_LDADD)

## spec_handlers/vm.c test ##
sh_vm_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/spec_handlers/vm_test.c

sh_vm_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

sh_vm_test_LDADD = \
	$(TEST_COMMON_LDADD)

## spec_handlers/platform.c test ##
sh_platform_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/spec_handlers/platform_test.c

sh_platform_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

sh_platform_test_LDADD = \
	$(TEST_COMMON_LDADD)

## spec_handlers/process.c test ##
sh_process_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/spec_handlers/process_test.c

sh_process_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

sh_process_test_LDADD = \
	$(TEST_COMMON_LDADD)

## spec_handlers/root.c test ##
sh_root_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/spec_handlers/root_test.c

sh_root_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

sh_root_test_LDADD = \
	$(TEST_COMMON_LDADD)

## spec_handlers/mounts.c test ##
sh_mounts_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/spec_handlers/mounts_test.c

sh_mounts_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

sh_mounts_test_LDADD = \
	$(TEST_COMMON_LDADD)

## spec_handlers/hooks.c test ##
sh_hooks_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/spec_handlers/hooks_test.c

sh_hooks_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

sh_hooks_test_LDADD = \
	$(TEST_COMMON_LDADD)

## pod.c test ##
pod_test_SOURCES = \
	$(TEST_COMMON_SOURCES) \
	tests/pod_test.c

pod_test_CFLAGS = \
	$(TEST_COMMON_CFLAGS)

pod_test_LDADD = \
	$(TEST_COMMON_LDADD)

CLEANFILES += tests/*~ tests/*.log tests/*.trs
CLEANFILES += core core.* vgcore.*
endif

check: $(CHECK_DEPS)

clean-local:
	rm -f commit_id
if FUNCTIONAL_TESTS
if AUTO_BUNDLE_CREATION
	$(AM_V_GEN)echo "Deleting generated bundle '$(BUNDLE_TEST_PATH)'"
	$(AM_V_GEN)rm -rf "$(BUNDLE_TEST_PATH)"
endif
endif
