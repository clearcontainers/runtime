#!/bin/bash
#
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

script_dir=$(dirname "$0")

source "${script_dir}/../versions.txt"
from_commit="$1"
new_release="$2"

usage(){
	echo "Usage $0 <commit/tag> [new-release-version]"
	echo "commit/tag: will be used as start point to get release notes"
	echo "new-release: new release version that will have the "
	exit -1
}

if [ -z "$from_commit" ]; then
	usage
fi

git fetch --tags

if [ -z "$new_release" ] ;then
	runtime_version=$(cat "${script_dir}/../VERSION")
else
	runtime_version="${new_release}"
fi

if git describe --exact-match --tags HEAD 2> /dev/null ;then 
	commit_id="$(git describe --exact-match --tags HEAD)"
else
	commit_id="$(git rev-parse HEAD)"
fi



changes(){
	echo "## Changes"
	echo "**FIXME - massage this section by hand to produce a summary please**"
	git log --merges  "$from_commit"..HEAD  | awk '/Merge pull/{getline; getline;print }'  | \
		while read -r pr
		do
			echo "- ${pr}"
		done

	echo ""

	echo "## Shortlog"
	for cr in  $(git log --merges  "$from_commit"..HEAD  | grep 'Merge:' | awk '{print $2".."$3}');
	do
		git log --oneline "$cr"
	done
}

limitations(){
	 grep -P '^###\s|^####\s|See issue' "${script_dir}/../docs/limitations.md"
}

cat << EOT
# Release ${runtime_version}

$(changes)

## Compatibility with Docker
Clear Containers ${runtime_version} is compatible with Docker ${docker_version}
## OCI Runtime Specification
Clear Containers ${runtime_version} support the OCI Runtime Specification [${oci_spec_version}][ocispec]

## Clear Linux Containers image
Clear Containers ${runtime_version} requires at least Clear Linux containers image [${clear_vm_image_version}][clearlinuximage]

## Clear Linux Containers Kernel
Clear Containers ${runtime_version} requires at least Clear Linux Containers  kernel [${clear_container_kernel}][kernel]

## Installation
- [Ubuntu][ubuntu]                                         
- [Fedora][fedora]                                         
- [Developers][developers]                                         


## Issues & limitations

$(limitations)
More information [Limitations][limitations]

[clearlinuximage]: https://download.clearlinux.org/releases/${clear_vm_image_version}/clear/clear-${clear_vm_image_version}-containers.img.xz
[kernel]: https://github.com/clearcontainers/linux/tree/${clear_container_kernel}
[ocispec]: https://github.com/opencontainers/runtime-spec/releases/tag/${oci_spec_version}
[limitations]: https://github.com/clearcontainers/runtime/blob/${commit_id}/docs/limitations.md
[ubuntu]: https://github.com/clearcontainers/runtime/blob/${commit_id}/docs/ubuntu-installation-guide.md
[fedora]: https://github.com/clearcontainers/runtime/blob/${commit_id}/docs/fedora-installation-guide.md
[developers]: https://github.com/clearcontainers/runtime/blob/${commit_id}/docs/developers-clear-containers-install.md

EOT
