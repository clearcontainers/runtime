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
	storage "github.com/01org/ciao/ciao-storage"
	"github.com/01org/ciao/payloads"
	"github.com/golang/glog"
)

func processDetachVolume(storageDriver storage.BlockDriver, monitorCh chan interface{}, cfg *vmConfig, instance, instanceDir, volumeUUID string, conn serverConn) *detachVolumeError {

	if cfg.Container {
		detachErr := &detachVolumeError{nil, payloads.DetachVolumeNotSupported}
		glog.Errorf("Cannot detach a volume from a container [%s]", string(detachErr.code))
		return detachErr
	}

	vol := cfg.findVolume(volumeUUID)
	if vol == nil {
		detachErr := &detachVolumeError{nil, payloads.DetachVolumeNotAttached}
		glog.Errorf("%s not attached to attach instance %s [%s]",
			volumeUUID, instance, string(detachErr.code))
		return detachErr
	}

	if vol.Bootable {
		glog.Errorf("Unable to detach volume %s from instance %s. Volume is bootable!", volumeUUID, instance)
		attachErr := &detachVolumeError{nil, payloads.DetachVolumeDetachFailure}
		return attachErr
	}

	if monitorCh != nil {
		responseCh := make(chan error)
		monitorCh <- virtualizerDetachCmd{
			responseCh: responseCh,
			volumeUUID: volumeUUID,
		}

		glog.Infof("Detaching Volume %v", volumeUUID)

		err := <-responseCh
		if err != nil {
			glog.Errorf("Unable to detach volume %s from instance %s", volumeUUID, instance)
			attachErr := &detachVolumeError{err, payloads.DetachVolumeDetachFailure}
			return attachErr
		}
	}

	// May fail if other instances are using the same device.  We'll ignore error for now
	// but we might be able to get good error info out of rbd.
	_ = storageDriver.UnmapVolumeFromNode(volumeUUID)

	oldVols := cfg.Volumes
	cfg.removeVolume(volumeUUID)

	err := cfg.save(instanceDir)
	if err != nil {
		// TODO: What should I do here.  Try to re-attach?
		cfg.Volumes = oldVols
		detachErr := &detachVolumeError{err, payloads.DetachVolumeDetachFailure}
		glog.Errorf("Unable to persist instance %s state [%s]: %v",
			instance, string(detachErr.code), err)
		return detachErr
	}

	return nil
}
