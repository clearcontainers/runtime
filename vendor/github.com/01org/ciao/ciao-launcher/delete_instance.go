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
	"os"

	"github.com/golang/glog"
)

func deleteVnic(instanceDir string, conn serverConn) {
	cfg, err := loadVMConfig(instanceDir)
	if err != nil {
		glog.Warningf("Unable to load instance state %s: %s", instanceDir, err)
		return
	}

	vnicCfg, err := createVnicCfg(cfg)
	if err != nil {
		glog.Warningf("Unable to create vnicCfg: %s", err)
		return
	}

	err = destroyVnic(conn, vnicCfg)
	if err != nil {
		glog.Warningf("Unable to destroy vnic: %s", err)
	}
}

func processDelete(vm virtualizer, instanceDir string, conn serverConn, running ovsRunningState) error {

	// We have to ignore these errors for the time being.  There's no way to distinguish
	// between the various sort of errors that docker can return.  We could be getting
	// a container not found error, if someone had deleted the container manually.  In this
	// case we definitely want to delete the instance.

	_ = vm.deleteImage()

	if networking && running != ovsPending {
		glog.Info("Deleting Vnic")
		deleteVnic(instanceDir, conn)
	}

	err := os.RemoveAll(instanceDir)
	if err != nil {
		glog.Warningf("Unable to remove instance dir: %v", err)
	}

	return err
}
