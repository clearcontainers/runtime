// Copyright (c) 2017 Intel Corporation
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

// Description: The true virtcontainers implementation of the RVC interface.
// This indirection is required to allow an alternative implemenation to be
// used for testing purposes.

package main

import (
	"syscall"

	"github.com/Sirupsen/logrus"
	vc "github.com/containers/virtcontainers"
)

// virtcontainers implementation type
type vcImpl struct {
}

func (impl *vcImpl) SetLogger(logger logrus.FieldLogger) {
	vc.SetLogger(logger)
}

func (impl *vcImpl) CreatePod(podConfig vc.PodConfig) (*vc.Pod, error) {
	return vc.CreatePod(podConfig)
}

func (impl *vcImpl) DeletePod(podID string) (*vc.Pod, error) {
	return vc.DeletePod(podID)
}

func (impl *vcImpl) StartPod(podID string) (*vc.Pod, error) {
	return vc.StartPod(podID)
}

func (impl *vcImpl) StopPod(podID string) (*vc.Pod, error) {
	return vc.StopPod(podID)
}

func (impl *vcImpl) RunPod(podConfig vc.PodConfig) (*vc.Pod, error) {
	return vc.RunPod(podConfig)
}

func (impl *vcImpl) ListPod() ([]vc.PodStatus, error) {
	return vc.ListPod()
}

func (impl *vcImpl) StatusPod(podID string) (vc.PodStatus, error) {
	return vc.StatusPod(podID)
}

func (impl *vcImpl) PausePod(podID string) (*vc.Pod, error) {
	return vc.PausePod(podID)
}

func (impl *vcImpl) ResumePod(podID string) (*vc.Pod, error) {
	return vc.ResumePod(podID)
}

func (impl *vcImpl) CreateContainer(podID string, containerConfig vc.ContainerConfig) (*vc.Pod, *vc.Container, error) {
	return vc.CreateContainer(podID, containerConfig)
}

func (impl *vcImpl) DeleteContainer(podID, containerID string) (*vc.Container, error) {
	return vc.DeleteContainer(podID, containerID)
}

func (impl *vcImpl) StartContainer(podID, containerID string) (*vc.Container, error) {
	return vc.StartContainer(podID, containerID)
}

func (impl *vcImpl) StopContainer(podID, containerID string) (*vc.Container, error) {
	return vc.StopContainer(podID, containerID)
}

func (impl *vcImpl) EnterContainer(podID, containerID string, cmd vc.Cmd) (*vc.Pod, *vc.Container, *vc.Process, error) {
	return vc.EnterContainer(podID, containerID, cmd)
}

func (impl *vcImpl) StatusContainer(podID, containerID string) (vc.ContainerStatus, error) {
	return vc.StatusContainer(podID, containerID)
}

func (impl *vcImpl) KillContainer(podID, containerID string, signal syscall.Signal, all bool) error {
	return vc.KillContainer(podID, containerID, signal, all)
}
