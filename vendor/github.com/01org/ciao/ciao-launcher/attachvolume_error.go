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
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/golang/glog"
)

type attachVolumeError struct {
	err  error
	code payloads.AttachVolumeFailureReason
}

func (ave *attachVolumeError) send(conn serverConn, instance, volume string) {
	if !conn.isConnected() {
		return
	}

	payload, err := generateAttachVolumeError(instance, volume, ave)
	if err != nil {
		glog.Errorf("Unable to generate payload for attach_volume_failure: %v", err)
		return
	}

	_, err = conn.SendError(ssntp.AttachVolumeFailure, payload)
	if err != nil {
		glog.Errorf("Unable to send attach_volume_failure: %v", err)
	}
}
