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

// Description: The mock implementation of virtcontainers that implements the RVC interface.
// This implementation provides the following behavioural options for all RVC interface functions:
//
// - calling the official virtcontainers API (default).
// - returning an error in a known format (that can be verified by the tests using isMockError()).
// - calling a custom function for more elaborate scenarios.

package main

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/Sirupsen/logrus"
	vc "github.com/containers/virtcontainers"
)

// mockImpl is a virtcontainers implementation type
type mockImpl struct {
	// cause all interface functions to fail if set
	forceFailure bool

	// Used to override behaviour of particular functions
	// (offering more fine-grained control than forceFailure)
	setLoggerFunc       func(logger logrus.FieldLogger)
	createPodFunc       func(podConfig vc.PodConfig) (*vc.Pod, error)
	deletePodFunc       func(podID string) (*vc.Pod, error)
	startPodFunc        func(podID string) (*vc.Pod, error)
	stopPodFunc         func(podID string) (*vc.Pod, error)
	runPodFunc          func(podConfig vc.PodConfig) (*vc.Pod, error)
	listPodFunc         func() ([]vc.PodStatus, error)
	statusPodFunc       func(podID string) (vc.PodStatus, error)
	pausePodFunc        func(podID string) (*vc.Pod, error)
	resumePodFunc       func(podID string) (*vc.Pod, error)
	createContainerFunc func(podID string, containerConfig vc.ContainerConfig) (*vc.Pod, *vc.Container, error)
	deleteContainerFunc func(podID, containerID string) (*vc.Container, error)
	startContainerFunc  func(podID, containerID string) (*vc.Container, error)
	stopContainerFunc   func(podID, containerID string) (*vc.Container, error)
	enterContainerFunc  func(podID, containerID string, cmd vc.Cmd) (*vc.Pod, *vc.Container, *vc.Process, error)
	statusContainerFunc func(podID, containerID string) (vc.ContainerStatus, error)
	killContainerFunc   func(podID, containerID string, signal syscall.Signal, all bool) error
}

// testingImpl is a concrete mock RVC implementation used for testing
var testingImpl = &mockImpl{}

// mockErrorPrefix is a string that all errors returned by the mock
// implementation itself will contain as a prefix.
const mockErrorPrefix = "mockImpl forced failure"

func init() {
	fmt.Printf("INFO: switching to fake virtcontainers implementation for testing\n")
	vci = testingImpl
}

func (impl *mockImpl) SetLogger(logger logrus.FieldLogger) {
	if impl.setLoggerFunc != nil {
		impl.setLoggerFunc(logger)
		return
	}

	vc.SetLogger(logger)
}

func (impl *mockImpl) CreatePod(podConfig vc.PodConfig) (*vc.Pod, error) {
	if impl.createPodFunc != nil {
		return impl.createPodFunc(podConfig)
	}

	if impl.forceFailure {
		return nil, fmt.Errorf("%s: CreatePod: podConfig: %v", mockErrorPrefix, podConfig)
	}

	return vc.CreatePod(podConfig)
}

func (impl *mockImpl) DeletePod(podID string) (*vc.Pod, error) {
	if impl.deletePodFunc != nil {
		return impl.deletePodFunc(podID)
	}

	if impl.forceFailure {
		return nil, fmt.Errorf("%s: DeletePod: podID: %v", mockErrorPrefix, podID)
	}

	return vc.DeletePod(podID)
}

func (impl *mockImpl) StartPod(podID string) (*vc.Pod, error) {
	if impl.startPodFunc != nil {
		return impl.startPodFunc(podID)
	}

	if impl.forceFailure {
		return nil, fmt.Errorf("%s: StartPod: podID: %v", mockErrorPrefix, podID)
	}

	return vc.StartPod(podID)
}

func (impl *mockImpl) StopPod(podID string) (*vc.Pod, error) {
	if impl.stopPodFunc != nil {
		return impl.stopPodFunc(podID)
	}

	if impl.forceFailure {
		return nil, fmt.Errorf("%s: StopPod: podID: %v", mockErrorPrefix, podID)
	}

	return vc.StopPod(podID)
}

func (impl *mockImpl) RunPod(podConfig vc.PodConfig) (*vc.Pod, error) {
	if impl.runPodFunc != nil {
		return impl.runPodFunc(podConfig)
	}

	if impl.forceFailure {
		return nil, fmt.Errorf("%s: RunPod: podConfig: %v", mockErrorPrefix, podConfig)
	}

	return vc.RunPod(podConfig)
}

func (impl *mockImpl) ListPod() ([]vc.PodStatus, error) {
	if impl.listPodFunc != nil {
		return impl.listPodFunc()
	}

	if impl.forceFailure {
		return nil, fmt.Errorf("%s: ListPod", mockErrorPrefix)
	}

	return vc.ListPod()
}

func (impl *mockImpl) StatusPod(podID string) (vc.PodStatus, error) {
	if impl.statusPodFunc != nil {
		return impl.statusPodFunc(podID)
	}

	if impl.forceFailure {
		return vc.PodStatus{}, fmt.Errorf("%s: StatusPod: podID: %v", mockErrorPrefix, podID)
	}

	return vc.StatusPod(podID)
}

func (impl *mockImpl) PausePod(podID string) (*vc.Pod, error) {
	if impl.pausePodFunc != nil {
		return impl.pausePodFunc(podID)
	}

	if impl.forceFailure {
		return nil, fmt.Errorf("%s: PausePod: podID: %v", mockErrorPrefix, podID)
	}

	return vc.PausePod(podID)
}

func (impl *mockImpl) ResumePod(podID string) (*vc.Pod, error) {
	if impl.resumePodFunc != nil {
		return impl.resumePodFunc(podID)
	}

	if impl.forceFailure {
		return nil, fmt.Errorf("%s: ResumePod: podID: %v", mockErrorPrefix, podID)
	}

	return vc.ResumePod(podID)
}

func (impl *mockImpl) CreateContainer(podID string, containerConfig vc.ContainerConfig) (*vc.Pod, *vc.Container, error) {
	if impl.createContainerFunc != nil {
		return impl.createContainerFunc(podID, containerConfig)
	}

	if impl.forceFailure {
		return nil, nil, fmt.Errorf("%s: CreateContainer: podID: %v, containerConfig: %v", mockErrorPrefix, podID, containerConfig)
	}

	return vc.CreateContainer(podID, containerConfig)
}

func (impl *mockImpl) DeleteContainer(podID, containerID string) (*vc.Container, error) {
	if impl.deleteContainerFunc != nil {
		return impl.deleteContainerFunc(podID, containerID)
	}

	if impl.forceFailure {
		return nil, fmt.Errorf("%s: DeleteContainer: podID: %v, containerID: %v", mockErrorPrefix, podID, containerID)
	}

	return vc.DeleteContainer(podID, containerID)
}

func (impl *mockImpl) StartContainer(podID, containerID string) (*vc.Container, error) {
	if impl.startContainerFunc != nil {
		return impl.startContainerFunc(podID, containerID)
	}

	if impl.forceFailure {
		return nil, fmt.Errorf("%s: StartContainer: podID: %v, containerID: %v", mockErrorPrefix, podID, containerID)
	}

	return vc.StartContainer(podID, containerID)
}

func (impl *mockImpl) StopContainer(podID, containerID string) (*vc.Container, error) {
	if impl.stopContainerFunc != nil {
		return impl.stopContainerFunc(podID, containerID)
	}

	if impl.forceFailure {
		return nil, fmt.Errorf("%s: StopContainer: podID: %v, containerID: %v", mockErrorPrefix, podID, containerID)
	}

	return vc.StopContainer(podID, containerID)
}

func (impl *mockImpl) EnterContainer(podID, containerID string, cmd vc.Cmd) (*vc.Pod, *vc.Container, *vc.Process, error) {
	if impl.enterContainerFunc != nil {
		return impl.enterContainerFunc(podID, containerID, cmd)
	}

	if impl.forceFailure {
		return nil, nil, nil, fmt.Errorf("%s: EnterContainer: podID: %v, containerID: %v, cmd: %v", mockErrorPrefix, podID, containerID, cmd)
	}

	return vc.EnterContainer(podID, containerID, cmd)
}

func (impl *mockImpl) StatusContainer(podID, containerID string) (vc.ContainerStatus, error) {
	if impl.statusContainerFunc != nil {
		return impl.statusContainerFunc(podID, containerID)
	}

	if impl.forceFailure {
		return vc.ContainerStatus{}, fmt.Errorf("%s: StatusContainer: podID: %v, containerID: %v", mockErrorPrefix, podID, containerID)
	}

	return vc.StatusContainer(podID, containerID)
}

func (impl *mockImpl) KillContainer(podID, containerID string, signal syscall.Signal, all bool) error {
	if impl.killContainerFunc != nil {
		return impl.killContainerFunc(podID, containerID, signal, all)
	}

	if impl.forceFailure {
		return fmt.Errorf("%s: StartContainer: podID: %v, containerID: %v, signal: %v", mockErrorPrefix, podID, containerID, signal)
	}

	return vc.KillContainer(podID, containerID, signal, all)
}

// helper functions
func isMockError(err error) bool {
	return strings.HasPrefix(err.Error(), mockErrorPrefix)
}
func listPodNoPods() ([]vc.PodStatus, error) {
	return []vc.PodStatus{}, nil
}
