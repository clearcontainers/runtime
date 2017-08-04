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
	"strings"
)

// HypervisorType describes an hypervisor type.
type HypervisorType string

const (
	// QemuHypervisor is the QEMU hypervisor.
	QemuHypervisor HypervisorType = "qemu"

	// MockHypervisor is a mock hypervisor for testing purposes
	MockHypervisor HypervisorType = "mock"
)

const (
	defaultVCPUs = 1
	// 2 GiB
	defaultMemSzMiB = 2048
)

// deviceType describes a virtualized device type.
type deviceType int

const (
	// ImgDev is the image device type.
	imgDev deviceType = iota

	// FsDev is the filesystem device type.
	fsDev

	// NetDev is the network device type.
	netDev

	// SerialDev is the serial device type.
	serialDev

	// BlockDev is the block device type.
	blockDev

	// ConsoleDev is the console device type.
	consoleDev

	// SerialPortDev is the serial port device type.
	serialPortDev
)

// Set sets an hypervisor type based on the input string.
func (hType *HypervisorType) Set(value string) error {
	switch value {
	case "qemu":
		*hType = QemuHypervisor
		return nil
	case "mock":
		*hType = MockHypervisor
		return nil
	default:
		return fmt.Errorf("Unknown hypervisor type %s", value)
	}
}

// String converts an hypervisor type to a string.
func (hType *HypervisorType) String() string {
	switch *hType {
	case QemuHypervisor:
		return string(QemuHypervisor)
	case MockHypervisor:
		return string(MockHypervisor)
	default:
		return ""
	}
}

// newHypervisor returns an hypervisor from and hypervisor type.
func newHypervisor(hType HypervisorType) (hypervisor, error) {
	switch hType {
	case QemuHypervisor:
		return &qemu{}, nil
	case MockHypervisor:
		return &mockHypervisor{}, nil
	default:
		return nil, fmt.Errorf("Unknown hypervisor type %s", hType)
	}
}

// Param is a key/value representation for hypervisor and kernel parameters.
type Param struct {
	Key   string
	Value string
}

// HypervisorConfig is the hypervisor configuration.
type HypervisorConfig struct {
	// KernelPath is the guest kernel host path.
	KernelPath string

	// ImagePath is the guest image host path.
	ImagePath string

	// HypervisorPath is the hypervisor executable host path.
	HypervisorPath string

	// DisableBlockDeviceUse disallows a block device from being used.
	DisableBlockDeviceUse bool

	// KernelParams are additional guest kernel parameters.
	KernelParams []Param

	// HypervisorParams are additional hypervisor parameters.
	HypervisorParams []Param

	// HypervisorMachineType specifies the type of machine being
	// emulated.
	HypervisorMachineType string

	// Debug changes the default hypervisor and kernel parameters to
	// enable debug output where available.
	Debug bool

	// DefaultVCPUs specifies default number of vCPUs for the VM.
	// Pod configuration VMConfig.VCPUs overwrites this.
	DefaultVCPUs uint32

	// DefaultMem specifies default memory size in MiB for the VM.
	// Pod configuration VMConfig.Memory overwrites this.
	DefaultMemSz uint32
}

func (conf *HypervisorConfig) valid() (bool, error) {
	if conf.KernelPath == "" {
		return false, fmt.Errorf("Missing kernel path")
	}

	if conf.ImagePath == "" {
		return false, fmt.Errorf("Missing image path")
	}

	if conf.DefaultVCPUs == 0 {
		conf.DefaultVCPUs = defaultVCPUs
	}

	if conf.DefaultMemSz == 0 {
		conf.DefaultMemSz = defaultMemSzMiB
	}

	return true, nil
}

// AddKernelParam allows the addition of new kernel parameters to an existing
// hypervisor configuration.
func (conf *HypervisorConfig) AddKernelParam(p Param) error {
	if p.Key == "" {
		return fmt.Errorf("Empty kernel parameter")
	}

	conf.KernelParams = append(conf.KernelParams, p)

	return nil
}

func appendParam(params []Param, parameter string, value string) []Param {
	return append(params, Param{parameter, value})
}

// SerializeParams converts []Param to []string
func SerializeParams(params []Param, delim string) []string {
	var parameters []string

	for _, p := range params {
		if p.Key == "" && p.Value == "" {
			continue
		} else if p.Key == "" {
			parameters = append(parameters, fmt.Sprintf("%s", p.Value))
		} else if p.Value == "" {
			parameters = append(parameters, fmt.Sprintf("%s", p.Key))
		} else if delim == "" {
			parameters = append(parameters, fmt.Sprintf("%s", p.Key))
			parameters = append(parameters, fmt.Sprintf("%s", p.Value))
		} else {
			parameters = append(parameters, fmt.Sprintf("%s%s%s", p.Key, delim, p.Value))
		}
	}

	return parameters
}

// DeserializeParams converts []string to []Param
func DeserializeParams(parameters []string) []Param {
	var params []Param

	for _, param := range parameters {
		if param == "" {
			continue
		}
		p := strings.SplitN(param, "=", 2)
		if len(p) == 2 {
			params = append(params, Param{Key: p[0], Value: p[1]})
		} else {
			params = append(params, Param{Key: p[0], Value: ""})
		}
	}

	return params
}

// hypervisor is the virtcontainers hypervisor interface.
// The default hypervisor implementation is Qemu.
type hypervisor interface {
	init(config HypervisorConfig) error
	createPod(podConfig PodConfig) error
	startPod(startCh, stopCh chan struct{}) error
	stopPod() error
	pausePod() error
	resumePod() error
	addDevice(devInfo interface{}, devType deviceType) error
	getPodConsole(podID string) string
}
