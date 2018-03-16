#!/bin/bash
#
# Copyright (c) 2018 Intel Corporation
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

# This script will allow to vendor kata-containers/runtime/virtcontainers

set -e

virtcontainers_repo="github.com/kata-containers/runtime"
clearcontainers_runtime_repo="github.com/clearcontainers/runtime"

function get_repo(){
	go get -u ${virtcontainers_repo} || true
}

function install_dep(){
	go get -u github.com/golang/dep/cmd/dep
}

function update_repo(){
	if [ "${ghprbAuthorRepoGitUrl}" ] && [ "${ghprbActualCommit}" ]
	then
		repo="$1"
		if [ ! -d "${GOPATH}/src/${repo}" ]; then
			go get -d "$repo" || true
		fi

		pushd "${GOPATH}/src/${repo}"

		# Update Gopkg.toml
		cat >> Gopkg.toml <<EOF

[[override]]
  name = "${virtcontainers_repo}"
  source = "${ghprbAuthorRepoGitUrl}"
  revision = "${ghprbActualCommit}"
EOF

		# Update the whole vendoring
		dep ensure && dep ensure -update "${virtcontainers_repo}" && dep prune

		popd
	fi
}

get_repo
install_dep
update_repo "${clearcontainers_runtime_repo}"
