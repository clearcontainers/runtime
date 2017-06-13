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

echo "Retrieve CNI plugins repository"
go get -d github.com/containernetworking/plugins || true
pushd $GOPATH/src/github.com/containernetworking/plugins

echo "Build CNI plugins"
./build.sh

echo "Install CNI binaries"
cni_bin_path="/opt/cni/bin"
sudo mkdir -p ${cni_bin_path}
sudo cp bin/* ${cni_bin_path}/

echo "Configure CNI"
cni_net_config_path="/etc/cni/net.d"
sudo mkdir -p ${cni_net_config_path}

sudo sh -c 'cat >/etc/cni/net.d/10-mynet.conf <<-EOF
{
    "cniVersion": "0.3.0",
    "name": "mynet",
    "type": "bridge",
    "bridge": "cni0",
    "isGateway": true,
    "ipMasq": true,
    "ipam": {
        "type": "host-local",
        "subnet": "10.88.0.0/16",
        "routes": [
            { "dst": "0.0.0.0/0"  }
        ]
    }
}
EOF'

sudo sh -c 'cat >/etc/cni/net.d/99-loopback.conf <<-EOF
{
    "cniVersion": "0.3.0",
    "type": "loopback"
}
EOF'

popd
