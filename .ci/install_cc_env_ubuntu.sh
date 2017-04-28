# Copyright (c) 2017 Intel Corporation

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
#

#!/bin/bash

set -e

echo "Add clear containers sources to apt list"
sudo sh -c "echo 'deb http://download.opensuse.org/repositories/home:/clearlinux:/preview:/clear-containers-2.1/xUbuntu_16.10/ /' >> /etc/apt/sources.list.d/cc-oci-runtime.list"

echo "Install chronic"
sudo apt-get install -y moreutils

echo "Install rpm2cpio"
chronic sudo apt-get install -y rpm2cpio

echo "Update apt repositories"
chronic sudo apt-get update

echo "Install linux-container kernel"
chronic sudo apt-get install -y --force-yes linux-container

echo "Install qemu-lite binary"
chronic sudo apt-get install -y --force-yes qemu-lite

echo "Download clear containers image"
latest_version=$(curl -sL https://download.clearlinux.org/latest)
curl -LO "https://download.clearlinux.org/current/clear-${latest_version}-containers.img.xz"

echo "Validate clear containers image checksum"
curl -LO "https://download.clearlinux.org/current/clear-${latest_version}-containers.img.xz-SHA512SUMS"
sha512sum -c clear-${latest_version}-containers.img.xz-SHA512SUMS

echo "Extract clear containers image"
unxz clear-${latest_version}-containers.img.xz

cc_img_path="/usr/share/clear-containers"
cc_img_link_name="clear-containers.img"
sudo mkdir -p ${cc_img_path}
echo "Install clear containers image"
sudo install --owner root --group root --mode 0755 clear-${latest_version}-containers.img ${cc_img_path}

echo -e "Create symbolic link ${cc_img_path}/${cc_img_link_name}"
sudo ln -fs ${cc_img_path}/clear-${latest_version}-containers.img ${cc_img_path}/${cc_img_link_name}

bug_url="https://github.com/clearcontainers/runtime/issues/91"
kernel_version="4.5-50"
cc_kernel_link_name="vmlinux.container"
echo -e "\nWARNING:"
echo "WARNING: Using backlevel kernel version ${kernel_version} due to bug ${bug_url}"
echo -e "WARNING:\n"
echo -e "Install clear containers kernel ${kernel_version}"
curl -LO "https://download.clearlinux.org/releases/12760/clear/x86_64/os/Packages/linux-container-${kernel_version}.x86_64.rpm"
rpm2cpio linux-container-${kernel_version}.x86_64.rpm | cpio -ivdm
sudo install --owner root --group root --mode 0755 .${cc_img_path}/vmlinux-${kernel_version}.container ${cc_img_path}

echo -e "Create symbolic link ${cc_img_path}/${cc_kernel_link_name}"
sudo ln -fs ${cc_img_path}/vmlinux-${kernel_version}.container ${cc_img_path}/${cc_kernel_link_name}
