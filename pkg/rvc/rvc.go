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

// Description: This file introduces an interface any
// virtcontainers-compatible implementations must conform to.
// It is used to allow the underlying virtcontainers package to be
// switched between the official package and any number
// of other implementations for testing purposes.

package rvc

import (
	"syscall"

	"github.com/Sirupsen/logrus"

	// All implementations need to manipulate the official types
	vc "github.com/containers/virtcontainers"
)

// RVC is a Runtime Virtcontainers implementation
type RVC interface {
	SetLogger(logger logrus.FieldLogger)

	CreatePod(podConfig vc.PodConfig) (*vc.Pod, error)
	DeletePod(podID string) (*vc.Pod, error)
	StartPod(podID string) (*vc.Pod, error)
	StopPod(podID string) (*vc.Pod, error)
	RunPod(podConfig vc.PodConfig) (*vc.Pod, error)
	ListPod() ([]vc.PodStatus, error)
	StatusPod(podID string) (vc.PodStatus, error)
	PausePod(podID string) (*vc.Pod, error)
	ResumePod(podID string) (*vc.Pod, error)

	CreateContainer(podID string, containerConfig vc.ContainerConfig) (*vc.Pod, *vc.Container, error)
	DeleteContainer(podID, containerID string) (*vc.Container, error)
	StartContainer(podID, containerID string) (*vc.Container, error)
	StopContainer(podID, containerID string) (*vc.Container, error)
	EnterContainer(podID, containerID string, cmd vc.Cmd) (*vc.Pod, *vc.Container, *vc.Process, error)
	StatusContainer(podID, containerID string) (vc.ContainerStatus, error)
	KillContainer(podID, containerID string, signal syscall.Signal, all bool) error
}
