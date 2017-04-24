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

// Package bat contains a number of helper functions that can be used to perform
// various operations on a ciao cluster such as creating an instance or retrieving
// a list of all the defined workloads, etc.  All of these helper functions are
// implemented by calling ciao-cli rather than by using ciao's REST APIs.  This
// package is mainly intended for use by BAT tests.  Manipulating the cluster
// via ciao-cli, rather than through the REST APIs, allows us to test a little
// bit more of ciao.
package bat

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// Source is provided to the disk structure to indicate whether the
// disk should be cloned from a volume or an image.
type Source struct {
	Type string `yaml:"service"`
	ID   string `yaml:"id"`
}

// Disk describes the storage for the workload definition.
type Disk struct {
	ID        *string `yaml:"volume_id"`
	Size      int     `yaml:"size"`
	Bootable  bool    `yaml:"bootable"`
	Source    *Source `yaml:"source"`
	Ephemeral bool    `yaml:"ephemeral"`
}

// DefaultResources indicate how many cpus and mem to allocate.
type DefaultResources struct {
	VCPUs int `yaml:"vcpus"`
	MemMB int `yaml:"mem_mb"`
}

// WorkloadOptions is used to generate a workload definition in yaml.
type WorkloadOptions struct {
	Description     string           `yaml:"description"`
	VMType          string           `yaml:"vm_type"`
	FWType          string           `yaml:"fw_type"`
	ImageName       string           `yaml:"image_id"`
	Defaults        DefaultResources `yaml:"defaults"`
	CloudConfigFile string           `yaml:"cloud_init"`
	Disks           []Disk           `yaml:"disks"`
}

// this function should output the yaml file for the workload definition
func createWorkloadDefinition(filename string, opt WorkloadOptions) error {
	// take the cloud config file and output it to a file.
	y, err := yaml.Marshal(opt)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, y, 0644)
}

// this function should output a cloud init file for the workload definition
func createCloudConfig(filename string, config string) error {
	// take the cloud config file and output it to a file.
	// convert string to bytes.
	return ioutil.WriteFile(filename, []byte(config), 0644)
}

func createWorkload(ctx context.Context, tenant string, opt WorkloadOptions, config string, public bool) (ID string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// create temp dir for the yaml files
	dir, err := ioutil.TempDir("", "ciao_bat")
	if err != nil {
		return "", err
	}

	// you have to call workload create from the same working dir
	// as you have placed your yaml files, so change into that dir.
	err = os.Chdir(dir)
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}

	// change back to cwd and clean up temp dir on exit.
	defer func() {
		chdirErr := os.Chdir(cwd)
		if err == nil {
			err = chdirErr
		}

		os.RemoveAll(dir)
	}()

	// write out the cloud config.
	err = createCloudConfig("config.yaml", config)
	if err != nil {
		return "", err
	}

	opt.CloudConfigFile = "config.yaml"

	// write out the workload definition
	err = createWorkloadDefinition("workload.yaml", opt)
	if err != nil {
		return "", err
	}

	// send the workload create command to ciao-cli
	args := []string{"workload", "create", "-yaml", "workload.yaml"}

	var data []byte
	if public {
		data, err = RunCIAOCLIAsAdmin(ctx, tenant, args)
	} else {
		data, err = RunCIAOCLI(ctx, tenant, args)
	}

	if err != nil {
		return "", err
	}

	// get the returned workload ID
	s := strings.SplitAfter(string(data), ":")
	if len(s) != 2 {
		return "", fmt.Errorf("Problem parsing workload ID")
	}

	return strings.TrimSpace(s[1]), nil
}

// CreatePublicWorkload will call ciao-cli as admin to create a workload.
// It will first output the cloud init yaml file to the current working
// directory. Then it will output the workload definition to the current
// working directory. Finally it will call ciao-cli workload create -yaml
// to upload the workload definition. It will clean up all the files it
// created when it is done.
func CreatePublicWorkload(ctx context.Context, tenant string, opt WorkloadOptions, config string) (string, error) {
	return createWorkload(ctx, "", opt, config, true)
}

// CreateWorkload will call ciao-cli to create a workload definition.
// It will first output the cloud init yaml file to the current working
// directory. Then it will output the workload definition to the current
// working directory. Finally it will call ciao-cli workload create -yaml
// to upload the workload definition. It will clean up all the files it
// created when it is done.
func CreateWorkload(ctx context.Context, tenant string, opt WorkloadOptions, config string) (string, error) {
	return createWorkload(ctx, tenant, opt, config, false)
}

// GetAllWorkloads retrieves a list of all workloads in the cluster by calling
// ciao-cli workload list.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
func GetAllWorkloads(ctx context.Context, tenant string) ([]Workload, error) {
	var workloads []Workload

	args := []string{"workload", "list", "-f", "{{tojson .}}"}
	err := RunCIAOCLIJS(ctx, tenant, args, &workloads)
	if err != nil {
		return nil, err
	}

	return workloads, nil
}

// GetWorkloadByName will return a specific workload referenced by name.
// An error will be returned if either no workloads exist in the cluster,
// or if the specific workload does not exist. It inherits all error
// conditions of GetAllWorkloads.
func GetWorkloadByName(ctx context.Context, tenant string, name string) (Workload, error) {
	wls, err := GetAllWorkloads(ctx, tenant)
	if err != nil {
		return Workload{}, err
	}

	if len(wls) == 0 {
		return Workload{}, fmt.Errorf("No workloads defined for tenant %s", tenant)
	}

	for _, w := range wls {
		if w.Name == name {
			return w, nil
		}
	}

	return Workload{}, fmt.Errorf("No matching workload for %s", name)
}

// GetWorkloadByID will return a specific workload referenced by name.
// An error will be returned if either no workloads exist in the cluster,
// or if the specific workload does not exist. It inherits all error
// conditions of GetAllWorkloads.
func GetWorkloadByID(ctx context.Context, tenant string, ID string) (Workload, error) {
	wls, err := GetAllWorkloads(ctx, tenant)
	if err != nil {
		return Workload{}, err
	}

	if len(wls) == 0 {
		return Workload{}, fmt.Errorf("No workloads defined for tenant %s", tenant)
	}

	for _, w := range wls {
		if w.ID == ID {
			return w, nil
		}
	}

	return Workload{}, fmt.Errorf("No matching workload for %s", ID)
}
