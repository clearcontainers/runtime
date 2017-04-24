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
	"strings"
	"testing"
)

const (
	testHyperstartPausePath    = "/tmp/bundles/pause_bundle/rootfs/bin"
	testHyperstartPauseBinName = "pause"
	containerID                = "1"
)

func newBasicTestCmd() Cmd {
	envs := []EnvVar{
		{
			Var:   "PATH",
			Value: "/bin:/usr/bin:/sbin:/usr/sbin",
		},
	}

	cmd := Cmd{
		Args:    strings.Split("/bin/sh", " "),
		Envs:    envs,
		WorkDir: "/",
	}

	return cmd
}

func newTestPodConfigNoop() PodConfig {
	// Define the container command and bundle.
	container := ContainerConfig{
		ID:     containerID,
		RootFs: filepath.Join(testDir, testBundle),
		Cmd:    newBasicTestCmd(),
	}

	// Sets the hypervisor configuration.
	hypervisorConfig := HypervisorConfig{
		KernelPath:     filepath.Join(testDir, testKernel),
		ImagePath:      filepath.Join(testDir, testImage),
		HypervisorPath: filepath.Join(testDir, testHypervisor),
	}

	podConfig := PodConfig{
		ID:               testPodID,
		HypervisorType:   MockHypervisor,
		HypervisorConfig: hypervisorConfig,

		AgentType: NoopAgentType,

		Containers: []ContainerConfig{container},
	}

	return podConfig
}

func newTestPodConfigHyperstartAgent() PodConfig {
	// Define the container command and bundle.
	container := ContainerConfig{
		ID:     containerID,
		RootFs: filepath.Join(testDir, testBundle),
		Cmd:    newBasicTestCmd(),
	}

	// Sets the hypervisor configuration.
	hypervisorConfig := HypervisorConfig{
		KernelPath:     filepath.Join(testDir, testKernel),
		ImagePath:      filepath.Join(testDir, testImage),
		HypervisorPath: filepath.Join(testDir, testHypervisor),
	}

	sockets := []Socket{{}, {}}

	agentConfig := HyperConfig{
		SockCtlName: testHyperstartCtlSocket,
		SockTtyName: testHyperstartTtySocket,
		Sockets:     sockets,
	}

	podConfig := PodConfig{
		ID:               testPodID,
		HypervisorType:   MockHypervisor,
		HypervisorConfig: hypervisorConfig,

		AgentType:   HyperstartAgent,
		AgentConfig: agentConfig,

		Containers: []ContainerConfig{container},
	}

	return podConfig
}

func newTestPodConfigHyperstartAgentCNINetwork() PodConfig {
	// Define the container command and bundle.
	container := ContainerConfig{
		ID:     containerID,
		RootFs: filepath.Join(testDir, testBundle),
		Cmd:    newBasicTestCmd(),
	}

	// Sets the hypervisor configuration.
	hypervisorConfig := HypervisorConfig{
		KernelPath:     filepath.Join(testDir, testKernel),
		ImagePath:      filepath.Join(testDir, testImage),
		HypervisorPath: filepath.Join(testDir, testHypervisor),
	}

	sockets := []Socket{{}, {}}

	agentConfig := HyperConfig{
		SockCtlName: testHyperstartCtlSocket,
		SockTtyName: testHyperstartTtySocket,
		Sockets:     sockets,
	}

	netConfig := NetworkConfig{
		NumInterfaces: 1,
	}

	podConfig := PodConfig{
		ID:               testPodID,
		HypervisorType:   MockHypervisor,
		HypervisorConfig: hypervisorConfig,

		AgentType:   HyperstartAgent,
		AgentConfig: agentConfig,

		NetworkModel:  CNINetworkModel,
		NetworkConfig: netConfig,

		Containers: []ContainerConfig{container},
	}

	return podConfig
}

func newTestPodConfigHyperstartAgentCNMNetwork() PodConfig {
	// Define the container command and bundle.
	container := ContainerConfig{
		ID:     containerID,
		RootFs: filepath.Join(testDir, testBundle),
		Cmd:    newBasicTestCmd(),
	}

	// Sets the hypervisor configuration.
	hypervisorConfig := HypervisorConfig{
		KernelPath:     filepath.Join(testDir, testKernel),
		ImagePath:      filepath.Join(testDir, testImage),
		HypervisorPath: filepath.Join(testDir, testHypervisor),
	}

	sockets := []Socket{{}, {}}

	agentConfig := HyperConfig{
		SockCtlName: testHyperstartCtlSocket,
		SockTtyName: testHyperstartTtySocket,
		Sockets:     sockets,
	}

	hooks := Hooks{
		PreStartHooks: []Hook{
			{
				Path: testBinHookPath,
				Args: []string{testKeyHook, testContainerIDHook, testControllerIDHook},
			},
		},
		PostStartHooks: []Hook{},
		PostStopHooks:  []Hook{},
	}

	netConfig := NetworkConfig{
		NumInterfaces: len(hooks.PreStartHooks),
	}

	podConfig := PodConfig{
		ID:    testPodID,
		Hooks: hooks,

		HypervisorType:   MockHypervisor,
		HypervisorConfig: hypervisorConfig,

		AgentType:   HyperstartAgent,
		AgentConfig: agentConfig,

		NetworkModel:  CNMNetworkModel,
		NetworkConfig: netConfig,

		Containers: []ContainerConfig{container},
	}

	return podConfig
}

func TestCreatePodNoopAgentSuccessful(t *testing.T) {
	cleanUp()

	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreatePodHyperstartAgentSuccessful(t *testing.T) {
	cleanUp()

	config := newTestPodConfigHyperstartAgent()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreatePodFailing(t *testing.T) {
	cleanUp()

	config := PodConfig{}

	p, err := CreatePod(config)
	if p != nil || err == nil {
		t.Fatal()
	}
}

func TestDeletePodNoopAgentSuccessful(t *testing.T) {
	cleanUp()

	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	p, err = DeletePod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(podDir)
	if err == nil {
		t.Fatal()
	}
}

func TestDeletePodHyperstartAgentSuccessful(t *testing.T) {
	cleanUp()

	config := newTestPodConfigHyperstartAgent()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	p, err = DeletePod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(podDir)
	if err == nil {
		t.Fatal(err)
	}
}

func TestDeletePodFailing(t *testing.T) {
	cleanUp()

	podDir := filepath.Join(configStoragePath, testPodID)
	os.Remove(podDir)

	p, err := DeletePod(testPodID)
	if p != nil || err == nil {
		t.Fatal()
	}
}

func TestStartPodNoopAgentSuccessful(t *testing.T) {
	cleanUp()

	config := newTestPodConfigNoop()

	p, _, err := createAndStartPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}
}

func TestStartPodHyperstartAgentSuccessful(t *testing.T) {
	cleanUp()

	if os.Geteuid() != 0 {
		t.Skip(testDisabledAsNonRoot)
	}

	config := newTestPodConfigHyperstartAgent()

	pauseBinPath := filepath.Join(testDir, testHyperstartPauseBinName)
	_, err := os.Create(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}

	hyperConfig := config.AgentConfig.(HyperConfig)
	hyperConfig.PauseBinPath = pauseBinPath
	config.AgentConfig = hyperConfig

	p, _, err := createAndStartPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	p.agent.(*hyper).bindUnmountAllRootfs(*p)

	err = os.Remove(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartPodFailing(t *testing.T) {
	cleanUp()

	podDir := filepath.Join(configStoragePath, testPodID)
	os.Remove(podDir)

	p, err := StartPod(testPodID)
	if p != nil || err == nil {
		t.Fatal()
	}
}

func TestStopPodNoopAgentSuccessful(t *testing.T) {
	cleanUp()

	config := newTestPodConfigNoop()

	p, _, err := createAndStartPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	p, err = StopPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}
}

func TestStopPodHyperstartAgentSuccessful(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip(testDisabledAsNonRoot)
	}

	cleanUp()

	config := newTestPodConfigHyperstartAgent()

	pauseBinPath := filepath.Join(testDir, testHyperstartPauseBinName)
	_, err := os.Create(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}

	hyperConfig := config.AgentConfig.(HyperConfig)
	hyperConfig.PauseBinPath = pauseBinPath
	config.AgentConfig = hyperConfig

	p, _, err := createAndStartPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	p, err = StopPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	err = os.Remove(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStopPodFailing(t *testing.T) {
	cleanUp()

	podDir := filepath.Join(configStoragePath, testPodID)
	os.Remove(podDir)

	p, err := StopPod(testPodID)
	if p != nil || err == nil {
		t.Fatal()
	}
}

func TestRunPodNoopAgentSuccessful(t *testing.T) {
	cleanUp()

	config := newTestPodConfigNoop()

	p, err := RunPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunPodHyperstartAgentSuccessful(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip(testDisabledAsNonRoot)
	}

	cleanUp()

	config := newTestPodConfigHyperstartAgent()

	pauseBinPath := filepath.Join(testDir, testHyperstartPauseBinName)
	_, err := os.Create(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}

	hyperConfig := config.AgentConfig.(HyperConfig)
	hyperConfig.PauseBinPath = pauseBinPath
	config.AgentConfig = hyperConfig

	p, err := RunPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	p.agent.(*hyper).bindUnmountAllRootfs(*p)

	err = os.Remove(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunPodFailing(t *testing.T) {
	cleanUp()

	config := PodConfig{}

	p, err := RunPod(config)
	if p != nil || err == nil {
		t.Fatal()
	}
}

func TestListPodSuccessful(t *testing.T) {
	cleanUp()

	os.RemoveAll(configStoragePath)

	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	_, err = ListPod()
	if err != nil {
		t.Fatal(err)
	}
}

func TestListPodNoPodDirectory(t *testing.T) {
	cleanUp()

	os.RemoveAll(configStoragePath)

	_, err := ListPod()
	if err != nil {
		t.Fatal(fmt.Sprintf("unexpected ListPod error from non-existent pod directory: %v", err))
	}
}

func TestStatusPodSuccessful(t *testing.T) {
	cleanUp()

	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	_, err = StatusPod(p.id)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStatusPodFailingFetchPodConfig(t *testing.T) {
	cleanUp()

	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(configStoragePath, p.id)
	os.RemoveAll(path)

	_, err = StatusPod(p.id)
	if err == nil {
		t.Fatal()
	}
}

func TestStatusPodPodFailingFetchPodState(t *testing.T) {
	cleanUp()

	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	os.RemoveAll(p.configPath)

	_, err = StatusPod(p.id)
	if err == nil {
		t.Fatal()
	}
}

func newTestContainerConfigNoop(contID string) ContainerConfig {
	// Define the container command and bundle.
	container := ContainerConfig{
		ID:     contID,
		RootFs: filepath.Join(testDir, testBundle),
		Cmd:    newBasicTestCmd(),
	}

	return container
}

func TestCreateContainerSuccessful(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	_, c, err := CreateContainer(p.id, contConfig)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	contDir := filepath.Join(podDir, contID)
	_, err = os.Stat(contDir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateContainerFailingNoPod(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	p, err = DeletePod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err == nil {
		t.Fatal()
	}

	contConfig := newTestContainerConfigNoop(contID)

	_, c, err := CreateContainer(p.id, contConfig)
	if c != nil || err == nil {
		t.Fatal(err)
	}
}

func TestDeleteContainerSuccessful(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	_, c, err := CreateContainer(p.id, contConfig)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	contDir := filepath.Join(podDir, contID)
	_, err = os.Stat(contDir)
	if err != nil {
		t.Fatal(err)
	}

	c, err = DeleteContainer(p.id, contID)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(contDir)
	if err == nil {
		t.Fatal()
	}
}

func TestDeleteContainerFailingNoPod(t *testing.T) {
	cleanUp()

	podDir := filepath.Join(configStoragePath, testPodID)
	contID := "100"
	os.RemoveAll(podDir)

	c, err := DeleteContainer(testPodID, contID)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestDeleteContainerFailingNoContainer(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	c, err := DeleteContainer(p.id, contID)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestStartContainerNoopAgentSuccessful(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, podDir, err := createAndStartPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}
	contConfig := newTestContainerConfigNoop(contID)

	_, c, err := CreateContainer(p.id, contConfig)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	contDir := filepath.Join(podDir, contID)
	_, err = os.Stat(contDir)
	if err != nil {
		t.Fatal(err)
	}

	c, err = StartContainer(p.id, contID)
	if c == nil || err != nil {
		t.Fatal(err)
	}
}

func TestStartContainerFailingNoPod(t *testing.T) {
	cleanUp()

	podDir := filepath.Join(configStoragePath, testPodID)
	contID := "100"
	os.RemoveAll(podDir)

	c, err := StartContainer(testPodID, contID)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestStartContainerFailingNoContainer(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	c, err := StartContainer(p.id, contID)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestStartContainerFailingPodNotStarted(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	_, c, err := CreateContainer(p.id, contConfig)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	contDir := filepath.Join(podDir, contID)
	_, err = os.Stat(contDir)
	if err != nil {
		t.Fatal(err)
	}

	c, err = StartContainer(p.id, contID)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestStopContainerNoopAgentSuccessful(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, podDir, err := createAndStartPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	_, c, err := CreateContainer(p.id, contConfig)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	contDir := filepath.Join(podDir, contID)
	_, err = os.Stat(contDir)
	if err != nil {
		t.Fatal(err)
	}

	c, err = StartContainer(p.id, contID)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	c, err = StopContainer(p.id, contID)
	if c == nil || err != nil {
		t.Fatal(err)
	}
}

func TestStartStopContainerHyperstartAgentSuccessful(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip(testDisabledAsNonRoot)
	}

	cleanUp()

	contID := "100"
	config := newTestPodConfigHyperstartAgent()

	pauseBinPath := filepath.Join(testDir, testHyperstartPauseBinName)
	_, err := os.Create(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}

	hyperConfig := config.AgentConfig.(HyperConfig)
	hyperConfig.PauseBinPath = pauseBinPath
	config.AgentConfig = hyperConfig

	p, podDir, err := createAndStartPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	_, c, err := CreateContainer(p.id, contConfig)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	contDir := filepath.Join(podDir, contID)
	_, err = os.Stat(contDir)
	if err != nil {
		t.Fatal(err)
	}

	c, err = StartContainer(p.id, contID)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	c, err = StopContainer(p.id, contID)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	p.agent.(*hyper).bindUnmountAllRootfs(*p)

	err = os.Remove(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartStopPodHyperstartAgentSuccessfulWithCNINetwork(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip(testDisabledAsNonRoot)
	}

	cleanUp()

	config := newTestPodConfigHyperstartAgentCNINetwork()

	pauseBinPath := filepath.Join(testDir, testHyperstartPauseBinName)
	_, err := os.Create(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}

	hyperConfig := config.AgentConfig.(HyperConfig)
	hyperConfig.PauseBinPath = pauseBinPath
	config.AgentConfig = hyperConfig

	p, _, err := createAndStartPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	p, err = StopPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	p, err = DeletePod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	err = os.Remove(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartStopPodHyperstartAgentSuccessfulWithCNMNetwork(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip(testDisabledAsNonRoot)
	}

	config := newTestPodConfigHyperstartAgentCNMNetwork()

	pauseBinPath := filepath.Join(testDir, testHyperstartPauseBinName)
	_, err := os.Create(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}

	hyperConfig := config.AgentConfig.(HyperConfig)
	hyperConfig.PauseBinPath = pauseBinPath
	config.AgentConfig = hyperConfig

	p, _, err := createAndStartPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	p, err = StopPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	p, err = DeletePod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	err = os.Remove(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStopContainerFailingNoPod(t *testing.T) {
	cleanUp()

	podDir := filepath.Join(configStoragePath, testPodID)
	contID := "100"
	os.RemoveAll(podDir)

	c, err := StopContainer(testPodID, contID)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestStopContainerFailingNoContainer(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	c, err := StopContainer(p.id, contID)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestStopContainerFailingContNotStarted(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, podDir, err := createAndStartPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	_, c, err := CreateContainer(p.id, contConfig)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	contDir := filepath.Join(podDir, contID)
	_, err = os.Stat(contDir)
	if err != nil {
		t.Fatal(err)
	}

	c, err = StopContainer(p.id, contID)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestEnterContainerNoopAgentSuccessful(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, podDir, err := createAndStartPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	_, c, err := CreateContainer(p.id, contConfig)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	contDir := filepath.Join(podDir, contID)
	_, err = os.Stat(contDir)
	if err != nil {
		t.Fatal(err)
	}

	c, err = StartContainer(p.id, contID)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	cmd := newBasicTestCmd()

	_, c, _, err = EnterContainer(p.id, contID, cmd)
	if c == nil || err != nil {
		t.Fatal(err)
	}
}

func TestEnterContainerHyperstartAgentSuccessful(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip(testDisabledAsNonRoot)
	}

	cleanUp()

	contID := "100"
	config := newTestPodConfigHyperstartAgent()

	pauseBinPath := filepath.Join(testDir, testHyperstartPauseBinName)
	_, err := os.Create(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}

	hyperConfig := config.AgentConfig.(HyperConfig)
	hyperConfig.PauseBinPath = pauseBinPath
	config.AgentConfig = hyperConfig

	p, podDir, err := createAndStartPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	_, _, err = CreateContainer(p.id, contConfig)
	if err != nil {
		t.Fatal(err)
	}

	contDir := filepath.Join(podDir, contID)
	_, err = os.Stat(contDir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = StartContainer(p.id, contID)
	if err != nil {
		t.Fatal(err)
	}

	cmd := newBasicTestCmd()

	_, _, _, err = EnterContainer(p.id, contID, cmd)
	if err != nil {
		t.Fatal(err)
	}

	_, err = StopContainer(p.id, contID)
	if err != nil {
		t.Fatal(err)
	}

	p.agent.(*hyper).bindUnmountAllRootfs(*p)

	err = os.Remove(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEnterContainerFailingNoPod(t *testing.T) {
	cleanUp()

	podDir := filepath.Join(configStoragePath, testPodID)
	contID := "100"
	os.RemoveAll(podDir)

	cmd := newBasicTestCmd()

	_, c, _, err := EnterContainer(testPodID, contID, cmd)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestEnterContainerFailingNoContainer(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	cmd := newBasicTestCmd()

	_, c, _, err := EnterContainer(p.id, contID, cmd)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestEnterContainerFailingContNotStarted(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, podDir, err := createAndStartPod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	_, c, err := CreateContainer(p.id, contConfig)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	contDir := filepath.Join(podDir, contID)
	_, err = os.Stat(contDir)
	if err != nil {
		t.Fatal(err)
	}

	cmd := newBasicTestCmd()

	_, c, _, err = EnterContainer(p.id, contID, cmd)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestStatusContainerSuccessful(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	_, c, err := CreateContainer(p.id, contConfig)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	contDir := filepath.Join(podDir, contID)
	_, err = os.Stat(contDir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = StatusContainer(p.id, contID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStatusContainerFailing(t *testing.T) {
	cleanUp()

	contID := "100"
	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	os.RemoveAll(p.configPath)

	_, err = StatusContainer(p.id, contID)
	if err == nil {
		t.Fatal()
	}
}

/*
 * Benchmarks
 */

func createNewPodConfig(hType HypervisorType, aType AgentType, aConfig interface{}, netModel NetworkModel) PodConfig {
	hypervisorConfig := HypervisorConfig{
		KernelPath:     "/usr/share/clear-containers/vmlinux.container",
		ImagePath:      "/usr/share/clear-containers/clear-containers.img",
		HypervisorPath: "/usr/bin/qemu-lite-system-x86_64",
	}

	netConfig := NetworkConfig{
		NumInterfaces: 1,
	}

	return PodConfig{
		ID:               testPodID,
		HypervisorType:   hType,
		HypervisorConfig: hypervisorConfig,

		AgentType:   aType,
		AgentConfig: aConfig,

		NetworkModel:  netModel,
		NetworkConfig: netConfig,
	}
}

func createNewContainerConfigs(numOfContainers int) []ContainerConfig {
	var contConfigs []ContainerConfig

	envs := []EnvVar{
		{
			Var:   "PATH",
			Value: "/bin:/usr/bin:/sbin:/usr/sbin",
		},
	}

	cmd := Cmd{
		Args:    strings.Split("/bin/ps -A", " "),
		Envs:    envs,
		WorkDir: "/",
	}

	rootFs := "/tmp/bundles/busybox/"

	for i := 0; i < numOfContainers; i++ {
		contConfig := ContainerConfig{
			ID:     fmt.Sprintf("%d", i),
			RootFs: rootFs,
			Cmd:    cmd,
		}

		contConfigs = append(contConfigs, contConfig)
	}

	return contConfigs
}

// createAndStartPod handles the common test operation of creating and
// starting a pod.
func createAndStartPod(config PodConfig) (pod *Pod, podDir string,
	err error) {

	// Create pod
	pod, err = CreatePod(config)
	if pod == nil || err != nil {
		return nil, "", err
	}

	podDir = filepath.Join(configStoragePath, pod.id)
	_, err = os.Stat(podDir)
	if err != nil {
		return nil, "", err
	}

	// Start pod
	pod, err = StartPod(pod.id)
	if pod == nil || err != nil {
		return nil, "", err
	}

	return pod, podDir, nil
}

func createStartStopDeletePod(b *testing.B, podConfig PodConfig) {
	p, _, err := createAndStartPod(podConfig)
	if p == nil || err != nil {
		b.Fatalf("Could not create and start pod: %s", err)
	}

	// Stop pod
	_, err = StopPod(p.id)
	if err != nil {
		b.Fatalf("Could not stop pod: %s", err)
	}

	// Delete pod
	_, err = DeletePod(p.id)
	if err != nil {
		b.Fatalf("Could not delete pod: %s", err)
	}
}

func createStartStopDeleteContainers(b *testing.B, podConfig PodConfig, contConfigs []ContainerConfig) {
	// Create pod
	p, err := CreatePod(podConfig)
	if err != nil {
		b.Fatalf("Could not create pod: %s", err)
	}

	// Start pod
	_, err = StartPod(p.id)
	if err != nil {
		b.Fatalf("Could not start pod: %s", err)
	}

	// Create containers
	for _, contConfig := range contConfigs {
		_, _, err := CreateContainer(p.id, contConfig)
		if err != nil {
			b.Fatalf("Could not create container %s: %s", contConfig.ID, err)
		}
	}

	// Start containers
	for _, contConfig := range contConfigs {
		_, err := StartContainer(p.id, contConfig.ID)
		if err != nil {
			b.Fatalf("Could not start container %s: %s", contConfig.ID, err)
		}
	}

	// Stop containers
	for _, contConfig := range contConfigs {
		_, err := StopContainer(p.id, contConfig.ID)
		if err != nil {
			b.Fatalf("Could not stop container %s: %s", contConfig.ID, err)
		}
	}

	// Delete containers
	for _, contConfig := range contConfigs {
		_, err := DeleteContainer(p.id, contConfig.ID)
		if err != nil {
			b.Fatalf("Could not delete container %s: %s", contConfig.ID, err)
		}
	}

	// Stop pod
	_, err = StopPod(p.id)
	if err != nil {
		b.Fatalf("Could not stop pod: %s", err)
	}

	// Delete pod
	_, err = DeletePod(p.id)
	if err != nil {
		b.Fatalf("Could not delete pod: %s", err)
	}
}

var benchmarkHyperConfig = HyperConfig{
	PauseBinPath: filepath.Join(testHyperstartPausePath, testHyperstartPauseBinName),
}

func BenchmarkCreateStartStopDeletePodQemuHypervisorHyperstartAgentNetworkCNI(b *testing.B) {
	for i := 0; i < b.N; i++ {
		podConfig := createNewPodConfig(QemuHypervisor, HyperstartAgent, benchmarkHyperConfig, CNINetworkModel)
		createStartStopDeletePod(b, podConfig)
	}
}

func BenchmarkCreateStartStopDeletePodQemuHypervisorNoopAgentNetworkCNI(b *testing.B) {
	for i := 0; i < b.N; i++ {
		podConfig := createNewPodConfig(QemuHypervisor, NoopAgentType, nil, CNINetworkModel)
		createStartStopDeletePod(b, podConfig)
	}
}

func BenchmarkCreateStartStopDeletePodQemuHypervisorHyperstartAgentNetworkNoop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		podConfig := createNewPodConfig(QemuHypervisor, HyperstartAgent, benchmarkHyperConfig, NoopNetworkModel)
		createStartStopDeletePod(b, podConfig)
	}
}

func BenchmarkCreateStartStopDeletePodQemuHypervisorNoopAgentNetworkNoop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		podConfig := createNewPodConfig(QemuHypervisor, NoopAgentType, nil, NoopNetworkModel)
		createStartStopDeletePod(b, podConfig)
	}
}

func BenchmarkCreateStartStopDeletePodMockHypervisorNoopAgentNetworkNoop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		podConfig := createNewPodConfig(MockHypervisor, NoopAgentType, nil, NoopNetworkModel)
		createStartStopDeletePod(b, podConfig)
	}
}

func BenchmarkStartStop1ContainerQemuHypervisorHyperstartAgentNetworkNoop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		podConfig := createNewPodConfig(QemuHypervisor, HyperstartAgent, benchmarkHyperConfig, NoopNetworkModel)
		contConfigs := createNewContainerConfigs(1)
		createStartStopDeleteContainers(b, podConfig, contConfigs)
	}
}

func BenchmarkStartStop10ContainerQemuHypervisorHyperstartAgentNetworkNoop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		podConfig := createNewPodConfig(QemuHypervisor, HyperstartAgent, benchmarkHyperConfig, NoopNetworkModel)
		contConfigs := createNewContainerConfigs(10)
		createStartStopDeleteContainers(b, podConfig, contConfigs)
	}
}
