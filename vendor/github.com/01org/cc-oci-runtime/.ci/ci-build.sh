#!/bin/bash
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

set -e -x

source $(dirname "$0")/ci-common.sh

./autogen.sh

# We run the travis script from an exploded `make dist` tarball to ensure
# `make dist` has the necessary files to compile and run the tests.
#
# Note: we don't use `make distcheck` here as we can't run everything we want
#       for distcheck in travis.
# Note: chronic is used to limit the travis output while still showing it if
# the command fails.
chronic make dist
tarball=`ls -1 cc-oci-runtime-*.tar.xz`
chronic tar xvf "$tarball"
tarball_dir=${tarball%.tar.xz}

configure_opts=""
configure_opts+=" --sysconfdir=/etc"
configure_opts+=" --localstatedir=/var"
configure_opts+=" --prefix=/usr"
configure_opts+=" --enable-cppcheck"
configure_opts+=" --enable-valgrind"
configure_opts+=" --disable-valgrind-helgrind"
configure_opts+=" --disable-valgrind-drd"
configure_opts+=" --disable-silent-rules"

if [ "$nested" = "Y" ]
then
    configure_opts+=" --enable-docker-tests"
    configure_opts+=" --enable-functional-tests"
    configure_opts+=" --with-cc-image=\"${clr_assets_dir}/clear-containers.img\""
    configure_opts+=" --with-cc-kernel=\"${clr_assets_dir}/vmlinux.container\""
    configure_opts+=" --with-qemu-path=\"${prefix_dir}/bin/qemu-system-x86_64\""
else
    configure_opts+=" --disable-docker-tests"
    configure_opts+=" --disable-functional-tests"
fi

(cd "$ci_build_dir" && \
 eval ../"$tarball_dir"/configure "$configure_opts" \
 && make -j5 CFLAGS="-Werror -Wno-error=pedantic" \
 && sudo make install \
 && make check)

$(dirname "$0")/ci-go-static-checks.sh
