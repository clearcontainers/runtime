#!/bin/bash

#  This file is part of cc-runtime.
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

# Description: This script installs Clear Containers on a
#   SLES 12.1 system.
#

# all errors are fatal
set -e

# enable tracing
set -x

SCRIPT_PATH=$(dirname "$(readlink -f "$0")")
source "${SCRIPT_PATH}/installation-setup.sh"
source "${SCRIPT_PATH}/../versions.txt"

# List of packages to install to satisfy build dependencies
pkgs=""

mnl_dev_pkg="libmnl-devel"

# general
pkgs+=" zlib-devel"
pkgs+=" libltdl7"
pkgs+=" gettext-tools"
pkgs+=" glib2-devel"

pkgs+=" bzip2"
pkgs+=" m4"

# runtime dependencies
pkgs+=" libuuid-devel"
pkgs+=" libmnl0"
pkgs+=" ${mnl_dev_pkg}"
pkgs+=" libffi48-devel"
pkgs+=" pcre-devel"

# qemu lite dependencies
pkgs+=" libattr-devel"
pkgs+=" libcap-devel"
pkgs+=" libcap-ng-devel"
pkgs+=" libpixman-1-0-devel"

pkgs+=" gcc-c++"

source /etc/os-release

major_version=$(echo "${VERSION}"|cut -d\- -f1)
service_pack=$(echo "${VERSION}"|cut -d\- -f2)
os_distribution=$(echo "${ID}" | cut -d\_ -f1)

if [ "${os_distribution}" = sles ]
then
    distro="SLE"
    if [ "$service_pack" != "" ]
    then
	distro_obs_string="${distro}_${major_version}_${service_pack}"
    else
	distro_obs_string="${distro}_${major_version}"
    fi
else
    echo >&2 "ERROR: Unrecognised distribution: ${os_distribution}"
    echo >&2 "ERROR: This script is designed to work on SLES systems only."
    exit 1
fi

site="http://download.opensuse.org"
dir="repositories/home:/clearcontainers:/clear-containers-3/${distro_obs_string}"
repo_file="home:clearcontainers:clear-containers-3.repo"
cc_repo_url="${site}/${dir}/${repo_file}"

eval sudo -E zypper refresh
eval sudo -E zypper -n install "${pkgs}"

pushd "${deps_dir}"

# Install pre-requisites for gcc
gmp_file="gmp-${gmp_version}.tar.bz2"
curl -L -O "https://gcc.gnu.org/pub/gcc/infrastructure/${gmp_file}"
compile gmp "${gmp_file}" "gmp-${gmp_version}"

mpfr_file="mpfr-${mpfr_version}.tar.bz2"
curl -L -O "https://gcc.gnu.org/pub/gcc/infrastructure/${mpfr_file}"
compile mpfr "${mpfr_file}" "mpfr-${mpfr_version}"

mpc_file="mpc-${mpc_version}.tar.gz"
curl -L -O "https://gcc.gnu.org/pub/gcc/infrastructure/${mpc_file}"
compile mpc "${mpc_file}" "mpc-${mpc_version}"

# Install glib
glib_setup

# Rebuild ldconfig cache
# This allows to json-glib to use the previously installed glib2 library.
sudo rm -f /etc/ld.so.cache
sudo ldconfig

# Install json-glib
json_glib_setup

# Install gcc
gcc_setup

# Install qemu-lite
qemu_lite_setup

popd

# Install Docker
sudo -E zypper -n install docker

# Install Clear Containers components and their dependencies
[ -z "$(zypper repos | grep home_clearcontainers_clear-containers-3)" ] && sudo -E zypper addrepo -r "${cc_repo_url}" || true
curl -OL https://build.opensuse.org/projects/home:clearcontainers:clear-containers-3/public_key
sudo rpmkeys --import public_key
sudo -E zypper -n install cc-runtime cc-proxy cc-shim linux-container clear-containers-image

# Override runtime configuration to use hypervisor from prefix_dir
# rather than the OBS default values.
sudo prefix_dir="${prefix_dir}" sed -i -e \
    "s,^path = \"/usr/bin/qemu-system-x86_64\",path = \"${prefix_dir}/bin/qemu-system-x86_64\",g" \
    /usr/share/defaults/clear-containers/configuration.toml

# Configure CC by default
service_dir="/etc/systemd/system/docker.service.d"
sudo mkdir -p "${service_dir}"
cat <<EOF|sudo -E tee "${service_dir}/clear-containers.conf"
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd -D --containerd /run/containerd/containerd.sock --add-runtime cc-runtime=/usr/bin/cc-runtime --default-runtime=cc-runtime
EOF

sudo systemctl daemon-reload
sudo systemctl restart docker
