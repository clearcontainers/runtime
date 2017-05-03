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

echo "Retrieve virtcontainers repository"
go get -d github.com/containers/virtcontainers
pushd $GOPATH/src/github.com/containers/virtcontainers

pause_bin_name="pause"
echo -e "Build ${pause_bin_name} binary"
make ${pause_bin_name}

pause_bin_path="/var/lib/clearcontainers/runtime/bundles/pause_bundle/bin"
echo -e "Create ${pause_bin_path}"
sudo mkdir -p ${pause_bin_path}

echo -e "Install ${pause_bin_name} binary"
sudo install --owner root --group root --mode 0755 pause/${pause_bin_name} ${pause_bin_path}

popd
