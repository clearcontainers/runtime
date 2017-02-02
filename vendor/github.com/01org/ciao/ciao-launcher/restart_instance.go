/*
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
*/

package main

import (
	"github.com/01org/ciao/networking/libsnnet"
	"github.com/01org/ciao/payloads"
	"github.com/golang/glog"
)

func processRestart(instanceDir string, vm virtualizer, conn serverConn, cfg *vmConfig) *restartError {
	var vnicName string
	var vnicCfg *libsnnet.VnicConfig
	var err error

	if networking {
		vnicCfg, err = createVnicCfg(cfg)
		if err != nil {
			glog.Errorf("Could not create VnicCFG: %s", err)
			return &restartError{err, payloads.RestartInstanceCorrupt}
		}
		vnicName, _, err = createVnic(conn, vnicCfg)
		if err != nil {
			return &restartError{err, payloads.RestartNetworkFailure}
		}
	}

	err = vm.startVM(vnicName, getNodeIPAddress(), cephID)
	if err != nil {
		return &restartError{err, payloads.RestartLaunchFailure}
	}

	return nil
}
