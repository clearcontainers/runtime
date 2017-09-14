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
#   CentOS 7 system.
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
pkgs+=" gettext-devel"
pkgs+=" libtool-ltdl-devel"
pkgs+=" libtool-ltdl"
pkgs+=" glib2-devel"
pkgs+=" bzip2"
pkgs+=" m4"

# for yum-config-manager
pkgs+=" yum-utils"

# runtime dependencies
pkgs+=" libuuid-devel"
pkgs+=" libmnl"
pkgs+=" ${mnl_dev_pkg}"
pkgs+=" libffi-devel"
pkgs+=" pcre-devel"

# qemu lite dependencies
pkgs+=" libattr-devel"
pkgs+=" libcap-devel"
pkgs+=" libcap-ng-devel"
pkgs+=" pixman-devel"

pkgs+=" gcc-c++"

source /etc/os-release

major_version=$(echo "${VERSION_ID}"|cut -d\. -f1)

if [ "${os_distribution}" = centos ]
then
    distro="CentOS"
else
    echo >&2 "ERROR: Unrecognised distribution: ${os_distribution}"
    echo >&2 "ERROR: This script is designed to work on CentOS systems only."
    exit 1
fi

site="http://download.opensuse.org"
dir="repositories/home:/clearcontainers:/clear-containers-3/${distro}_${major_version}"
repo_file="home:clearcontainers:clear-containers-3.repo"
cc_repo_url="${site}/${dir}/${repo_file}"

sudo yum -y update
eval sudo yum -y install "${pkgs}"

sudo yum groupinstall -y 'Development Tools'

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

# Install json-glib
json_glib_setup

# Install gcc
gcc_setup

# Install qemu-lite
qemu_lite_setup

popd

# Install docker
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum -y install docker-ce

# Install Clear Containers components and their dependencies
sudo yum-config-manager --add-repo "${cc_repo_url}"
sudo yum -y install cc-runtime cc-proxy cc-shim linux-container clear-containers-image

# Override runtime configuration to use hypervisor from prefix_dir
# rather than the OBS default values.
sudo -E prefix_dir="${prefix_dir}" sed -i -e \
    "s,^path = \"/usr/bin/qemu-system-x86_64\",path = \"${prefix_dir}/bin/qemu-system-x86_64\",g" \
    /etc/clear-containers/configuration.toml

# Configure CC by default
service_dir="/etc/systemd/system/docker.service.d"
sudo mkdir -p "${service_dir}"
cat <<EOF|sudo tee "${service_dir}/clear-containers.conf"
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd -D --add-runtime cc-runtime=/usr/bin/cc-runtime --default-runtime=cc-runtime
EOF

sudo systemctl daemon-reload
sudo systemctl restart docker
sudo systemctl enable cc-proxy.socket
sudo systemctl start cc-proxy.socket
