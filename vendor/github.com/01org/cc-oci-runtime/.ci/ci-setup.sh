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

root=$(cd `dirname "$0"`/..; pwd -P)
source "$root/versions.txt"

if [ "$SEMAPHORE" = true ]
then
    # SemaphoreCI has different environments that builds can run in. The
    # default environment does not have docker enabled so it is
    # necessary to specify a docker-enabled build environment on the
    # semaphoreci.com web site.
    #
    # However, currently, the docker-enabled environment does not
    # provide nested KVM (whereas the default environment does), so
    # manually enable nested kvm for the time being.
    sudo rmmod kvm-intel || :
    sudo sh -c "echo 'options kvm-intel nested=y' >> /etc/modprobe.d/dist.conf" || :
    sudo modprobe kvm-intel || :
fi

source $(dirname "$0")/ci-common.sh

if [ "$nested" = "Y" ]
then
    # Ensure the user can access the kvm device
    sudo chmod g+rw /dev/kvm
    sudo chgrp "$USER" /dev/kvm
fi

#
# Install go
#

go_tarball="go${go_version}.linux-amd64.tar.gz"
curl -L -O "https://storage.googleapis.com/golang/$go_tarball"
tar xvf $go_tarball 1>/dev/null
mv go $GOROOT
# Unfortunately, go doesn't support vendoring outside of GOPATH (maybe in 1.8?)
# So, we setup a GOPATH tree with our vendored dependencies.
# See: https://github.com/golang/go/issues/14566
mkdir -p "$GOPATH/src"
cp -r vendor/* "$GOPATH/src"
# We also need to put the runtime into its right place in the GOPATH so we can
# self-import internal packages
mkdir -p "$GOPATH/src/github.com/01org/"
ln -s $PWD "$GOPATH/src/github.com/01org/"

#
# Install cc-oci-runtime dependencies
#

# Ensure "make install" as root can find clang
#
# See: https://github.com/travis-ci/travis-ci/issues/2607
export CC=$(which "$CC")

gnome_dl=https://download.gnome.org/sources

# Install required dependencies to build
# glib, json-glib, libmnl-dev, check, gcc, cc-oci-runtime and qemu-lite

pkgs=""

# general
pkgs+=" pkg-config"
pkgs+=" gettext"
pkgs+=" rpm2cpio"
pkgs+=" valgrind"

# runtime dependencies
pkgs+=" uuid-dev"
pkgs+=" cppcheck"
pkgs+=" libmnl-dev"
pkgs+=" libffi-dev"
pkgs+=" libpcre3-dev"

# runtime + qemu-lite
pkgs+=" zlib1g-dev"

# qemu-lite
pkgs+=" libpixman-1-dev"

# gcc
pkgs+=" libcap-ng-dev"
pkgs+=" libgmp-dev"
pkgs+=" libmpfr-dev"
pkgs+=" libmpc-dev"

# code coverage
pkgs+=" lcov"

# chronic(1)
pkgs+=" moreutils"

# qemu-lite won't be built
# some unit tests need qemu-system
if [ "$nested" != "Y" ]
then
	pkgs+=" qemu-system-x86"
fi

eval sudo apt-get -qq install "$pkgs"

function compile {
	name="$1"
	tarball="$2"
	directory="$3"
	configure_opts="$4"

	chronic tar -xvf ${tarball}
	pushd ${directory}

	if [ -n "$configure_opts" ]
	then
		args="$configure_opts"
	else
		args=""
		args+=" --disable-silent-rules"
		args+=" --prefix=\"${prefix_dir}\""
	fi

	eval CC=${CC:-cc} chronic ./configure "$args"
	chronic make -j5
	chronic sudo make install
	popd
}

# Determined if the specified command has already been installed
function cmd_installed {
    local cmd="$1"

    local path="${prefix_dir}/bin/$cmd"

    [ -e "$path" ]
}

# Determine if the specified library version is available
function lib_installed {
    local name="$1"
    local required_version="$2"

    version=$(pkg-config --print-provides "$name" 2>/dev/null | awk '{print $3}')

    [ "$version" = "$required_version" ]
}

pushd "$deps_dir"

# Build glib
glib_major=`echo $glib_version | cut -d. -f1`
glib_minor=`echo $glib_version | cut -d. -f2`
file="glib-${glib_version}.tar.xz"

if ! lib_installed "glib-2.0" "$glib_version"
then
    if [ ! -e "$file" ]
    then
        curl -L -O "$gnome_dl/glib/${glib_major}.${glib_minor}/$file"
    fi

    compile glib glib-${glib_version}.tar.xz glib-${glib_version}
fi

# Build json-glib
json_major=`echo $json_glib_version | cut -d. -f1`
json_minor=`echo $json_glib_version | cut -d. -f2`
file="json-glib-${json_glib_version}.tar.xz"

if ! lib_installed "json-glib-1.0" "$json_glib_version"
then
    if [ ! -e "$file" ]
    then
        curl -L -O "$gnome_dl/json-glib/${json_major}.${json_minor}/$file"
    fi

    compile json-glib json-glib-${json_glib_version}.tar.xz json-glib-${json_glib_version}
fi

# Build check
# We need to build check as the check version in the OS used by travis isn't
# -pedantic safe.
if ! lib_installed "check" "${check_version}"
then
    file="check-${check_version}.tar.gz"

    if [ ! -e "$file" ]
    then
        curl -L -O "https://github.com/libcheck/check/releases/download/${check_version}/$file"
    fi

    compile check check-${check_version}.tar.gz check-${check_version}
fi

cmd="bats"
if ! cmd_installed "$cmd"
then
    # Install bats
    [ ! -d bats ] && git clone https://github.com/sstephenson/bats.git
    pushd bats
    sudo ./install.sh "$prefix_dir"
    popd
fi

if [ "$nested" != "Y" ]
then
    popd
    exit 0
fi

cmd="gcc"
if ! cmd_installed "$cmd"
then
    # build gcc (required for qemu-lite)
    gcc_dir="gcc-${gcc_version}"
    gcc_site="http://mirrors.kernel.org/gnu/gcc/${gcc_dir}"
    gcc_file="gcc-${gcc_version}.tar.bz2"
    gcc_url="${gcc_site}/${gcc_file}"

    if [ ! -e "$gcc_file" ]
    then
        curl -L -O "$gcc_url"
    fi

    gcc_opts=""
    gcc_opts+=" --enable-languages=c"
    gcc_opts+=" --disable-multilib"
    gcc_opts+=" --disable-libstdcxx"
    gcc_opts+=" --disable-bootstrap"
    gcc_opts+=" --disable-nls"
    gcc_opts+=" --prefix=\"${prefix_dir}\""

    compile gcc "$gcc_file" "$gcc_dir" "$gcc_opts"
fi

# Use built version of gcc
export CC="${prefix_dir}/bin/gcc"

# build qemu-lite
cmd="qemu-system-x86_64"
if ! cmd_installed "$cmd"
then
    qemu_lite_site="https://github.com/01org/qemu-lite/archive/"
    qemu_lite_file="${qemu_lite_version}.tar.gz"
    qemu_lite_url="${qemu_lite_site}/${qemu_lite_file}"
    qemu_lite_dir="qemu-lite-${qemu_lite_version}"

    qemu_lite_opts=""
    qemu_lite_opts+=" --disable-bluez"
    qemu_lite_opts+=" --disable-brlapi"
    qemu_lite_opts+=" --disable-bzip2"
    qemu_lite_opts+=" --disable-curl"
    qemu_lite_opts+=" --disable-curses"
    qemu_lite_opts+=" --disable-debug-tcg"
    qemu_lite_opts+=" --disable-fdt"
    qemu_lite_opts+=" --disable-glusterfs"
    qemu_lite_opts+=" --disable-gtk"
    qemu_lite_opts+=" --disable-libiscsi"
    qemu_lite_opts+=" --disable-libnfs"
    qemu_lite_opts+=" --disable-libssh2"
    qemu_lite_opts+=" --disable-libusb"
    qemu_lite_opts+=" --disable-linux-aio"
    qemu_lite_opts+=" --disable-lzo"
    qemu_lite_opts+=" --disable-opengl"
    qemu_lite_opts+=" --disable-qom-cast-debug"
    qemu_lite_opts+=" --disable-rbd"
    qemu_lite_opts+=" --disable-rdma"
    qemu_lite_opts+=" --disable-sdl"
    qemu_lite_opts+=" --disable-seccomp"
    qemu_lite_opts+=" --disable-slirp"
    qemu_lite_opts+=" --disable-snappy"
    qemu_lite_opts+=" --disable-spice"
    qemu_lite_opts+=" --disable-strip"
    qemu_lite_opts+=" --disable-tcg-interpreter"
    qemu_lite_opts+=" --disable-tcmalloc"
    qemu_lite_opts+=" --disable-tools"
    qemu_lite_opts+=" --disable-tpm"
    qemu_lite_opts+=" --disable-usb-redir"
    qemu_lite_opts+=" --disable-uuid"
    qemu_lite_opts+=" --disable-vnc"
    qemu_lite_opts+=" --disable-vnc-{jpeg,png,sasl}"
    qemu_lite_opts+=" --disable-vte"
    qemu_lite_opts+=" --disable-xen"
    qemu_lite_opts+=" --enable-attr"
    qemu_lite_opts+=" --enable-cap-ng"
    qemu_lite_opts+=" --enable-kvm"
    qemu_lite_opts+=" --enable-virtfs"
    qemu_lite_opts+=" --enable-vhost-net"
    qemu_lite_opts+=" --target-list=x86_64-softmmu"
    qemu_lite_opts+=" --extra-cflags=\"-fno-semantic-interposition -O3 -falign-functions=32\""
    qemu_lite_opts+=" --prefix=\"${prefix_dir}\""
    qemu_lite_opts+=" --datadir=\"${prefix_dir}/share/qemu-lite\""
    qemu_lite_opts+=" --libdir=\"${prefix_dir}/lib64/qemu-lite\""
    qemu_lite_opts+=" --libexecdir=\"${prefix_dir}/libexec/qemu-lite\""

    if [ ! -e "$qemu_lite_file" ]
    then
        curl -L -O "${qemu_lite_url}"
    fi

    compile qemu-lite "$qemu_lite_file" \
        "$qemu_lite_dir" "$qemu_lite_opts"
fi

# install kernel + Clear Containers image
mkdir -p assets
pushd assets
clr_dl_site="https://download.clearlinux.org"
clr_release=$(curl -L "${clr_dl_site}/latest")
clr_kernel_base_url="${clr_dl_site}/releases/${clr_release}/clear/x86_64/os/Packages"

sudo mkdir -p "$clr_assets_dir"

# find newest containers kernel
clr_kernel=$(curl -l -s -L "${clr_kernel_base_url}" |\
    grep -o "linux-container-[0-9][0-9.-]*\.x86_64.rpm" |\
    sort -u)

# download kernel
if [ ! -e "${clr_assets_dir}/${clr_kernel}" ]
then
    if [ ! -e "$clr_kernel" ]
    then
        curl -L -O "${clr_kernel_base_url}/${clr_kernel}"
    fi

    # install kernel
    # (note: cpio on trusty does not support "-D")
    rpm2cpio "${clr_kernel}"| (cd / && sudo cpio -idv)
fi

clr_image_url="${clr_dl_site}/current/clear-${clr_release}-containers.img.xz"
clr_image_compressed=$(basename "$clr_image_url")

# uncompressed image name
clr_image=${clr_image_compressed/.xz/}

# download image
if [ ! -e "${clr_assets_dir}/${clr_image}" ]
then
    for file in "${clr_image_url}-SHA512SUMS" "${clr_image_url}"
    do
        [ ! -e "$file" ] && curl -L -O "$file"
    done

    # verify image
    checksum_file="${clr_image_compressed}-SHA512SUMS"
    sha512sum -c "${checksum_file}"

    # unpack image
    unxz --force "${clr_image_compressed}"

    # install image
    sudo install "${clr_image}" "${clr_assets_dir}"

    rm -f "${checksum_file}" "${clr_image}" "${clr_image_compressed}"
fi

# change kernel+image ownership
sudo chown -R "$USER" "${clr_assets_dir}"

# create image symlink (kernel will already have one)
clr_image_link=clear-containers.img
sudo rm -f "${clr_assets_dir}/${clr_image_link}"
(cd "${clr_assets_dir}" && sudo ln -s "${clr_image}" "${clr_image_link}")

popd

popd

if [ "$SEMAPHORE" = true ]
then
    distro=$(lsb_release -c|awk '{print $2}' || :)
    if [ "$distro" = trusty ]
    then
        # Configure docker to use the runtime
        docker_opts=""
        docker_opts+=" --add-runtime cor=cc-oci-runtime"
        docker_opts+=" --default-runtime=cor"

        sudo initctl stop docker

        # Remove first as apt-get doesn't like downgrading this package
        # on trusty.
        sudo apt-get -qq purge docker-engine

        sudo apt-get -qq install \
            docker-engine="$docker_engine_semaphoreci_ubuntu_version"

        echo "DOCKER_OPTS=\"$docker_opts\"" |\
            sudo tee -a /etc/default/docker

        sudo initctl restart docker

    else
        echo "ERROR: unhandled Semaphore distro: $distro"
        exit 1
    fi
fi
