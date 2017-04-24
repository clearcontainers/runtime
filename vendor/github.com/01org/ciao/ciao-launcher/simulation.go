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
	"math/rand"
	"sync"
	"time"

	"github.com/golang/glog"
)

type simulation struct {
	uuid        string
	instanceDir string

	closedCh    chan struct{}
	connectedCh chan struct{}
	killCh      chan struct{}
	monitorCh   chan interface{}
	wg          *sync.WaitGroup

	cpus int
	mem  int
	disk int
}

func (s *simulation) init(cfg *vmConfig, instanceDir string) {
	s.cpus = cfg.Cpus
	s.mem = cfg.Mem
	s.disk = cfg.Disk
	s.instanceDir = instanceDir
}

func (s *simulation) checkBackingImage() error {
	return nil
}

func (s *simulation) downloadBackingImage() error {
	return nil
}

func (s *simulation) createImage(bridge string, userData, metaData []byte) error {
	return nil
}

func (s *simulation) deleteImage() error {
	return nil
}

func fakeVM(s *simulation) {
	glog.Infof("fakeVM started")
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	delay := r.Int63n(1000)
	delay++
	glog.Infof("Will start in %d milliseconds", delay)

	ticker := time.NewTicker(time.Duration(delay) * time.Millisecond)
VM:
	for {
		select {
		case cmd, ok := <-s.monitorCh:
			if !ok {
				s.monitorCh = nil
				break VM
			}
			if _, stopCmd := cmd.(virtualizerStopCmd); stopCmd {
				break VM
			}
		case <-s.killCh:
			break VM
		case <-ticker.C:
			ticker.Stop()
			close(s.connectedCh)
			ticker.C = nil
		}
	}

	if s.wg != nil {
		s.wg.Done()
	}

}

func (s *simulation) startVM(vnicName, ipAddress, cephID string) error {
	glog.Infof("startVM\n")

	s.killCh = make(chan struct{})

	return nil
}

func (s *simulation) monitorVM(closedCh chan struct{}, connectedCh chan struct{}, wg *sync.WaitGroup, boot bool) chan interface{} {
	glog.Infof("monitorVM\n")
	s.closedCh = closedCh
	s.connectedCh = connectedCh
	s.wg = wg

	s.monitorCh = make(chan interface{})

	go fakeVM(s)

	return s.monitorCh
}

func (s *simulation) stats() (disk, memory, cpu int) {
	return s.disk / 10, s.mem / 10, s.cpus / 10
}

func (s *simulation) connected() {
	glog.Infof("connected\n")
}

func (s *simulation) lostVM() {
	glog.Infof("simulation: lostVM\n")
}
