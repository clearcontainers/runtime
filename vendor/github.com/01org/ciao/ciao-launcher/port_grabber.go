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

const (
	portGrabberStart = 5900
	portGrabberMax   = 6900
)

/*
Just because a port is in the free map doesn't mean it's free.  It could be
used by some other qemu process or otherwise that is not managed by launcher.
In addition, we have the tricky case of restarting launcher after a crash
where there are already running instances.  We could ask those instances for
their spice port and them remove them from the map (and perhaps we will in
the future), but there will still be a race condition, if a new start command
comes in while we query the domain socket of running instances.

To summarize, we need to try to detect the fact that qemu has failed due to
an in-use port and restart it with a new port.  We could try to detect whether
a port was in use or not, a la libvirt, but there'd still be a race condition.
*/

type portGrabber struct {
	sync.Mutex
	free map[int]struct{}
}

var uiPortGrabber = portGrabber{}

func init() {
	uiPortGrabber.free = make(map[int]struct{})
	for i := portGrabberStart; i < portGrabberMax; i++ {
		uiPortGrabber.free[i] = struct{}{}
	}
}

func (pg *portGrabber) grabPort() int {
	port := 0

	pg.Lock()
	glog.Infof("Ports available %d", len(pg.free))
	for key := range pg.free {
		port = key
		break
	}

	if port != 0 {
		delete(pg.free, port)
		glog.Infof("Grabbing port: %d", port)
	}
	pg.Unlock()

	return port
}

func (pg *portGrabber) releasePort(port int) {
	glog.Infof("Releasing port: %d", port)

	if port < portGrabberStart || port >= portGrabberMax {
		glog.Warningf("Unable to release invalid port number %d", port)
		return
	}

	pg.Lock()
	pg.free[port] = struct{}{}
	glog.Infof("Ports available %d", len(pg.free))
	pg.Unlock()
}
