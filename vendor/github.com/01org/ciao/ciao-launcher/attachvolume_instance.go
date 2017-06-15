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

func processAttachVolume(storageDriver storage.BlockDriver, monitorCh chan interface{}, cfg *vmConfig,
	instance, instanceDir, volumeUUID string, conn serverConn) *attachVolumeError {

	if cfg.Container {
		attachErr := &attachVolumeError{nil, payloads.AttachVolumeNotSupported}
		glog.Errorf("Cannot attach a volume to a container [%s]", string(attachErr.code))
		return attachErr
	}

	if cfg.findVolume(volumeUUID) != nil {
		attachErr := &attachVolumeError{nil, payloads.AttachVolumeAlreadyAttached}
		glog.Errorf("%s is already attached to attach instance %s [%s]",
			volumeUUID, instance, string(attachErr.code))
		return attachErr
	}

	if monitorCh != nil {
		volumeMap, err := storageDriver.GetVolumeMapping()
		if err != nil {
			attachErr := &attachVolumeError{err, payloads.AttachVolumeAttachFailure}
			glog.Errorf("Unable to retrieve list of mapped volumes [%s]: %v",
				string(attachErr.code), err)
			return attachErr
		}

		var devName string

		if len(volumeMap[volumeUUID]) > 0 {
			devName = volumeMap[volumeUUID][0]
			glog.Infof("Volume %s already mapped %s", volumeUUID, devName)
		} else {
			devName, err = storageDriver.MapVolumeToNode(volumeUUID)
			if err != nil {
				attachErr := &attachVolumeError{err, payloads.AttachVolumeAttachFailure}
				glog.Errorf("Unable to map volume  %s [%s]: %v",
					volumeUUID, string(attachErr.code), err)
				return attachErr
			}
			glog.Infof("Mapped instance %s volume %s as %s", instance, volumeUUID, devName)
		}

		responseCh := make(chan error)

		monitorCh <- virtualizerAttachCmd{
			responseCh: responseCh,
			volumeUUID: volumeUUID,
			device:     devName,
		}

		err = <-responseCh
		if err != nil {
			glog.Errorf("Unable to attach volume %s to instance %s: %v",
				volumeUUID, instance, err)
			_ = storageDriver.UnmapVolumeFromNode(devName)
			attachErr := &attachVolumeError{err, payloads.AttachVolumeAttachFailure}
			return attachErr
		}
	}

	cfg.Volumes = append(cfg.Volumes, volumeConfig{UUID: volumeUUID})

	err := cfg.save(instanceDir)
	if err != nil {
		// TODO: should we detach and unmap here?
		cfg.removeVolume(volumeUUID)
		attachErr := &attachVolumeError{err, payloads.AttachVolumeStateFailure}
		glog.Errorf("Unable to persist instance %s state [%s]: %v",
			instance, string(attachErr.code), err)
		return attachErr
	}

	return nil
}
