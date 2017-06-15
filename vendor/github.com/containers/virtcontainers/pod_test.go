//
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
//

package virtcontainers

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func newHypervisorConfig(kernelParams []Param, hParams []Param) HypervisorConfig {
	return HypervisorConfig{
		KernelPath:       filepath.Join(testDir, testKernel),
		ImagePath:        filepath.Join(testDir, testImage),
		HypervisorPath:   filepath.Join(testDir, testHypervisor),
		KernelParams:     kernelParams,
		HypervisorParams: hParams,
	}

}

func testCreatePod(t *testing.T, id string,
	htype HypervisorType, hconfig HypervisorConfig, atype AgentType,
	nmodel NetworkModel, nconfig NetworkConfig, containers []ContainerConfig,
	volumes []Volume) (*Pod, error) {

	config := PodConfig{
		ID:               id,
		HypervisorType:   htype,
		HypervisorConfig: hconfig,
		AgentType:        atype,
		NetworkModel:     nmodel,
		NetworkConfig:    nconfig,
		Volumes:          volumes,
		Containers:       containers,
	}

	pod, err := createPod(config)
	if err != nil {
		return nil, fmt.Errorf("Could not create pod: %s", err)
	}

	if pod.id == "" {
		return pod, fmt.Errorf("Invalid empty pod ID")
	}

	if id != "" && pod.id != id {
		return pod, fmt.Errorf("Invalid ID %s vs %s", id, pod.id)
	}

	return pod, nil
}

func TestCreateEmtpyPod(t *testing.T) {
	_, err := testCreatePod(t, testPodID, MockHypervisor, HypervisorConfig{}, NoopAgentType, NoopNetworkModel, NetworkConfig{}, nil, nil)
	if err == nil {
		t.Fatalf("VirtContainers should not allow empty pods")
	}
}

func TestCreateEmtpyHypervisorPod(t *testing.T) {
	_, err := testCreatePod(t, testPodID, QemuHypervisor, HypervisorConfig{}, NoopAgentType, NoopNetworkModel, NetworkConfig{}, nil, nil)
	if err == nil {
		t.Fatalf("VirtContainers should not allow pods with empty hypervisors")
	}
}

func TestCreateMockPod(t *testing.T) {
	hConfig := newHypervisorConfig(nil, nil)

	_, err := testCreatePod(t, testPodID, MockHypervisor, hConfig, NoopAgentType, NoopNetworkModel, NetworkConfig{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreatePodEmtpyID(t *testing.T) {
	hConfig := newHypervisorConfig(nil, nil)

	p, err := testCreatePod(t, "", MockHypervisor, hConfig, NoopAgentType, NoopNetworkModel, NetworkConfig{}, nil, nil)
	if err == nil {
		t.Fatalf("Expected pod with empty ID to fail, but got pod %v", p)
	}
}

func testPodStateTransition(t *testing.T, state stateString, newState stateString) error {
	hConfig := newHypervisorConfig(nil, nil)

	p, err := testCreatePod(t, testPodID, MockHypervisor, hConfig, NoopAgentType, NoopNetworkModel, NetworkConfig{}, nil, nil)
	if err != nil {
		return err
	}

	p.state = State{
		State: state,
	}

	return p.state.validTransition(state, newState)
}

func TestPodStateReadyRunning(t *testing.T) {
	err := testPodStateTransition(t, StateReady, StateRunning)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPodStateRunningPaused(t *testing.T) {
	err := testPodStateTransition(t, StateRunning, StatePaused)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPodStatePausedRunning(t *testing.T) {
	err := testPodStateTransition(t, StatePaused, StateRunning)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPodStatePausedStopped(t *testing.T) {
	err := testPodStateTransition(t, StatePaused, StateStopped)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPodStateRunningStopped(t *testing.T) {
	err := testPodStateTransition(t, StateRunning, StateStopped)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPodStateReadyPaused(t *testing.T) {
	err := testPodStateTransition(t, StateReady, StateStopped)
	if err == nil {
		t.Fatal("Invalid transition from Ready to Paused")
	}
}

func TestPodStatePausedReady(t *testing.T) {
	err := testPodStateTransition(t, StateStopped, StateReady)
	if err == nil {
		t.Fatal("Invalid transition from Ready to Paused")
	}
}

func testPodDir(t *testing.T, resource podResource, expected string) error {
	fs := filesystem{}
	_, dir, err := fs.podURI(testPodID, resource)
	if err != nil {
		return err
	}

	if dir != expected {
		return fmt.Errorf("Unexpected pod directory %s vs %s", dir, expected)
	}

	return nil
}

func testPodFile(t *testing.T, resource podResource, expected string) error {
	fs := filesystem{}
	file, _, err := fs.podURI(testPodID, resource)
	if err != nil {
		return err
	}

	if file != expected {
		return fmt.Errorf("Unexpected pod file %s vs %s", file, expected)
	}

	return nil
}

func TestPodDirConfig(t *testing.T) {
	err := testPodDir(t, configFileType, podDirConfig)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPodDirState(t *testing.T) {
	err := testPodDir(t, stateFileType, podDirState)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPodDirLock(t *testing.T) {
	err := testPodDir(t, lockFileType, podDirLock)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPodDirNegative(t *testing.T) {
	fs := filesystem{}
	_, _, err := fs.podURI("", lockFileType)
	if err == nil {
		t.Fatal("Empty pod IDs should not be allowed")
	}
}

func TestPodFileConfig(t *testing.T) {
	err := testPodFile(t, configFileType, podFileConfig)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPodFileState(t *testing.T) {
	err := testPodFile(t, stateFileType, podFileState)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPodFileLock(t *testing.T) {
	err := testPodFile(t, lockFileType, podFileLock)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPodFileNegative(t *testing.T) {
	fs := filesystem{}
	_, _, err := fs.podURI("", lockFileType)
	if err == nil {
		t.Fatal("Empty pod IDs should not be allowed")
	}
}

func testStateValid(t *testing.T, stateStr stateString, expected bool) {
	state := &State{
		State: stateStr,
	}

	ok := state.valid()
	if ok != expected {
		t.Fatal()
	}
}

func TestStateValidSuccessful(t *testing.T) {
	testStateValid(t, StateReady, true)
	testStateValid(t, StateRunning, true)
	testStateValid(t, StatePaused, true)
	testStateValid(t, StateStopped, true)
}

func TestStateValidFailing(t *testing.T) {
	testStateValid(t, "", false)
}

func TestValidTransitionFailingOldStateMismatch(t *testing.T) {
	state := &State{
		State: StateReady,
	}

	err := state.validTransition(StateRunning, StateStopped)
	if err == nil {
		t.Fatal()
	}
}

func TestVolumesSetSuccessful(t *testing.T) {
	volumes := &Volumes{}

	volStr := "mountTag1:hostPath1 mountTag2:hostPath2"

	expected := Volumes{
		{
			MountTag: "mountTag1",
			HostPath: "hostPath1",
		},
		{
			MountTag: "mountTag2",
			HostPath: "hostPath2",
		},
	}

	err := volumes.Set(volStr)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(*volumes, expected) == false {
		t.Fatal()
	}
}

func TestVolumesSetFailingTooFewArguments(t *testing.T) {
	volumes := &Volumes{}

	volStr := "mountTag1 mountTag2"

	err := volumes.Set(volStr)
	if err == nil {
		t.Fatal()
	}
}

func TestVolumesSetFailingTooManyArguments(t *testing.T) {
	volumes := &Volumes{}

	volStr := "mountTag1:hostPath1:Foo1 mountTag2:hostPath2:Foo2"

	err := volumes.Set(volStr)
	if err == nil {
		t.Fatal()
	}
}

func TestVolumesSetFailingVoidArguments(t *testing.T) {
	volumes := &Volumes{}

	volStr := ": : :"

	err := volumes.Set(volStr)
	if err == nil {
		t.Fatal()
	}
}

func TestVolumesStringSuccessful(t *testing.T) {
	volumes := &Volumes{
		{
			MountTag: "mountTag1",
			HostPath: "hostPath1",
		},
		{
			MountTag: "mountTag2",
			HostPath: "hostPath2",
		},
	}

	expected := "mountTag1:hostPath1 mountTag2:hostPath2"

	result := volumes.String()
	if result != expected {
		t.Fatal()
	}
}

func TestSocketsSetSuccessful(t *testing.T) {
	sockets := &Sockets{}

	sockStr := "devID1:id1:hostPath1:Name1 devID2:id2:hostPath2:Name2"

	expected := Sockets{
		{
			DeviceID: "devID1",
			ID:       "id1",
			HostPath: "hostPath1",
			Name:     "Name1",
		},
		{
			DeviceID: "devID2",
			ID:       "id2",
			HostPath: "hostPath2",
			Name:     "Name2",
		},
	}

	err := sockets.Set(sockStr)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(*sockets, expected) == false {
		t.Fatal()
	}
}

func TestSocketsSetFailingWrongArgsAmount(t *testing.T) {
	sockets := &Sockets{}

	sockStr := "devID1:id1:hostPath1"

	err := sockets.Set(sockStr)
	if err == nil {
		t.Fatal()
	}
}

func TestSocketsSetFailingVoidArguments(t *testing.T) {
	sockets := &Sockets{}

	sockStr := ":::"

	err := sockets.Set(sockStr)
	if err == nil {
		t.Fatal()
	}
}

func TestSocketsStringSuccessful(t *testing.T) {
	sockets := &Sockets{
		{
			DeviceID: "devID1",
			ID:       "id1",
			HostPath: "hostPath1",
			Name:     "Name1",
		},
		{
			DeviceID: "devID2",
			ID:       "id2",
			HostPath: "hostPath2",
			Name:     "Name2",
		},
	}

	expected := "devID1:id1:hostPath1:Name1 devID2:id2:hostPath2:Name2"

	result := sockets.String()
	if result != expected {
		t.Fatal()
	}
}

func TestPodListSuccessful(t *testing.T) {
	pod := &Pod{}

	podList, err := pod.list()
	if podList != nil || err != nil {
		t.Fatal()
	}
}

func TestPodEnterSuccessful(t *testing.T) {
	pod := &Pod{}

	err := pod.enter([]string{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPodSetPodAndContainerState(t *testing.T) {
	contID := "505"
	contConfig := newTestContainerConfigNoop(contID)
	hConfig := newHypervisorConfig(nil, nil)

	// create a pod
	p, err := testCreatePod(t, testPodID, MockHypervisor, hConfig, NoopAgentType, NoopNetworkModel, NetworkConfig{}, []ContainerConfig{contConfig}, nil)
	if err != nil {
		t.Fatal(err)
	}

	l := len(p.GetAllContainers())
	if l != 1 {
		t.Fatalf("Expected 1 container found %v", l)
	}

	initialPodState := State{
		State: StateReady,
		URL:   "",
	}

	// initially, a container has an empty state
	initialContainerState := State{
		State: "",
		URL:   "",
	}

	if p.state.State != initialPodState.State {
		t.Errorf("Expected pod state %v, got %v", initialPodState.State, p.state.State)
	}

	if p.state.URL != initialPodState.URL {
		t.Errorf("Expected pod state URL %v, got %v", initialPodState.URL, p.state.URL)
	}

	c, err := p.getContainer(contID)
	if err != nil {
		t.Fatalf("Failed to retrieve container %v: %v", contID, err)
	}

	if c.state.State != initialContainerState.State {
		t.Errorf("Expected container state %v, got %v", initialContainerState.State, c.state.State)
	}

	if c.state.URL != initialContainerState.URL {
		t.Errorf("Expected container state URL %v, got %v", initialContainerState.URL, c.state.URL)
	}

	// persist to disk
	err = p.storePod()
	if err != nil {
		t.Fatal(err)
	}

	newPodState := State{
		State: StateRunning,
		URL:   "http://pod/url",
	}

	newContainerState := State{
		State: StateStopped,
		URL:   "",
	}

	// force pod state change
	err = p.setPodState(newPodState)
	if err != nil {
		t.Fatalf("Unexpected error: %v (pod %+v)", err, p)
	}

	// check the in-memory state is correct
	if p.state.State != newPodState.State {
		t.Errorf("Expected state %v, got %v", newPodState.State, p.state.State)
	}

	if p.state.URL != newPodState.URL {
		t.Errorf("Expected state URL %v, got %v", newPodState.URL, p.state.URL)
	}

	// force container state change
	err = p.setContainerState(contID, newContainerState.State)
	if err != nil {
		t.Fatalf("Unexpected error: %v (pod %+v)", err, p)
	}

	// check the in-memory state is correct
	if c.state.State != newContainerState.State {
		t.Errorf("Expected state %v, got %v", newContainerState.State, c.state.State)
	}

	if c.state.URL != newContainerState.URL {
		t.Errorf("Expected state URL %v, got %v", newContainerState.URL, c.state.URL)
	}

	// force state to be read from disk
	p2, err := fetchPod(p.ID())
	if err != nil {
		t.Fatalf("Failed to fetch pod %v: %v", p.ID(), err)
	}

	// check on-disk state is correct
	if p2.state.State != newPodState.State {
		t.Errorf("Expected state %v, got %v", newPodState.State, p2.state.State)
	}

	if p2.state.URL != newPodState.URL {
		t.Errorf("Expected state URL %v, got %v", newPodState.URL, p2.state.URL)
	}

	c2, err := p2.getContainer(contID)
	if err != nil {
		t.Fatalf("Failed to find container %v: %v", contID, err)
	}

	// check on-disk state is correct
	if c2.state.State != newContainerState.State {
		t.Errorf("Expected state %v, got %v", newContainerState.State, c2.state.State)
	}

	if c2.state.URL != newContainerState.URL {
		t.Errorf("Expected state URL %v, got %v", newContainerState.URL, c2.state.URL)
	}

	// revert pod state to allow it to be deleted
	err = p.setPodState(initialPodState)
	if err != nil {
		t.Fatalf("Unexpected error: %v (pod %+v)", err, p)
	}

	// clean up
	err = p.delete()
	if err != nil {
		t.Fatal(err)
	}
}

func TestPodSetPodStateFailingStorePodResource(t *testing.T) {
	fs := &filesystem{}
	pod := &Pod{
		storage: fs,
	}

	pod.state.State = StateReady
	err := pod.setPodState(pod.state)
	if err == nil {
		t.Fatal()
	}
}

func TestPodSetContainerStateFailingStoreContainerResource(t *testing.T) {
	fs := &filesystem{}
	pod := &Pod{
		storage: fs,
	}

	err := pod.setContainerState("100", StateReady)
	if err == nil {
		t.Fatal()
	}
}

func TestPodSetContainersStateFailingEmptyPodID(t *testing.T) {
	containers := []ContainerConfig{
		{
			ID: "100",
		},
	}

	podConfig := &PodConfig{
		Containers: containers,
	}

	fs := &filesystem{}
	pod := &Pod{
		config:  podConfig,
		storage: fs,
	}

	err := pod.setContainersState(StateReady)
	if err == nil {
		t.Fatal()
	}
}

func TestPodDeleteContainerStateSuccessful(t *testing.T) {
	contID := "100"

	fs := &filesystem{}
	pod := &Pod{
		id:      testPodID,
		storage: fs,
	}

	path := filepath.Join(runStoragePath, testPodID, contID)
	err := os.MkdirAll(path, dirMode)
	if err != nil {
		t.Fatal(err)
	}

	stateFilePath := filepath.Join(path, stateFile)

	os.Remove(stateFilePath)

	_, err = os.Create(stateFilePath)
	if err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(stateFilePath)
	if err != nil {
		t.Fatal(err)
	}

	err = pod.deleteContainerState(contID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(stateFilePath)
	if err == nil {
		t.Fatal()
	}
}

func TestPodDeleteContainerStateFailingEmptyPodID(t *testing.T) {
	contID := "100"

	fs := &filesystem{}
	pod := &Pod{
		storage: fs,
	}

	err := pod.deleteContainerState(contID)
	if err == nil {
		t.Fatal()
	}
}

func TestPodDeleteContainersStateSuccessful(t *testing.T) {
	var err error

	containers := []ContainerConfig{
		{
			ID: "100",
		},
		{
			ID: "200",
		},
	}

	podConfig := &PodConfig{
		Containers: containers,
	}

	fs := &filesystem{}
	pod := &Pod{
		id:      testPodID,
		config:  podConfig,
		storage: fs,
	}

	for _, c := range containers {
		path := filepath.Join(runStoragePath, testPodID, c.ID)
		err = os.MkdirAll(path, dirMode)
		if err != nil {
			t.Fatal(err)
		}

		stateFilePath := filepath.Join(path, stateFile)

		os.Remove(stateFilePath)

		_, err = os.Create(stateFilePath)
		if err != nil {
			t.Fatal(err)
		}

		_, err = os.Stat(stateFilePath)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = pod.deleteContainersState()
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range containers {
		stateFilePath := filepath.Join(runStoragePath, testPodID, c.ID, stateFile)
		_, err = os.Stat(stateFilePath)
		if err == nil {
			t.Fatal()
		}
	}
}

func TestPodDeleteContainersStateFailingEmptyPodID(t *testing.T) {
	containers := []ContainerConfig{
		{
			ID: "100",
		},
	}

	podConfig := &PodConfig{
		Containers: containers,
	}

	fs := &filesystem{}
	pod := &Pod{
		config:  podConfig,
		storage: fs,
	}

	err := pod.deleteContainersState()
	if err == nil {
		t.Fatal()
	}
}

func TestPodCheckContainerStateFailingEmptyPodID(t *testing.T) {
	contID := "100"
	fs := &filesystem{}
	pod := &Pod{
		storage: fs,
	}

	err := pod.checkContainerState(contID, StateReady)
	if err == nil {
		t.Fatal()
	}
}

func TestPodCheckContainerStateFailingNotExpectedState(t *testing.T) {
	contID := "100"

	fs := &filesystem{}
	pod := &Pod{
		id:      testPodID,
		storage: fs,
	}

	path := filepath.Join(runStoragePath, testPodID, contID)
	err := os.MkdirAll(path, dirMode)
	if err != nil {
		t.Fatal(err)
	}

	stateFilePath := filepath.Join(path, stateFile)

	os.Remove(stateFilePath)

	f, err := os.Create(stateFilePath)
	if err != nil {
		t.Fatal(err)
	}

	stateData := "{\"state\":\"ready\"}"
	n, err := f.WriteString(stateData)
	if err != nil || n != len(stateData) {
		f.Close()
		t.Fatal()
	}
	f.Close()

	_, err = os.Stat(stateFilePath)
	if err != nil {
		t.Fatal(err)
	}

	err = pod.checkContainerState(contID, StateStopped)
	if err == nil {
		t.Fatal()
	}
}

func TestPodCheckContainersStateFailingEmptyPodID(t *testing.T) {
	containers := []ContainerConfig{
		{
			ID: "100",
		},
	}

	podConfig := &PodConfig{
		Containers: containers,
	}

	fs := &filesystem{}
	pod := &Pod{
		config:  podConfig,
		storage: fs,
	}

	err := pod.checkContainersState(StateReady)
	if err == nil {
		t.Fatal()
	}
}

func TestGetContainer(t *testing.T) {
	containerIDs := []string{"abc", "123", "xyz", "rgb"}
	containers := []*Container{}

	for _, id := range containerIDs {
		c := Container{id: id}
		containers = append(containers, &c)
	}

	pod := Pod{
		containers: containers,
	}

	c := pod.GetContainer("noid")
	if c != nil {
		t.Fatal()
	}

	for _, id := range containerIDs {
		c = pod.GetContainer(id)
		if c == nil {
			t.Fatal()
		}
	}
}

func TestGetAllContainers(t *testing.T) {
	containerIDs := []string{"abc", "123", "xyz", "rgb"}
	containers := []*Container{}

	for _, id := range containerIDs {
		c := Container{id: id}
		containers = append(containers, &c)
	}

	pod := Pod{
		containers: containers,
	}

	list := pod.GetAllContainers()

	for i, c := range list {
		if c.id != containerIDs[i] {
			t.Fatal()
		}
	}
}

func TestSetAnnotations(t *testing.T) {
	pod := Pod{
		id:              "abcxyz123",
		storage:         &filesystem{},
		annotationsLock: &sync.RWMutex{},
		config: &PodConfig{
			Annotations: map[string]string{
				"annotation1": "abc",
			},
		},
	}

	keyAnnotation := "annotation2"
	valueAnnotation := "xyz"
	newAnnotations := map[string]string{
		keyAnnotation: valueAnnotation,
	}

	// Add a new annotation
	pod.SetAnnotations(newAnnotations)

	v, err := pod.Annotations(keyAnnotation)
	if err != nil {
		t.Fatal()
	}

	if v != valueAnnotation {
		t.Fatal()
	}

	//Change the value of an annotation
	valueAnnotation = "123"
	newAnnotations[keyAnnotation] = valueAnnotation

	pod.SetAnnotations(newAnnotations)

	v, err = pod.Annotations(keyAnnotation)
	if err != nil {
		t.Fatal()
	}

	if v != valueAnnotation {
		t.Fatal()
	}
}

func TestPodGetContainer(t *testing.T) {

	emptyPod := Pod{}
	_, err := emptyPod.getContainer("")
	if err == nil {
		t.Fatal("Expected error for containerless pod")
	}

	_, err = emptyPod.getContainer("foo")
	if err == nil {
		t.Fatal("Expected error for containerless pod and invalid containerID")
	}

	hConfig := newHypervisorConfig(nil, nil)
	p, err := testCreatePod(t, testPodID, MockHypervisor, hConfig, NoopAgentType, NoopNetworkModel, NetworkConfig{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	p.state.URL = noopProxyURL

	contID := "999"
	contConfig := newTestContainerConfigNoop(contID)
	_, err = createContainer(p, contConfig)
	if err != nil {
		t.Fatalf("Failed to create container %+v in pod %+v: %v", contConfig, p, err)
	}

	got := false
	for _, c := range p.GetAllContainers() {
		c2, err := p.getContainer(c.ID())
		if err != nil {
			t.Fatalf("Failed to find container %v: %v", c.ID(), err)
		}

		if c2.ID() != c.ID() {
			t.Fatalf("Expected container %v but got %v", c.ID(), c2.ID())
		}

		if c2.ID() == contID {
			got = true
		}
	}

	if !got {
		t.Fatalf("Failed to find container %v", contID)
	}
}
