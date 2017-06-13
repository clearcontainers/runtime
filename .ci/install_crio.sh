# Copyright (c) 2017 Intel Corporation
#
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
#
#!/bin/bash

set -e

echo "Install dependencies: libglib2.0-dev libseccomp-dev libapparmor-dev libgpgme11-dev libdevmapper-dev btrfs-tools"
sudo apt-get install -y libglib2.0-dev libseccomp-dev libapparmor-dev libgpgme11-dev libdevmapper-dev btrfs-tools

echo "Install BATS from sources"
go get -d github.com/sstephenson/bats || true
pushd $GOPATH/src/github.com/sstephenson/bats
sudo -E PATH=$PATH sh -c "./install.sh /usr"
popd

echo "Get CRI-O sources"
go get -d github.com/kubernetes-incubator/cri-o || true
pushd $GOPATH/src/github.com/kubernetes-incubator/cri-o

echo "Installing CRI-O"
make install.tools
make
sudo make install
sudo make install.config

echo "Setup cc-runtime as the runtime to use"
sudo sed -i.bak 's/\/usr\/bin\/runc/\/usr\/local\/bin\/cc-runtime/g' /etc/crio/crio.conf

echo "Setup aufs as the storage driver"
sudo sed -i.bak 's/storage_driver = \"\"/storage_driver = \"aufs\"/g' /etc/crio/crio.conf

popd
