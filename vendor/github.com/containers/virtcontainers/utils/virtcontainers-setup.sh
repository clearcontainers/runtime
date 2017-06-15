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

set -e

if [[ $EUID -ne 0 ]]; then
	echo "This script must be run as root"
	exit 1
fi

tmpdir=$(mktemp -d)
virtcontainers_build_dir="virtcontainers/build"
echo -e "Create temporary build directory ${tmpdir}/${virtcontainers_build_dir}"
mkdir -p ${tmpdir}/${virtcontainers_build_dir}

TMPDIR="/tmp"
OPTDIR="/opt"
ETCDIR="/etc"

echo -e "Create ${TMPDIR}/cni/bin (needed by testing)"
rm -rf ${TMPDIR}/cni/bin
mkdir -p ${TMPDIR}/cni/bin
echo -e "Create cni directories ${OPTDIR}/cni/bin and ${ETCDIR}/cni/net.d"
mkdir -p ${OPTDIR}/cni/bin
mkdir -p ${ETCDIR}/cni/net.d

bundlesdir="${TMPDIR}/bundles"
echo -e "Create bundles in ${bundlesdir}"
rm -rf ${bundlesdir}
busybox_bundle="${bundlesdir}/busybox"
mkdir -p ${busybox_bundle}
docker pull busybox
pushd ${busybox_bundle}
rootfsdir="rootfs"
mkdir ${rootfsdir}
docker export $(docker create busybox) | tar -C ${rootfsdir} -xvf -
echo -e '#!/bin/sh\ncd "\"\n"sh"' > ${rootfsdir}/.containerexec
echo -e 'HOME=/root\nPATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\nTERM=xterm' > ${rootfsdir}/.containerenv
popd
pause_bundle="${bundlesdir}/pause_bundle"
mkdir -p ${pause_bundle}
cp -r ${busybox_bundle}/* ${pause_bundle}/

echo -e "Move to ${tmpdir}/${virtcontainers_build_dir}"
pushd ${tmpdir}/${virtcontainers_build_dir}
echo "Clone cni"
git clone https://github.com/containernetworking/plugins.git

echo "Copy CNI config files"
cp $GOPATH/src/github.com/containers/virtcontainers/test/cni/10-mynet.conf ${ETCDIR}/cni/net.d/
cp $GOPATH/src/github.com/containers/virtcontainers/test/cni/99-loopback.conf ${ETCDIR}/cni/net.d/

echo -e "Build pause binary and copy it to pause bundle (${pause_bundle}/${rootfsdir}/bin)"
pushd $GOPATH/src/github.com/containers/virtcontainers/pause
make
cp pause ${pause_bundle}/${rootfsdir}/bin/
make clean
popd
pushd plugins
./build.sh
cp ./bin/bridge ${TMPDIR}/cni/bin/cni-bridge
cp ./bin/loopback ${TMPDIR}/cni/bin/loopback
cp ./bin/host-local ${TMPDIR}/cni/bin/host-local
popd
popd
cp ${TMPDIR}/cni/bin/* /opt/cni/bin/

rm -rf ${tmpdir}
