#!/bin/bash

#  This file is part of cc-runtime.
#
#  Copyright (C) 2018 Intel Corporation
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

# Description: This script installs Clear Containers on a RHEL system.

# enable tracing
set -x

SCRIPT_PATH=$(dirname "$(readlink -f "$0")")
source "${SCRIPT_PATH}/installation-setup.sh"
source "${SCRIPT_PATH}/../versions.txt"

# List of packages to install to satisfy build dependencies
pkgs=""

# general
pkgs+=" bc"
pkgs+=" curl"
pkgs+=" yum-utils"

# qemu lite dependencies
pkgs+=" libattr-devel"
pkgs+=" libcap-devel"
pkgs+=" libcap-ng-devel"
pkgs+=" pixman-devel"
pkgs+=" glib2-devel"
pkgs+=" zlib-devel"

source /etc/os-release

major_version=$(echo "${VERSION_ID}"|cut -d\. -f1)

# Check that the system is RHEL

if [ "${os_distribution}" = rhel ]; then
	distro="RHEL"
else
	echo >&2 "ERROR: Unrecognised distribution: ${os_distribution}."
	echo >&2 "ERROR: This script is designed to work on RHEL systems only."
	exit 1
fi

sudo yum -y update
eval sudo yum -y install "${pkgs}"

# Check that Docker is installed in the system
docker --version
if [ $? -ne 0 ]; then
	echo >&2 "ERROR: Docker-EE or Docker-CE should be installed in the system."
	# Here the required repository will be enabled in order to have Docker
	subscription-manager repos --enable=rhel-${rhel_devtoolset_version}-server-extras-rpms
	if [ $? -ne 0 ]; then
		echo >&2 "ERROR: Server extras rpms repository can not be enabled."
		exit 1
	fi
	sudo yum -y install docker && systemctl enable --now docker
fi

# Check that Golang is installed in the system
go version
if [ $? -ne 0 ]; then
	echo >&2 "ERROR: Golang should be installed in the system."
	# Here the required repository will be enabled in order to have Golang
	subscription-manager repos --enable rhel-${rhel_devtoolset_version}-server-optional-rpms
	if [ $? -ne 0 ]; then
		echo >&2 "ERROR: Server optional rpms repository can not be enabled."
		exit 1
	fi
	sudo yum -y install golang
fi

# Check that GCC is installed in the system
gcc --version
if [ $? -ne 0 ]; then
	echo >&2 "ERROR: GCC should be installed in the system."
	sudo -y install gcc
	if [ $? -ne 0 ]; then
		echo >&2 "ERROR: GCC can not be installed in the system".
		exit 1
	fi
fi

# Check that GCC is greater or equal to version 6.1

current_gcc_version=$(gcc -dumpversion| cut -d '.' -f1-2)

if [ $(echo "${current_gcc_version} < ${rhel_required_gcc_version}" | bc -l) -ne 0 ]; then
	echo >&2 "GCC version should be equal or greater than 6.1."
	# Here the required repositories will be enabled in order to have GCC greater or equal to version 6.1
	subscription-manager repos --enable rhel-${rhel_devtoolset_version}-server-optional-rpms
	if [ $? -ne 0 ]; then
		echo >&2 "ERROR: Server optional rpms repository can not be enabled."
		exit 1
	fi
	subscription-manager repos --enable rhel-server-rhscl-${rhel_devtoolset_version}-rpms
	if [ $? -ne 0 ]; then
		echo >&2 "ERROR: Server rhscl rpms repository can not be enabled."
		exit 1
	fi
	# Install GCC version equal or greater than version 6.1
	sudo yum -y install devtoolset-${rhel_devtoolset_version}
	# Add DTS to our environment
	source scl_source enable devtoolset-${rhel_devtoolset_version}
fi

# Install qemu-lite
qemu_lite_setup

# Install Clear Containers components and their dependencies
site="http://download.opensuse.org"
dir="repositories/home:/clearcontainers:/clear-containers-3/${distro}_${major_version}"
repo_file="home:clearcontainers:clear-containers-3.repo"
cc_repo_url="${site}/${dir}/${repo_file}"

sudo yum-config-manager --add-repo "${cc_repo_url}"
sudo yum -y install cc-runtime cc-proxy cc-shim linux-container clear-containers-image

# Override runtime configuration to use hypervisor from prefix_dir
# rather than the OBS default values.
sudo sed -i -e \
	"s,^path = \"/usr/bin/qemu-system-x86_64\",path = \"/usr/local/bin/qemu-system-x86_64\",g" \
	/usr/share/defaults/clear-containers/configuration.toml

# Configure CC by default
service_dir="/etc/systemd/system/docker.service.d"
sudo mkdir -p "${service_dir}"
cat <<EOF|sudo tee "${service_dir}/clear-containers.conf"
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd -D --add-runtime cc-runtime=/usr/bin/cc-runtime --default-runtime=cc-runtime
EOF

# Remove package oci-systemd-hook
sudo rpm -e --nodeps oci-systemd-hook

sudo systemctl daemon-reload
sudo systemctl enable docker.service
sudo systemctl restart docker
