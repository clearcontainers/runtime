// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// +build debug

package main

import "flag"

var launchWithUI uiFlag = "nc"
var qemuVirtualisation qemuVirtualisationFlag = "kvm"

func init() {
	flag.Var(&launchWithUI, "with-ui", "Enables virtual consoles on VM instances.  Can be 'none', 'spice', 'nc'")
	flag.Var(&qemuVirtualisation, "qemu-virtualisation", "QEMU virtualisation method. Can be 'kvm', 'auto' or 'software'")
}
