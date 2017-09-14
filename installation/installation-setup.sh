#!/bin/bash

#  This file is part of cc-oci-runtime.
#
#  Copyright (C) 2017 Intel Corporation
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

os_distribution=`cat /etc/os-release | grep -w ID | cut -d '=' -f2 | sed s/\"//g`

export prefix_dir="/usr/local/"
export GOROOT="$prefix_dir/go"
export deps_dir="$PWD/dependencies"
mkdir -p "$deps_dir"

SCRIPT_PATH=$(dirname "$(readlink -f "$0")")
source "${SCRIPT_PATH}/../versions.txt"

# if chronic(1) is available, it will be used to hide all output
# (unless an error occurs).
chronic=$(command -v chronic || true)

# Ensure "make install" as root can find clang
#
# See: https://github.com/travis-ci/travis-ci/issues/2607
export CC=$(which "$CC")
gnome_dl=https://download.gnome.org/sources

function compile {
	name="$1"
	tarball="$2"
	directory="$3"
	configure_opts="$4"
	eval $chronic tar -xvf "${tarball}"
	pushd ${directory}

	if [ -n "$configure_opts" ]
	then
		args="$configure_opts"
	else
		args=""
		args+=" --disable-silent-rules"
		args+=" --prefix=\"${prefix_dir}\""
	fi

	eval CC=${CC:-cc} "$chronic" ./configure "$args"
	eval "$chronic make -j5"
	eval "$chronic sudo make install"
	popd
}

# Determine if the specified library version is available
# Note that this is an exact match test: this is by design - see the
# top-level "versions.txt" file.
function lib_installed {
	local name="$1"
	local required_version="$2"
	version=$(pkg-config --print-provides "$name" 2>/dev/null | awk '{print $3}')
	[ "$version" = "$required_version" ]
}

# Determined if the specified command has already been installed
function cmd_installed {
	local cmd="$1"
	local path="${prefix_dir}/bin/$cmd"
	[ -e "$path" ]
}

# Build glib
function glib_setup {
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
}

# Build json-glib
function json_glib_setup {
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
}

function gcc_setup {
	cmd="gcc"
	if ! cmd_installed "$cmd"
	then
		# build gcc (required for qemu-lite)
		gcc_dir="gcc-${gcc_version}"
		gcc_site="https://mirrors.kernel.org/gnu/gcc/${gcc_dir}"
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
}

function qemu_lite_setup {
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
}
