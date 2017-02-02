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
	"sync"

	"github.com/golang/glog"
)

type imageStats struct {
	done      chan struct{}
	minSizeMB int
	err       error
}

var imagesMap struct {
	sync.Mutex
	images map[string]*imageStats
}

func init() {
	imagesMap.images = make(map[string]*imageStats)
}

type imageInspector interface {
	imageInfo(imagePath string) (int, error)
}

// Originally this was supposed to be a generic
// feature which could be used by any virtualisation technology.  However, since
// we currently only support QEMU and docker and docker doesn't have a way to
// set disk quotas on the rootfs, this feature is only used by QEMU, hence the
// qemu parameter.  In the future we might add imageInfo back to the virtualiser
// interface.

func getMinImageSize(vm imageInspector, imagePath string) (minSizeMB int, err error) {
	imagesMap.Lock()
	info := imagesMap.images[imagePath]
	if info == nil {
		info = &imageStats{
			done:      make(chan struct{}),
			minSizeMB: -1,
		}
		imagesMap.images[imagePath] = info
		imagesMap.Unlock()

		info.minSizeMB, info.err = vm.imageInfo(imagePath)

		glog.Infof("Min image size of %s = %d", imagePath, info.minSizeMB)
		close(info.done)
	} else {
		imagesMap.Unlock()

		<-info.done
	}

	return info.minSizeMB, info.err
}
