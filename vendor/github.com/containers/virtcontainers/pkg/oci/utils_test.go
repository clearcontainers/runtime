//
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
//

package oci

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	vc "github.com/containers/virtcontainers"
	"github.com/kubernetes-incubator/cri-o/pkg/annotations"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const tempBundlePath = "/tmp/virtc/ocibundle/"
const containerID = "virtc-oci-test"
const consolePath = "/tmp/virtc/console"
const fileMode = os.FileMode(0640)
const dirMode = os.FileMode(0750)

func createConfig(fileName string, fileData string) (string, error) {
	configPath := path.Join(tempBundlePath, fileName)

	err := ioutil.WriteFile(configPath, []byte(fileData), fileMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create config file %s %v\n", configPath, err)
		return "", err
	}

	return configPath, nil
}

func TestMinimalPodConfig(t *testing.T) {
	configPath, err := createConfig("config.json", minimalConfig)
	if err != nil {
		t.Fatal(err)
	}

	runtimeConfig := RuntimeConfig{
		HypervisorType: vc.QemuHypervisor,
		AgentType:      vc.HyperstartAgent,
		ProxyType:      vc.CCProxyType,
		ShimType:       vc.CCShimType,
		Console:        consolePath,
	}

	expectedCmd := vc.Cmd{
		Args: []string{"sh"},
		Envs: []vc.EnvVar{
			{
				Var:   "PATH",
				Value: "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			},
			{
				Var:   "TERM",
				Value: "xterm",
			},
		},
		WorkDir:             "/",
		User:                "0",
		PrimaryGroup:        "0",
		SupplementaryGroups: []string{"10", "29"},
		Interactive:         true,
		Console:             consolePath,
	}

	expectedMounts := []vc.Mount{
		{
			Source:      "proc",
			Destination: "/proc",
			Type:        "proc",
			Options:     nil,
			HostPath:    "",
		},
		{
			Source:      "tmpfs",
			Destination: "/dev",
			Type:        "tmpfs",
			Options:     []string{"nosuid", "strictatime", "mode=755", "size=65536k"},
			HostPath:    "",
		},
		{
			Source:      "devpts",
			Destination: "/dev/pts",
			Type:        "devpts",
			Options:     []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620", "gid=5"},
			HostPath:    "",
		},
	}

	expectedContainerConfig := vc.ContainerConfig{
		ID:             containerID,
		RootFs:         path.Join(tempBundlePath, "rootfs"),
		ReadonlyRootfs: true,
		Cmd:            expectedCmd,
		Annotations: map[string]string{
			ConfigPathKey:    configPath,
			BundlePathKey:    tempBundlePath,
			ContainerTypeKey: string(vc.PodSandbox),
		},
		Mounts: expectedMounts,
	}

	expectedNetworkConfig := vc.NetworkConfig{
		NumInterfaces: 1,
	}

	expectedPodConfig := vc.PodConfig{
		ID:       containerID,
		Hostname: "testHostname",

		HypervisorType: vc.QemuHypervisor,
		AgentType:      vc.HyperstartAgent,
		ProxyType:      vc.CCProxyType,
		ShimType:       vc.CCShimType,

		NetworkModel:  vc.CNMNetworkModel,
		NetworkConfig: expectedNetworkConfig,

		Containers: []vc.ContainerConfig{expectedContainerConfig},

		Annotations: map[string]string{
			ConfigPathKey: configPath,
			BundlePathKey: tempBundlePath,
		},
	}

	ociSpec, err := ParseConfigJSON(tempBundlePath)
	if err != nil {
		t.Fatalf("Could not parse config.json: %v", err)
	}

	podConfig, err := PodConfig(ociSpec, runtimeConfig, tempBundlePath, containerID, consolePath, false)
	if err != nil {
		t.Fatalf("Could not create Pod configuration %v", err)
	}

	if reflect.DeepEqual(podConfig, expectedPodConfig) == false {
		t.Fatalf("Got %v\n expecting %v", podConfig, expectedPodConfig)
	}

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestVmConfig(t *testing.T) {
	var limitBytes int64 = 128 * 1024 * 1024
	var quota int64 = 200000
	var period uint64 = 100000

	config := RuntimeConfig{
		VMConfig: vc.Resources{
			Memory: 2048,
		},
	}

	expectedResources := vc.Resources{
		Memory: 128,
		VCPUs:  2,
	}

	ocispec := CompatOCISpec{
		Spec: specs.Spec{
			Linux: &specs.Linux{
				Resources: &specs.LinuxResources{
					Memory: &specs.LinuxMemory{
						Limit: &limitBytes,
					},
					CPU: &specs.LinuxCPU{
						Quota:  &quota,
						Period: &period,
					},
				},
			},
		},
	}

	resources, err := vmConfig(ocispec, config)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(resources, expectedResources) == false {
		t.Fatalf("Got %v\n expecting %v", resources, expectedResources)
	}

	limitBytes = -128 * 1024 * 1024
	ocispec.Linux.Resources.Memory.Limit = &limitBytes

	resources, err = vmConfig(ocispec, config)
	if err == nil {
		t.Fatalf("Got %v\n expecting error", resources)
	}

	limitBytes = 128 * 1024 * 1024
	quota = -1
	ocispec.Linux.Resources.CPU.Quota = &quota

	resources, err = vmConfig(ocispec, config)
	if err == nil {
		t.Fatalf("Got %v\n expecting error", resources)
	}

	quota = 100000
	period = 0
	ocispec.Linux.Resources.CPU.Quota = &quota

	resources, err = vmConfig(ocispec, config)
	if err == nil {
		t.Fatalf("Got %v\n expecting error", resources)
	}
}

func testStatusToOCIStateSuccessful(t *testing.T, cStatus vc.ContainerStatus, expected specs.State) {
	ociState := StatusToOCIState(cStatus)

	if reflect.DeepEqual(ociState, expected) == false {
		t.Fatalf("Got %v\n expecting %v", ociState, expected)
	}
}

func TestStatusToOCIStateSuccessfulWithReadyState(t *testing.T) {
	configPath, err := createConfig("config.json", minimalConfig)
	if err != nil {
		t.Fatal(err)
	}

	testContID := "testContID"
	testPID := 12345
	testRootFs := "testRootFs"

	state := vc.State{
		State: vc.StateReady,
	}

	containerAnnotations := map[string]string{
		ConfigPathKey: configPath,
		BundlePathKey: tempBundlePath,
	}

	cStatus := vc.ContainerStatus{
		ID:          testContID,
		State:       state,
		PID:         testPID,
		RootFs:      testRootFs,
		Annotations: containerAnnotations,
	}

	expected := specs.State{
		Version:     specs.Version,
		ID:          testContID,
		Status:      "created",
		Pid:         testPID,
		Bundle:      tempBundlePath,
		Annotations: containerAnnotations,
	}

	testStatusToOCIStateSuccessful(t, cStatus, expected)

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestStatusToOCIStateSuccessfulWithRunningState(t *testing.T) {
	configPath, err := createConfig("config.json", minimalConfig)
	if err != nil {
		t.Fatal(err)
	}

	testContID := "testContID"
	testPID := 12345
	testRootFs := "testRootFs"

	state := vc.State{
		State: vc.StateRunning,
	}

	containerAnnotations := map[string]string{
		ConfigPathKey: configPath,
		BundlePathKey: tempBundlePath,
	}

	cStatus := vc.ContainerStatus{
		ID:          testContID,
		State:       state,
		PID:         testPID,
		RootFs:      testRootFs,
		Annotations: containerAnnotations,
	}

	expected := specs.State{
		Version:     specs.Version,
		ID:          testContID,
		Status:      "running",
		Pid:         testPID,
		Bundle:      tempBundlePath,
		Annotations: containerAnnotations,
	}

	testStatusToOCIStateSuccessful(t, cStatus, expected)

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestStatusToOCIStateSuccessfulWithStoppedState(t *testing.T) {
	configPath, err := createConfig("config.json", minimalConfig)
	if err != nil {
		t.Fatal(err)
	}

	testContID := "testContID"
	testPID := 12345
	testRootFs := "testRootFs"

	state := vc.State{
		State: vc.StateStopped,
	}

	containerAnnotations := map[string]string{
		ConfigPathKey: configPath,
		BundlePathKey: tempBundlePath,
	}

	cStatus := vc.ContainerStatus{
		ID:          testContID,
		State:       state,
		PID:         testPID,
		RootFs:      testRootFs,
		Annotations: containerAnnotations,
	}

	expected := specs.State{
		Version:     specs.Version,
		ID:          testContID,
		Status:      "stopped",
		Pid:         testPID,
		Bundle:      tempBundlePath,
		Annotations: containerAnnotations,
	}

	testStatusToOCIStateSuccessful(t, cStatus, expected)

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestStatusToOCIStateSuccessfulWithNoState(t *testing.T) {
	configPath, err := createConfig("config.json", minimalConfig)
	if err != nil {
		t.Fatal(err)
	}

	testContID := "testContID"
	testPID := 12345
	testRootFs := "testRootFs"

	containerAnnotations := map[string]string{
		ConfigPathKey: configPath,
		BundlePathKey: tempBundlePath,
	}

	cStatus := vc.ContainerStatus{
		ID:          testContID,
		PID:         testPID,
		RootFs:      testRootFs,
		Annotations: containerAnnotations,
	}

	expected := specs.State{
		Version:     specs.Version,
		ID:          testContID,
		Status:      "",
		Pid:         testPID,
		Bundle:      tempBundlePath,
		Annotations: containerAnnotations,
	}

	testStatusToOCIStateSuccessful(t, cStatus, expected)

	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestStateToOCIState(t *testing.T) {
	var state vc.State

	if ociState := StateToOCIState(state); ociState != "" {
		t.Fatalf("Expecting \"created\" state, got \"%s\"", ociState)
	}

	state.State = vc.StateReady
	if ociState := StateToOCIState(state); ociState != "created" {
		t.Fatalf("Expecting \"created\" state, got \"%s\"", ociState)
	}

	state.State = vc.StateRunning
	if ociState := StateToOCIState(state); ociState != "running" {
		t.Fatalf("Expecting \"created\" state, got \"%s\"", ociState)
	}

	state.State = vc.StateStopped
	if ociState := StateToOCIState(state); ociState != "stopped" {
		t.Fatalf("Expecting \"created\" state, got \"%s\"", ociState)
	}
}

func TestEnvVars(t *testing.T) {
	envVars := []string{"foo=bar", "TERM=xterm", "HOME=/home/foo", "TERM=\"bar\"", "foo=\"\""}
	expectecVcEnvVars := []vc.EnvVar{
		{
			Var:   "foo",
			Value: "bar",
		},
		{
			Var:   "TERM",
			Value: "xterm",
		},
		{
			Var:   "HOME",
			Value: "/home/foo",
		},
		{
			Var:   "TERM",
			Value: "\"bar\"",
		},
		{
			Var:   "foo",
			Value: "\"\"",
		},
	}

	vcEnvVars, err := EnvVars(envVars)
	if err != nil {
		t.Fatalf("Could not create environment variable slice %v", err)
	}

	if reflect.DeepEqual(vcEnvVars, expectecVcEnvVars) == false {
		t.Fatalf("Got %v\n expecting %v", vcEnvVars, expectecVcEnvVars)
	}
}

func TestMalformedEnvVars(t *testing.T) {
	envVars := []string{"foo"}
	r, err := EnvVars(envVars)
	if err == nil {
		t.Fatalf("EnvVars() succeeded unexpectedly: [%s] variable=%s value=%s", envVars[0], r[0].Var, r[0].Value)
	}

	envVars = []string{"TERM="}
	r, err = EnvVars(envVars)
	if err == nil {
		t.Fatalf("EnvVars() succeeded unexpectedly: [%s] variable=%s value=%s", envVars[0], r[0].Var, r[0].Value)
	}

	envVars = []string{"=foo"}
	r, err = EnvVars(envVars)
	if err == nil {
		t.Fatalf("EnvVars() succeeded unexpectedly: [%s] variable=%s value=%s", envVars[0], r[0].Var, r[0].Value)
	}

	envVars = []string{"=foo="}
	r, err = EnvVars(envVars)
	if err == nil {
		t.Fatalf("EnvVars() succeeded unexpectedly: [%s] variable=%s value=%s", envVars[0], r[0].Var, r[0].Value)
	}
}

func TestGetConfigPath(t *testing.T) {
	expected := filepath.Join(tempBundlePath, "config.json")

	configPath := getConfigPath(tempBundlePath)

	if configPath != expected {
		t.Fatalf("Got %s, Expecting %s", configPath, expected)
	}
}

func testGetContainerTypeSuccessful(t *testing.T, annotations map[string]string, expected vc.ContainerType) {
	containerType, err := GetContainerType(annotations)
	if err != nil {
		t.Fatal(err)
	}

	if containerType != expected {
		t.Fatalf("Got %s, Expecting %s", containerType, expected)
	}
}

func TestGetContainerTypePodSandbox(t *testing.T) {
	annotations := map[string]string{
		ContainerTypeKey: string(vc.PodSandbox),
	}

	testGetContainerTypeSuccessful(t, annotations, vc.PodSandbox)
}

func TestGetContainerTypePodContainer(t *testing.T) {
	annotations := map[string]string{
		ContainerTypeKey: string(vc.PodContainer),
	}

	testGetContainerTypeSuccessful(t, annotations, vc.PodContainer)
}

func TestGetContainerTypeFailure(t *testing.T) {
	expected := vc.UnknownContainerType

	containerType, err := GetContainerType(map[string]string{})
	if err == nil {
		t.Fatalf("This test should fail because annotations is empty")
	}

	if containerType != expected {
		t.Fatalf("Got %s, Expecting %s", containerType, expected)
	}
}

func testContainerTypeSuccessful(t *testing.T, ociSpec CompatOCISpec, expected vc.ContainerType) {
	containerType, err := ociSpec.ContainerType()
	if err != nil {
		t.Fatal(err)
	}

	if containerType != expected {
		t.Fatalf("Got %s, Expecting %s", containerType, expected)
	}
}

func TestContainerTypePodSandbox(t *testing.T) {
	var ociSpec CompatOCISpec

	ociSpec.Annotations = map[string]string{
		annotations.ContainerType: annotations.ContainerTypeSandbox,
	}

	testContainerTypeSuccessful(t, ociSpec, vc.PodSandbox)
}

func TestContainerTypePodContainer(t *testing.T) {
	var ociSpec CompatOCISpec

	ociSpec.Annotations = map[string]string{
		annotations.ContainerType: annotations.ContainerTypeContainer,
	}

	testContainerTypeSuccessful(t, ociSpec, vc.PodContainer)
}

func TestContainerTypePodSandboxEmptyAnnotation(t *testing.T) {
	testContainerTypeSuccessful(t, CompatOCISpec{}, vc.PodSandbox)
}

func TestContainerTypeFailure(t *testing.T) {
	var ociSpec CompatOCISpec
	expected := vc.UnknownContainerType
	unknownType := "unknown_type"

	ociSpec.Annotations = map[string]string{
		annotations.ContainerType: unknownType,
	}

	containerType, err := ociSpec.ContainerType()
	if err == nil {
		t.Fatalf("This test should fail because the container type is %s", unknownType)
	}

	if containerType != expected {
		t.Fatalf("Got %s, Expecting %s", containerType, expected)
	}
}

func TestPodIDSuccessful(t *testing.T) {
	var ociSpec CompatOCISpec
	testPodID := "testPodID"

	ociSpec.Annotations = map[string]string{
		annotations.SandboxID: testPodID,
	}

	podID, err := ociSpec.PodID()
	if err != nil {
		t.Fatal(err)
	}

	if podID != testPodID {
		t.Fatalf("Got %s, Expecting %s", podID, testPodID)
	}
}

func TestPodIDFailure(t *testing.T) {
	var ociSpec CompatOCISpec

	podID, err := ociSpec.PodID()
	if err == nil {
		t.Fatalf("This test should fail because annotations is empty")
	}

	if podID != "" {
		t.Fatalf("Got %s, Expecting empty pod ID", podID)
	}
}

func TestAddKernelParamValid(t *testing.T) {
	var config RuntimeConfig

	expected := []vc.Param{
		{"foo", "bar"},
	}

	err := config.AddKernelParam(expected[0])
	if err != nil || reflect.DeepEqual(config.HypervisorConfig.KernelParams, expected) == false {
		t.Fatal()
	}
}

func TestAddKernelParamInvalid(t *testing.T) {
	var config RuntimeConfig

	invalid := []vc.Param{
		{"", "bar"},
	}

	err := config.AddKernelParam(invalid[0])
	if err == nil {
		t.Fatal()
	}
}

func TestMain(m *testing.M) {
	/* Create temp bundle directory if necessary */
	err := os.MkdirAll(tempBundlePath, dirMode)
	if err != nil {
		fmt.Printf("Unable to create %s %v\n", tempBundlePath, err)
		os.Exit(1)
	}

	defer os.RemoveAll(tempBundlePath)

	os.Exit(m.Run())
}
