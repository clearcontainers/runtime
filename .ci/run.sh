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

# This script will execute the Clear Containers Test Suite. 

set -e

# Current docker tests:
# TODO: Move these docker commands to more formal tests in 
# `clearcontainers/tests` repository. See issue: 
# https://github.com/clearcontainers/tests/issues/59
container_id=$(sudo docker create busybox /bin/sleep 60)
sudo docker ps -a
sudo docker start ${container_id}
sudo docker ps -a
sudo docker exec ${container_id} echo hello
sudo docker ps -a

# try to unpause a container that isn't paused a few times
tmpfile=$(mktemp)
for i in 1 2 3
do
	{ sudo docker unpause ${container_id} &>"$tmpfile"; ret=$?; } || true
	[ $ret -eq 0 ] && { cat "$tmpfile"; exit 1; }
done

sudo docker pause ${container_id}
sudo docker inspect ${container_id}|grep -qi "\"Paused\":.*true"

# try to pause an already paused container a few times
for i in 1 2 3
do
	{ sudo docker pause ${container_id} &>"$tmpfile"; ret=$?; } || true
	[ $ret -eq 0 ] && { cat "$tmpfile"; exit 1; }
done

rm -f "$tmpfile"

sudo docker unpause ${container_id}
sudo docker inspect ${container_id}|grep -qi "\"Paused\":.*false"

sudo docker ps -a
sudo docker stop ${container_id}
sudo docker ps -a
sudo docker rm ${container_id}
sudo docker ps -a

# Execute the tests under `clearcontainers/tests` repository.
test_repo="github.com/clearcontainers/tests"
cd "${GOPATH}/src/${test_repo}"
sudo -E PATH=$PATH make check
