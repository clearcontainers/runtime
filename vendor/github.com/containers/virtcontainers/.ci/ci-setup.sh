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

echo "Install chronic"
sudo apt-get install -y moreutils

echo "Install curl"
chronic sudo apt-get install -y curl

echo "Add clear containers sources to apt list"
sudo sh -c "echo 'deb http://download.opensuse.org/repositories/home:/clearlinux:/preview:/clear-containers-2.1/xUbuntu_16.10/ /' >> /etc/apt/sources.list.d/cc-oci-runtime.list"

echo "Update apt repositories"
chronic sudo apt-get update

echo "Install linux-container kernel"
chronic sudo apt-get install -y --force-yes linux-container

echo "Install qemu-lite binary"
chronic sudo apt-get install -y --force-yes qemu-lite

echo "Download clear containers image"
clear_linux_base_url="https://download.clearlinux.org"
latest_version=$(curl -sL ${clear_linux_base_url}/latest)
clear_linux_current_path="${clear_linux_base_url}/current"
containers_img="clear-${latest_version}-containers.img"
compressed_containers_img="${containers_img}.xz"
chronic curl -LO "${clear_linux_current_path}/${compressed_containers_img}"

echo "Validate clear containers image checksum"
compressed_signed_cont_img="${compressed_containers_img}-SHA512SUMS"
chronic curl -LO "${clear_linux_current_path}/${compressed_signed_cont_img}"
chronic sha512sum -c ${compressed_signed_cont_img}

echo "Extract clear containers image"
chronic unxz ${compressed_containers_img}

cc_img_path="/usr/share/clear-containers"
cc_img_link_name="clear-containers.img"
chronic sudo mkdir -p ${cc_img_path}
echo "Install clear containers image"
chronic sudo install --owner root --group root --mode 0644 ${containers_img} ${cc_img_path}

echo -e "Create symbolic link ${cc_img_path}/${cc_img_link_name}"
chronic sudo ln -fs ${cc_img_path}/${containers_img} ${cc_img_path}/${cc_img_link_name}

echo "Setup virtcontainers environment"
chronic sudo -E bash utils/virtcontainers-setup.sh

echo "Install virtcontainers"
chronic make
chronic sudo make install
