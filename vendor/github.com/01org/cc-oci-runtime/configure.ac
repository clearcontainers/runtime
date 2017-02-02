#                                               -*- Autoconf -*-
# Process this file with autoconf to produce a configure script.
#
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

AC_PREREQ([2.69])
AC_INIT([cc-oci-runtime], [2.1.0-rc.5])
AC_CONFIG_MACRO_DIR([m4])
AM_INIT_AUTOMAKE([foreign -Wall -Werror -Wno-portability silent-rules subdir-objects color-tests no-dist-gzip dist-xz])
AM_SILENT_RULES([yes])
AC_CONFIG_SRCDIR([src/main.c])
AC_CONFIG_HEADERS([config.h])
AC_CONFIG_FILES([Makefile data/hypervisor.args])
AC_LANG(C)

LT_INIT

AX_VALGRIND_DFLT([sgcheck], [off])
AX_VALGRIND_CHECK

AX_CODE_COVERAGE

# Checks for programs.
AC_PROG_CC
AM_PROG_CC_C_O
AC_PROG_INSTALL
AC_PROG_MKDIR_P

# We need go!
AC_CHECK_PROG(GO,[go],[go],[no])
test "$GO" == "no" && AC_MSG_ERROR([go is required to build cc-oci-runtime])

# Checks for header files.
AC_CHECK_HEADERS([fcntl.h mntent.h stdlib.h string.h sys/mount.h unistd.h uuid/uuid.h gio/gio.h gio/gunixsocketaddress.h])

# Checks for typedefs, structures, and compiler characteristics.
AC_CHECK_HEADER_STDBOOL
AC_C_INLINE
AC_TYPE_INT64_T
AC_TYPE_PID_T
AC_TYPE_SIZE_T

# Checks for library functions.
AC_FUNC_FORK
AC_CHECK_FUNCS([atexit dup2 memset strdup])

AC_ARG_ENABLE(debug, AS_HELP_STRING([--enable-debug], [enable debug mode @<:@default=no@:>@]),
	      [], [enable_debug=no])
AS_IF([test "x$enable_debug" = "xyes"],
	[AC_DEFINE([DEBUG], [1], [Debugging mode enabled])],
	[AC_DEFINE([NDEBUG], [1], [Debugging and assertions disabled])])
AM_CONDITIONAL([DEBUG], [test x$enable_debug = x"yes"])

AS_IF([test "x$enable_debug" = "xyes"],
      [[CFLAGS=`echo "$CFLAGS" | sed -e "s/ -O[1-9s]*\b/ -O0/g"`
       CFLAGS=`echo "$CFLAGS" | sed -e "s/ -g\b/ -ggdb3/g"`
       LDFLAGS=`echo "$LDFLAGS" | sed -e "s/ -Wl,-O[0-9]*\b//g"`
       LDFLAGS=`echo "$LDFLAGS" | sed -e "s/ -g\b/ -ggdb3/g"`
      ]])

AC_ARG_ENABLE(tests, AS_HELP_STRING([--enable-tests], [build unit tests @<:@default=yes@:>@]),
          [], [enable_tests=yes])
AS_IF([test x"$enable_tests" = "xyes"],
    [AC_SUBST([BUILD_TESTS], [1])],
    [AC_SUBST([BUILD_TESTS], [0])])
AM_CONDITIONAL([BUILD_TESTS], [test x$enable_tests = x"yes"])

AC_ARG_ENABLE(cppcheck, AS_HELP_STRING([--enable-cppcheck], [enable static analysis @<:@default=no@:>@]),
          [], [enable_cppcheck=no])
AS_IF([test x"$enable_cppcheck" = "xyes"],
    [AC_SUBST([CPPCHECK], [1])],
    [AC_SUBST([CPPCHECK], [0])])
AM_CONDITIONAL([CPPCHECK], [test x$enable_cppcheck = x"yes"])

# Runtime functional tests

AC_ARG_ENABLE(functional-tests,
              AS_HELP_STRING([--disable-functional-tests],
                             [disable functional tests @<:@default=no@:>@]))

# If "--[enable|disable]-functional-tests" isn't specified, the variable will not
# have a value (meaning *DO* run the tests).
AS_IF([test x"$enable_functional_tests" = x"yes" -o x"$enable_functional_tests" = x""],
    [run_functional_tests=yes],
    [run_functional_tests=no])

AS_IF([test x"$run_functional_tests" = x"yes"],
    [AC_SUBST([FUNCTIONAL_TESTS], [1])],
    [AC_SUBST([FUNCTIONAL_TESTS], [0])])
AM_CONDITIONAL([FUNCTIONAL_TESTS], [test x"$run_functional_tests" = x"yes"])

# Docker integration tests
AC_ARG_ENABLE(docker-tests, AS_HELP_STRING([--disable-docker-tests],
              [disable docker tests @<:@default=no@:>@]))

AS_IF([test x"$enable_docker_tests" = x"yes" -o x"$enable_docker_tests" = x""],
    [run_docker_tests=yes],
    [run_docker_tests=no])

AS_IF([test x"$run_docker_tests" = x"yes"],
    [AC_SUBST([DOCKER_TESTS], [1])],
    [AC_SUBST([DOCKER_TESTS], [0])])
AM_CONDITIONAL([DOCKER_TESTS], [test x"$run_docker_tests" = x"yes"])

# Checks for libraries.
source $srcdir/versions.txt

PKG_CHECK_MODULES([GLIB], [glib-2.0 >= "$glib_version"], have_required_glib=yes, have_required_glib=no)
AS_IF([test "$have_required_glib" != "yes"], AC_MSG_ERROR([ERROR: need glib ${glib_version}+ for the `G_FILE_MONITOR_WATCH_MOVES' flag]))

PKG_CHECK_MODULES([GIO], [gio-unix-2.0])
PKG_CHECK_MODULES([JSON_GLIB], [json-glib-1.0])
PKG_CHECK_MODULES([UUID], [uuid])
PKG_CHECK_MODULES([LIBMNL], [libmnl])

AS_IF([test $BUILD_TESTS = 1],
[PKG_CHECK_MODULES([CHECK], [check])]
)

AS_IF([test $CPPCHECK = 1],
   [AC_PATH_PROG([CPPCHECK_PATH], [cppcheck], [no], [$PATH:/sbin:/usr/sbin])
   if test x"$CPPCHECK_PATH" == x"no" ; then
       AC_MSG_ERROR([Please install cppcheck.])
   fi
   AC_SUBST(CPPCHECK_PATH)]
)

AS_IF([test $FUNCTIONAL_TESTS = 1],
   [AC_PATH_PROG([BATS_PATH], [bats], [no], [$PATH:/sbin:/usr/sbin])
   if test x"$BATS_PATH" == x"no" ; then
       AC_MSG_ERROR([Please install bats (Bash Automated Testing System).)])
   fi
   AC_SUBST(BATS_PATH)]
)

#path to install systemd files used to boot by cc-image
AC_ARG_WITH([cc-image-systemdsystemunitdir],
	AS_HELP_STRING([--with-cc-image-systemdsystemunitdir=DIR],[path to systemd service directory for cc-image]),
	[
		CC_IMAGE_SYSTEMDSYSTEMUNIT_PATH=${withval}
		AC_SUBST(CC_IMAGE_SYSTEMDSYSTEMUNIT_PATH)
	],
	[CC_IMAGE_SYSTEMDSYSTEMUNIT_PATH=no]
)
AM_CONDITIONAL([CC_IMAGE_SYSTEMDSYSTEMUNIT],[test x"$CC_IMAGE_SYSTEMDSYSTEMUNIT_PATH" != xno])

		AC_SUBST([CC_IMAGE_SYSTEMDSYSTEMUNIT], [1])
#Check for qemu
DEFAULT_QEMU_PATH=/usr/bin/qemu-system-x86_64

AC_ARG_WITH([qemu-path],
    [AS_HELP_STRING([--with-qemu-path=[[QEMU-PATH]]],
        [path to qemu with pc-lite support  @<:@default=qemu-system-x86_64@:>@])
    ],
    [
        QEMU=$with_qemu_path
        QEMUout=`$QEMU -machine help | grep -o pc-lite | tail -1`
        AS_IF([test "x$QEMUout" != "xpc-lite"],
             AC_MSG_WARN([$QEMU does not support pc-lite machine]),
                         [has_pc_lite=yes]
        )
    ],
    [with_qemu_path=no]
)

AC_CACHE_CHECK(
    [for qemu that supports pc-lite machine], [ac_cv_path_QEMU],
    [
        AC_PATH_PROGS_FEATURE_CHECK([QEMU],
            [qemu-lite-system-x86_64 qemu-system-x86_64],
            [
                QEMUout=`$ac_path_QEMU -machine help | grep -o pc-lite | tail -1`
                AS_IF([test "x$QEMUout" = "xpc-lite"],
                      [
                          ac_cv_path_QEMU=$ac_path_QEMU
                          has_pc_lite=yes
                          ac_path_QEMU_found=:
                      ])
            ],
            [
                AC_MSG_WARN([No qemu detected with pc-lite machine support using:])
                ac_cv_path_QEMU=$DEFAULT_QEMU_PATH
                has_pc_lite=no
            ]
        )
    ]
)

AC_SUBST([QEMU_PATH], [$ac_cv_path_QEMU])
AC_DEFINE_UNQUOTED([QEMU_PATH], ["$QEMU_PATH"], "Path to qemu with pc-lite support")

#Read Kernel parameters
AC_SUBST([CMDLINE], [$(cat $srcdir/data/kernel-cmdline)])

# Read Dockerfile parameters
AC_SUBST([DOCKER_FEDORA_VERSION], [$docker_fedora_version])
AC_SUBST([DOCKER_ENGINE_FEDORA_VERSION], [$docker_engine_fedora_version])
AC_SUBST([DOCKER_UBUNTU_VERSION], [$docker_ubuntu_version])
AC_SUBST([DOCKER_ENGINE_UBUNTU_VERSION], [$docker_engine_ubuntu_version])

# Sytemd unit files (host)
AC_ARG_WITH([systemdsystemunitdir],
            AS_HELP_STRING([--with-systemdsystemunitdir=DIR], [Directory for systemd service files]),
            [],
            [with_systemdsystemunitdir=$($PKG_CONFIG --variable=systemdsystemunitdir systemd)])
if test "x$with_systemdsystemunitdir" != "xno"; then
  AC_SUBST([systemdsystemunitdir], [$with_systemdsystemunitdir])
fi
AM_CONDITIONAL(HAVE_SYSTEMD, [test -n "$systemdsystemunitdir"])

#bundle path for functional-test
AC_ARG_WITH([tests-bundle-path],
    [AS_HELP_STRING([--with-tests-bundle-path=[[BUNDLE-PATH]]],
            [path to OCI bundle to execute functional tests])
    ]
)

AC_ARG_ENABLE([auto-bundle-creation],
    AS_HELP_STRING([--disable-auto-bundle-creation], [do not
        automatically create a bundle directory
        if --with-tests-bundle-path= is not specified.]),
        [auto_bundle_creation=no],
        [auto_bundle_creation=yes])

AC_ARG_WITH([auto-bundle-creation-path],
    [AS_HELP_STRING([--with-auto-bundle-creation-path=[[BUNDLE-PATH]]],
            [path to create an OCI bundle in])
    ],
    [
        auto_bundle_creation_path="$with_auto_bundle_creation_path"
    ],
    [
        # default value
        auto_bundle_creation_path="tests/functional/bundle"
    ]
)

AS_IF([echo "$auto_bundle_creation_path"|grep -qv ^/],
      [auto_bundle_creation_path="$(pwd)/$auto_bundle_creation_path"])

# Determine if automatic bundle creation should be enabled.
AS_IF([test x"$auto_bundle_creation" = x"yes" dnl
    -a x"$with_tests_bundle_path" = x"" dnl
    -a x"$enable_tests" = x"yes" dnl
    -a x"$run_functional_tests" = x"yes"],
    [enable_auto_bundle_creation=yes])

AM_CONDITIONAL([AUTO_BUNDLE_CREATION],
               [test x"$enable_auto_bundle_creation" = x"yes"])

# Determine the bundle path.
AS_IF([test x"$with_tests_bundle_path" != x""],
      [AC_SUBST([BUNDLE_TEST_PATH], [$with_tests_bundle_path])],
      [test x"$auto_bundle_creation_path" != x""],
      [AC_SUBST([BUNDLE_TEST_PATH], [$auto_bundle_creation_path])],
      [])

#Kernel path for Clear Containers
DEFAULT_KERNEL_PATH=$localstatedir/lib/$PACKAGE_NAME/data/kernel/vmlinux.container
AC_ARG_WITH([cc-kernel],
    [AS_HELP_STRING([--with-cc-kernel=[[KERNEL-PATH]]],
        [clear container kernel path])
    ],
    [
        CONTAINER_KERNEL=$with_cc_kernel
        AC_SUBST([CONTAINER_KERNEL])
    ],
    [
        AC_SUBST([CONTAINER_KERNEL], [$DEFAULT_KERNEL_PATH])
    ]
)

#Container image path for Clear Containers
DEFAULT_CONTAINER_IMG=$localstatedir/lib/$PACKAGE_NAME/data/image/clear-containers.img
AC_ARG_WITH([cc-image],
    [AS_HELP_STRING([--with-cc-image=[[IMAGE-PATH]]],
        [clear container image path])
    ],
    [
        CONTAINERS_IMG=$with_cc_image
        AC_SUBST([CONTAINERS_IMG])
    ],
    [
        AC_SUBST([CONTAINERS_IMG],[$DEFAULT_CONTAINER_IMG])
    ]
)

AC_OUTPUT

dnl === Summary ===============================================================

AS_IF([test x"$disable_docker_tests" = xno],
      [enable_docker_tests=yes], [enable_docker_tests=no])

AC_MSG_RESULT([
cc-oci-runtime - $VERSION

 • General:

       QEMU                     : $ac_cv_path_QEMU
       QEMU pc-lite             : $has_pc_lite
       Container kernel         : $CONTAINER_KERNEL
       Container image          : $CONTAINERS_IMG
       Container image services : $CC_IMAGE_SYSTEMDSYSTEMUNIT_PATH
       Debug build              : $enable_debug

 • Proxy:

       Systemd unit directory   : $systemdsystemunitdir

 • Tests:

       Unit tests               : $enable_tests
       Functional tests         : $run_functional_tests
       Docker tests             : $run_docker_tests
       Cppcheck                 : $enable_cppcheck
       Valgrind                 : $VALGRIND_ENABLED
       Code coverage            : $CODE_COVERAGE_ENABLED
       Bundle path              : $BUNDLE_TEST_PATH
       Create + delete bundle   : $auto_bundle_creation
])
