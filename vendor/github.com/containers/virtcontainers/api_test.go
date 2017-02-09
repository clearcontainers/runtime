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
	TestHyperstartCtlSocket    = "/tmp/test_hyper.sock"
	TestHyperstartTtySocket    = "/tmp/test_tty.sock"
	TestHyperstartPauseBinName = "pause"
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
		ID:     "1",
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
		ID:     "1",
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
		SockCtlName: TestHyperstartCtlSocket,
		SockTtyName: TestHyperstartTtySocket,
		Sockets:     sockets,
	}

	podConfig := PodConfig{
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
		ID:     "1",
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
		SockCtlName: TestHyperstartCtlSocket,
		SockTtyName: TestHyperstartTtySocket,
		Sockets:     sockets,
	}

	netConfig := NetworkConfig{
		NumInterfaces: 1,
	}

	podConfig := PodConfig{
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

func TestCreatePodNoopAgentSuccessful(t *testing.T) {
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
	config := PodConfig{}

	p, err := CreatePod(config)
	if p != nil || err == nil {
		t.Fatal()
	}
}

func TestDeletePodNoopAgentSuccessful(t *testing.T) {
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
	podDir := filepath.Join(configStoragePath, testPodID)
	os.Remove(podDir)

	p, err := DeletePod(testPodID)
	if p != nil || err == nil {
		t.Fatal()
	}
}

func TestStartPodNoopAgentSuccessful(t *testing.T) {
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

	p, err = StartPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}
}

func TestStartPodHyperstartAgentSuccessful(t *testing.T) {
	config := newTestPodConfigHyperstartAgent()

	pauseBinPath := filepath.Join(testDir, TestHyperstartPauseBinName)
	_, err := os.Create(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}

	hyperConfig := config.AgentConfig.(HyperConfig)
	hyperConfig.PauseBinPath = pauseBinPath
	config.AgentConfig = hyperConfig

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	p, err = StartPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	p.agent.(*hyper).bindUnmountAllRootfs()

	err = os.Remove(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartPodFailing(t *testing.T) {
	podDir := filepath.Join(configStoragePath, testPodID)
	os.Remove(podDir)

	p, err := StartPod(testPodID)
	if p != nil || err == nil {
		t.Fatal()
	}
}

func TestStopPodNoopAgentSuccessful(t *testing.T) {
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

	p, err = StartPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	p, err = StopPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}
}

func TestStopPodHyperstartAgentSuccessful(t *testing.T) {
	config := newTestPodConfigHyperstartAgent()

	pauseBinPath := filepath.Join(testDir, TestHyperstartPauseBinName)
	_, err := os.Create(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}

	hyperConfig := config.AgentConfig.(HyperConfig)
	hyperConfig.PauseBinPath = pauseBinPath
	config.AgentConfig = hyperConfig

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	p, err = StartPod(p.id)
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
	podDir := filepath.Join(configStoragePath, testPodID)
	os.Remove(podDir)

	p, err := StopPod(testPodID)
	if p != nil || err == nil {
		t.Fatal()
	}
}

func TestRunPodNoopAgentSuccessful(t *testing.T) {
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
	config := newTestPodConfigHyperstartAgent()

	pauseBinPath := filepath.Join(testDir, TestHyperstartPauseBinName)
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

	p.agent.(*hyper).bindUnmountAllRootfs()

	err = os.Remove(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunPodFailing(t *testing.T) {
	config := PodConfig{}

	p, err := RunPod(config)
	if p != nil || err == nil {
		t.Fatal()
	}
}

func TestListPodSuccessful(t *testing.T) {
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

func TestListPodFailing(t *testing.T) {
	os.RemoveAll(configStoragePath)

	_, err := ListPod()
	if err == nil {
		t.Fatal()
	}
}

func TestStatusPodSuccessful(t *testing.T) {
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

func TestListPodFailingFetchPodConfig(t *testing.T) {
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

func TestListPodFailingFetchPodState(t *testing.T) {
	config := newTestPodConfigNoop()

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(runStoragePath, p.id)
	os.RemoveAll(path)

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

	c, err := CreateContainer(p.id, contConfig)
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

	c, err := CreateContainer(p.id, contConfig)
	if c != nil || err == nil {
		t.Fatal(err)
	}
}

func TestDeleteContainerSuccessful(t *testing.T) {
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

	c, err := CreateContainer(p.id, contConfig)
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
	podDir := filepath.Join(configStoragePath, testPodID)
	contID := "100"
	os.RemoveAll(podDir)

	c, err := DeleteContainer(testPodID, contID)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestDeleteContainerFailingNoContainer(t *testing.T) {
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

	p, err = StartPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	c, err := CreateContainer(p.id, contConfig)
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
	podDir := filepath.Join(configStoragePath, testPodID)
	contID := "100"
	os.RemoveAll(podDir)

	c, err := StartContainer(testPodID, contID)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestStartContainerFailingNoContainer(t *testing.T) {
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

	c, err := CreateContainer(p.id, contConfig)
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

	p, err = StartPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	c, err := CreateContainer(p.id, contConfig)
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
	contID := "100"
	config := newTestPodConfigHyperstartAgent()

	pauseBinPath := filepath.Join(testDir, TestHyperstartPauseBinName)
	_, err := os.Create(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}

	hyperConfig := config.AgentConfig.(HyperConfig)
	hyperConfig.PauseBinPath = pauseBinPath
	config.AgentConfig = hyperConfig

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	p, err = StartPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	c, err := CreateContainer(p.id, contConfig)
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

	p.agent.(*hyper).bindUnmountAllRootfs()

	err = os.Remove(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartStopPodHyperstartAgentSuccessfulWithCNINetwork(t *testing.T) {
	config := newTestPodConfigHyperstartAgentCNINetwork()

	pauseBinPath := filepath.Join(testDir, TestHyperstartPauseBinName)
	_, err := os.Create(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}

	hyperConfig := config.AgentConfig.(HyperConfig)
	hyperConfig.PauseBinPath = pauseBinPath
	config.AgentConfig = hyperConfig

	p, err := CreatePod(config)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	p, err = StartPod(p.id)
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
	podDir := filepath.Join(configStoragePath, testPodID)
	contID := "100"
	os.RemoveAll(podDir)

	c, err := StopContainer(testPodID, contID)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestStopContainerFailingNoContainer(t *testing.T) {
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

	p, err = StartPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	c, err := CreateContainer(p.id, contConfig)
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

	p, err = StartPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	c, err := CreateContainer(p.id, contConfig)
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

	c, err = EnterContainer(p.id, contID, cmd)
	if c == nil || err != nil {
		t.Fatal(err)
	}
}

func TestEnterContainerHyperstartAgentSuccessful(t *testing.T) {
	contID := "100"
	config := newTestPodConfigHyperstartAgent()

	pauseBinPath := filepath.Join(testDir, TestHyperstartPauseBinName)
	_, err := os.Create(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}

	hyperConfig := config.AgentConfig.(HyperConfig)
	hyperConfig.PauseBinPath = pauseBinPath
	config.AgentConfig = hyperConfig

	p, err := CreatePod(config)
	if err != nil {
		t.Fatal(err)
	}

	podDir := filepath.Join(configStoragePath, p.id)
	_, err = os.Stat(podDir)
	if err != nil {
		t.Fatal(err)
	}

	p, err = StartPod(p.id)
	if err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	_, err = CreateContainer(p.id, contConfig)
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

	_, err = EnterContainer(p.id, contID, cmd)
	if err != nil {
		t.Fatal(err)
	}

	_, err = StopContainer(p.id, contID)
	if err != nil {
		t.Fatal(err)
	}

	p.agent.(*hyper).bindUnmountAllRootfs()

	err = os.Remove(pauseBinPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEnterContainerFailingNoPod(t *testing.T) {
	podDir := filepath.Join(configStoragePath, testPodID)
	contID := "100"
	os.RemoveAll(podDir)

	cmd := newBasicTestCmd()

	c, err := EnterContainer(testPodID, contID, cmd)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestEnterContainerFailingNoContainer(t *testing.T) {
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

	c, err := EnterContainer(p.id, contID, cmd)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestEnterContainerFailingContNotStarted(t *testing.T) {
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

	p, err = StartPod(p.id)
	if p == nil || err != nil {
		t.Fatal(err)
	}

	contConfig := newTestContainerConfigNoop(contID)

	c, err := CreateContainer(p.id, contConfig)
	if c == nil || err != nil {
		t.Fatal(err)
	}

	contDir := filepath.Join(podDir, contID)
	_, err = os.Stat(contDir)
	if err != nil {
		t.Fatal(err)
	}

	cmd := newBasicTestCmd()

	c, err = EnterContainer(p.id, contID, cmd)
	if c != nil || err == nil {
		t.Fatal()
	}
}

func TestStatusContainerSuccessful(t *testing.T) {
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

	c, err := CreateContainer(p.id, contConfig)
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

func createStartStopDeletePod(b *testing.B, podConfig PodConfig) {
	// Create pod
	p, err := CreatePod(podConfig)
	if err != nil {
		b.Logf("Could not create pod: %s", err)
	}

	// Start pod
	_, err = StartPod(p.id)
	if err != nil {
		b.Logf("Could not start pod: %s", err)
	}

	// Stop pod
	_, err = StopPod(p.id)
	if err != nil {
		b.Logf("Could not stop pod: %s", err)
	}

	// Delete pod
	_, err = DeletePod(p.id)
	if err != nil {
		b.Logf("Could not delete pod: %s", err)
	}
}

func createStartStopDeleteContainers(b *testing.B, podConfig PodConfig, contConfigs []ContainerConfig) {
	// Create pod
	p, err := CreatePod(podConfig)
	if err != nil {
		b.Logf("Could not create pod: %s", err)
	}

	// Start pod
	_, err = StartPod(p.id)
	if err != nil {
		b.Logf("Could not start pod: %s", err)
	}

	// Create containers
	for _, contConfig := range contConfigs {
		_, err := CreateContainer(p.id, contConfig)
		if err != nil {
			b.Logf("Could not create container %s: %s", contConfig.ID, err)
		}
	}

	// Start containers
	for _, contConfig := range contConfigs {
		_, err := StartContainer(p.id, contConfig.ID)
		if err != nil {
			b.Logf("Could not start container %s: %s", contConfig.ID, err)
		}
	}

	// Stop containers
	for _, contConfig := range contConfigs {
		_, err := StopContainer(p.id, contConfig.ID)
		if err != nil {
			b.Logf("Could not stop container %s: %s", contConfig.ID, err)
		}
	}

	// Delete containers
	for _, contConfig := range contConfigs {
		_, err := DeleteContainer(p.id, contConfig.ID)
		if err != nil {
			b.Logf("Could not delete container %s: %s", contConfig.ID, err)
		}
	}

	// Stop pod
	_, err = StopPod(p.id)
	if err != nil {
		b.Logf("Could not stop pod: %s", err)
	}

	// Delete pod
	_, err = DeletePod(p.id)
	if err != nil {
		b.Logf("Could not delete pod: %s", err)
	}
}

func BenchmarkCreateStartStopDeletePodQemuHypervisorHyperstartAgentNetworkCNI(b *testing.B) {
	for i := 0; i < b.N; i++ {
		podConfig := createNewPodConfig(QemuHypervisor, HyperstartAgent, HyperConfig{}, CNINetworkModel)
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
		podConfig := createNewPodConfig(QemuHypervisor, HyperstartAgent, HyperConfig{}, NoopNetworkModel)
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
		podConfig := createNewPodConfig(QemuHypervisor, HyperstartAgent, HyperConfig{}, NoopNetworkModel)
		contConfigs := createNewContainerConfigs(1)
		createStartStopDeleteContainers(b, podConfig, contConfigs)
	}
}

func BenchmarkStartStop10ContainerQemuHypervisorHyperstartAgentNetworkNoop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		podConfig := createNewPodConfig(QemuHypervisor, HyperstartAgent, HyperConfig{}, NoopNetworkModel)
		contConfigs := createNewContainerConfigs(10)
		createStartStopDeleteContainers(b, podConfig, contConfigs)
	}
}
