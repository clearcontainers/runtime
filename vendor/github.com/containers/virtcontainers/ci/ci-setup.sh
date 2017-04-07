#
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

if [[ $EUID -ne 0 ]]; then
	echo "This script must be run as root"
	exit 1
fi

echo "Create temporary build directory /tmp/virtcontainers/build"
rm -rf /tmp/virtcontainers/build
mkdir -p /tmp/virtcontainers/build
echo "Create /tmp/bin"
rm -rf /tmp/bin
mkdir -p /tmp/bin
echo "Create /tmp/cni/bin"
rm -rf /tmp/cni/bin
mkdir -p /tmp/cni/bin
echo "Create cni directories /opt/cni/bin and /etc/cni/net.d"
mkdir -p /opt/cni/bin
mkdir -p /etc/cni/net.d

echo "Create bundles in /tmp/bundles"
rm -rf /tmp/bundles
mkdir -p /tmp/bundles/busybox
docker pull busybox
pushd /tmp/bundles/busybox/
mkdir rootfs
docker export $(docker create busybox) | tar -C rootfs -xvf -
echo -e '#!/bin/sh\ncd "\"\n"sh"' > rootfs/.containerexec
echo -e 'HOME=/root\nPATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\nTERM=xterm' > rootfs/.containerenv
popd
mkdir -p /tmp/bundles/pause_bundle
cp -r /tmp/bundles/busybox/* /tmp/bundles/pause_bundle/

echo "Move to /tmp/virtcontainers/build"
pushd /tmp/virtcontainers/build
echo "Clone virtcontainers"
git clone https://github.com/containers/virtcontainers.git
echo "Clone cni"
git clone https://github.com/containernetworking/cni.git
echo "Copy CNI config files"
cp virtcontainers/test/cni/10-mynet.conf /etc/cni/net.d/
cp virtcontainers/test/cni/99-loopback.conf /etc/cni/net.d/
echo "Build hook binary and copy it to /tmp/bin"
go build virtcontainers/hook/mock/hook.go
cp hook /tmp/bin/
echo "Build pause binary and copy it to pause bundle (/tmp/bundles/pause_bundle/rootfs/bin)"
pushd virtcontainers/pause
make
cp pause /tmp/bundles/pause_bundle/rootfs/bin/
popd
pushd cni
./build.sh
cp ./bin/bridge /tmp/cni/bin/cni-bridge
cp ./bin/loopback /tmp/cni/bin/loopback
cp ./bin/host-local /tmp/cni/bin/host-local
popd
popd
rm -r /tmp/virtcontainers/build
cp /tmp/cni/bin/* /opt/cni/bin/
