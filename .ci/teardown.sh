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

# This script will get any execution log that may be useful for
# debugging any issue related to Clear Containers and clean the
# test environment.

sudo cat /var/lib/clear-containers/runtime/runtime.log
sudo cat /var/log/upstart/cc-proxy.log
sudo cat /var/log/upstart/crio.log
