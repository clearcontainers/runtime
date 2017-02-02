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

# Return true if currently running in a recognised CI environment
cor_ci_env()
{
    # Set by TravisCI and SemaphoreCI
    [ "$CI" = true ]
}

nested=$(cat /sys/module/kvm_intel/parameters/nested 2>/dev/null \
    || echo N)

# Do not display output if this file is sourced via a BATS test since it
# will cause the test to fail.
[ -z "$BATS_TEST_DIRNAME" ] && echo "INFO: Nested kvm available: $nested"

if [ -n "$SEMAPHORE_CACHE_DIR" ]
then
    # Running under SemaphoreCI
    prefix_dir="$SEMAPHORE_CACHE_DIR/cor"
else
    prefix_dir="$HOME/.cache/cor"
fi

deps_dir="${prefix_dir}/dependencies"
cor_ci_env && mkdir -p "$deps_dir" || :

export LD_LIBRARY_PATH="${prefix_dir}/lib:$LD_LIBRARY_PATH"
export PKG_CONFIG_PATH="${prefix_dir}/lib/pkgconfig:$PKG_CONFIG_PATH"
export ACLOCAL_FLAGS="-I \"${prefix_dir}/share/aclocal\" $ACLOCAL_FLAG"
export GOROOT=$HOME/go
export GOPATH=$HOME/gopath
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH
export PATH="${prefix_dir}/bin:${prefix_dir}/sbin:$PATH"

# Install Clear Containers assets into same directory as used on Clear
# Linux to avoid having to reconfigure the runtime to look elsewhere.
clr_assets_dir=/usr/share/clear-containers

# Directory to run build and tests in.
#
# An out-of-tree build is used to ensure all necessary files for
# building and testing are distributed and to check srcdir vs builddir
# discrepancies.
if [ -n "$SEMAPHORE_PROJECT_DIR" ]
then
    ci_build_dir="$SEMAPHORE_PROJECT_DIR/ci_build"
else
    ci_build_dir="$TRAVIS_BUILD_DIR/ci_build"
fi

cor_ci_env && mkdir -p "$ci_build_dir" || :
