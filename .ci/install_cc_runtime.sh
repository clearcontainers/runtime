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

echo "Retrieve cc-runtime repository"
go get -d github.com/clearcontainers/runtime
pushd $GOPATH/src/github.com/clearcontainers/runtime

echo "Test cc-runtime"
make check

echo "Build cc-runtime"
make

echo "Install cc-runtime"
sudo make install

echo -e "Add cc-runtime as a new/default Docker runtime. Docker version \"$(docker --version)\" could change according to Semaphore CI updates."
sudo mkdir -p /etc/default
cat << EOF | sudo tee /etc/default/docker
DOCKER_OPTS="-D --add-runtime cor=/usr/local/bin/cc-runtime --default-runtime=cor"
EOF

echo "Restart docker service"
sudo service docker stop
sudo service docker start

config_path="/etc/clear-containers"
config_file="configuration.toml"
echo -e "Install cc-runtime ${config_file} to ${config_path}"
sudo mkdir -p ${config_path}
sed 's/^#\(\[runtime\]\|global_log_path =\)/\1/g' config/${config_file} | sudo tee ${config_path}/${config_file}

echo "Install cc-proxy service (/etc/init/cc-proxy.conf)"
upstart_services_path="/etc/init"
sudo cp .ci/upstart-services/cc-proxy.conf ${upstart_services_path}/

echo "Install crio service (/etc/init/crio.conf)"
sudo cp .ci/upstart-services/crio.conf ${upstart_services_path}/

bash .ci/install_cc_env_ubuntu.sh

bash .ci/install_cc_proxy.sh

bash .ci/install_cc_shim.sh

bash .ci/install_virtcontainers.sh

bash .ci/install_cni_plugins.sh

bash .ci/install_crio.sh

bash .ci/install_cc_tests.sh

echo "Install CRI-O bats"
sudo cp .ci/tests/crio.bats $GOPATH/src/github.com/kubernetes-incubator/cri-o/test/

popd

echo "Start cc-proxy service"
sudo service cc-proxy start

echo "Start crio service"
sudo service crio start
