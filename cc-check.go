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

package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	vc "github.com/containers/virtcontainers"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type kernelModule struct {
	// description
	desc string

	// maps parameter names to values
	parameters map[string]string
}

const (
	moduleParamDir        = "parameters"
	cpuFlagsTag           = "flags"
	successMessage        = "System is capable of running " + project
	failMessage           = "System is not capable of running " + project
	kernelPropertyCorrect = "Kernel property value correct"
)

// variables rather than consts to allow tests to modify them
var (
	procCPUInfo  = "/proc/cpuinfo"
	sysModuleDir = "/sys/module"
	modInfoCmd   = "modinfo"
)

// requiredCPUFlags maps a CPU flag value to search for and a
// human-readable description of that value.
var requiredCPUFlags = map[string]string{
	"vmx":    "Virtualization support",
	"lm":     "64Bit CPU",
	"sse4_1": "SSE4.1",
}

// requiredCPUAttribs maps a CPU (non-CPU flag) attribute value to search for
// and a human-readable description of that value.
var requiredCPUAttribs = map[string]string{
	"GenuineIntel": "Intel Architecture CPU",
}

// requiredKernelModules maps a required module name to a human-readable
// description of the modules functionality and an optional list of
// required module parameters.
var requiredKernelModules = map[string]kernelModule{
	"kvm": {
		desc: "Kernel-based Virtual Machine",
	},
	"kvm_intel": {
		desc: "Intel KVM",
		parameters: map[string]string{
			"nested":             "Y",
			"unrestricted_guest": "Y",
		},
	},
	"vhost": {
		desc: "Host kernel accelerator for virtio",
	},
	"vhost_net": {
		desc: "Host kernel accelerator for virtio network",
	},
}

// getCPUInfo returns details of the first CPU
func getCPUInfo(cpuInfoFile string) (string, error) {
	text, err := getFileContents(cpuInfoFile)
	if err != nil {
		return "", err
	}

	cpus := strings.SplitAfter(text, "\n\n")

	trimmed := strings.TrimSpace(cpus[0])
	if trimmed == "" {
		return "", fmt.Errorf("Cannot determine CPU details")
	}

	return trimmed, nil
}

func findAnchoredString(haystack, needle string) bool {
	if haystack == "" || needle == "" {
		return false
	}

	// Ensure the search string is anchored
	pattern := regexp.MustCompile(`\b` + needle + `\b`)

	return pattern.MatchString(haystack)
}

func getCPUFlags(cpuinfo string) string {
	for _, line := range strings.Split(cpuinfo, "\n") {
		if strings.HasPrefix(line, cpuFlagsTag) {
			fields := strings.Split(line, ":")
			if len(fields) == 2 {
				return strings.TrimSpace(fields[1])
			}
		}
	}

	return ""
}

func haveKernelModule(module string) bool {
	// First, check to see if the module is already loaded
	path := filepath.Join(sysModuleDir, module)
	if fileExists(path) {
		return true
	}

	// Now, check if the module is unloaded, but available
	cmd := exec.Command(modInfoCmd, module)
	err := cmd.Run()
	return err == nil
}

// checkCPU checks all required CPU attributes modules and returns a count of
// the number of CPU attribute errors (all of which are logged by this
// function). Only fatal errors result in an error return.
func checkCPU(tag, cpuinfo string, attribs map[string]string) (count uint32, err error) {
	if cpuinfo == "" {
		return 0, fmt.Errorf("Need cpuinfo")
	}

	for attrib, desc := range attribs {
		fields := logrus.Fields{
			"type":        tag,
			"name":        attrib,
			"description": desc,
		}

		found := findAnchoredString(cpuinfo, attrib)
		if !found {
			ccLog.WithFields(fields).Errorf("CPU property not found")
			count++
			continue

		}

		ccLog.WithFields(fields).Infof("CPU property found")
	}

	return count, nil
}

func checkCPUFlags(cpuflags string, required map[string]string) (uint32, error) {
	return checkCPU("flag", cpuflags, required)
}

func checkCPUAttribs(cpuinfo string, attribs map[string]string) (uint32, error) {
	return checkCPU("attribute", cpuinfo, attribs)
}

// checkKernelModules checks all required kernel modules modules and returns a count of
// the number of module errors (all of which are logged by this
// function). Only fatal errors result in an error return.
func checkKernelModules(modules map[string]kernelModule) (count uint32, err error) {
	onVMM, err := vc.RunningOnVMM(procCPUInfo)
	if err != nil {
		return 0, err
	}

	for module, details := range modules {
		fields := logrus.Fields{
			"type":        "module",
			"name":        module,
			"description": details.desc,
		}

		if !haveKernelModule(module) {
			ccLog.WithFields(fields).Error("kernel property not found")
			count++
			continue
		}

		ccLog.WithFields(fields).Infof("kernel property found")

		for param, expected := range details.parameters {
			path := filepath.Join(sysModuleDir, module, moduleParamDir, param)
			value, err := getFileContents(path)
			if err != nil {
				return 0, err
			}

			value = strings.TrimRight(value, "\n\r")

			if value != expected {
				fields["expected"] = expected
				fields["actual"] = value
				fields["parameter"] = param

				msg := "kernel module parameter has unexpected value"

				// this option is not required when
				// already running under a hypervisor.
				if param == "unrestricted_guest" && onVMM {
					ccLog.WithFields(fields).Warn(kernelPropertyCorrect)
					continue
				}

				if param == "nested" {
					ccLog.WithFields(fields).Warn(msg)
					continue
				}

				ccLog.WithFields(fields).Error(msg)
				count++
			}

			ccLog.WithFields(fields).Info(kernelPropertyCorrect)
		}
	}

	return count, nil
}

// hostIsClearContainersCapable determines if the system is capable of
// running Clear Containers.
func hostIsClearContainersCapable(cpuinfoFile string) error {
	cpuinfo, err := getCPUInfo(cpuinfoFile)
	if err != nil {
		return err
	}

	cpuFlags := getCPUFlags(cpuinfo)
	if cpuFlags == "" {
		return fmt.Errorf("Cannot find CPU flags")
	}

	// Keep a track of the error count, but don't error until all tests
	// have been performed!
	errorCount := uint32(0)

	count, err := checkCPUAttribs(cpuinfo, requiredCPUAttribs)
	if err != nil {
		return err
	}

	errorCount += count

	count, err = checkCPUFlags(cpuFlags, requiredCPUFlags)
	if err != nil {
		return err
	}

	errorCount += count

	count, err = checkKernelModules(requiredKernelModules)
	if err != nil {
		return err
	}

	errorCount += count

	if errorCount == 0 {
		return nil
	}

	return fmt.Errorf("ERROR: %s", failMessage)
}

var ccCheckCLICommand = cli.Command{
	Name:  "cc-check",
	Usage: "tests if system can run " + project,
	Action: func(context *cli.Context) error {
		err := hostIsClearContainersCapable(procCPUInfo)
		if err != nil {
			return err
		}

		ccLog.Info(successMessage)

		return nil
	},
}
