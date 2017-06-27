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

	"github.com/urfave/cli"
)

type kernelModule struct {
	// description
	desc string

	// maps parameter names to values
	parameters map[string]string
}

const (
	moduleParamDir = "parameters"
	cpuFlagsTag    = "flags"
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

// return details of the first CPU
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

	matched := pattern.MatchString(haystack)

	if matched {
		return true
	}

	return false
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
	if err == nil {
		return true
	}

	return false
}

func checkCPU(tag, cpuinfo string, attribs map[string]string) error {
	if cpuinfo == "" {
		return fmt.Errorf("Need cpuinfo")
	}

	for attrib, desc := range attribs {
		found := findAnchoredString(cpuinfo, attrib)
		if found {
			ccLog.Infof("Found CPU %v %q (%s)", tag, desc, attrib)
		} else {
			return fmt.Errorf("CPU does not have required %v: %q (%s)", tag, desc, attrib)
		}
	}

	return nil
}
func checkCPUFlags(cpuflags string, required map[string]string) error {
	return checkCPU("flag", cpuflags, required)
}

func checkCPUAttribs(cpuinfo string, attribs map[string]string) error {
	return checkCPU("attribute", cpuinfo, attribs)
}

func checkKernelModules(modules map[string]kernelModule) error {
	for module, details := range modules {
		if !haveKernelModule(module) {
			return fmt.Errorf("kernel module %q (%s) not found", module, details.desc)
		}

		ccLog.Infof("Found kernel module %q (%s)", details.desc, module)

		for param, expected := range details.parameters {
			path := filepath.Join(sysModuleDir, module, moduleParamDir, param)
			value, err := getFileContents(path)
			if err != nil {
				return err
			}

			value = strings.TrimRight(value, "\n\r")

			if value == expected {
				ccLog.Infof("Kernel module %q parameter %q has correct value", details.desc, param)
			} else {
				return fmt.Errorf("kernel module %q parameter %q has value %q (expected %q)", details.desc, param, value, expected)
			}
		}
	}

	return nil
}

// hostIsClearContainersCapable determines if the system is capable of
// running Clear Containers.
func hostIsClearContainersCapable(cpuinfoFile string) error {
	cpuinfo, err := getCPUInfo(cpuinfoFile)
	if err != nil {
		return err
	}

	if err = checkCPUAttribs(cpuinfo, requiredCPUAttribs); err != nil {
		return err
	}

	cpuFlags := getCPUFlags(cpuinfo)
	if cpuFlags == "" {
		return fmt.Errorf("Cannot find CPU flags")
	}

	if err = checkCPUFlags(cpuFlags, requiredCPUFlags); err != nil {
		return err
	}

	if err = checkKernelModules(requiredKernelModules); err != nil {
		return err
	}

	return nil
}

var ccCheckCommand = cli.Command{
	Name:  "cc-check",
	Usage: "tests if system can run " + project,
	Action: func(context *cli.Context) error {
		err := hostIsClearContainersCapable(procCPUInfo)
		if err != nil {
			return fmt.Errorf("ERROR: %v", err)
		}

		ccLog.Info("")
		ccLog.Info("System is capable of running " + project)

		return nil
	},
}
