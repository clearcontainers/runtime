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

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/01org/ciao/ciao-controller/api"
	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/openstack/compute"
	"github.com/01org/ciao/payloads"

	"gopkg.in/yaml.v2"
)

var workloadCommand = &command{
	SubCommands: map[string]subCommand{
		"list":   new(workloadListCommand),
		"create": new(workloadCreateCommand),
	},
}

type workloadListCommand struct {
	Flag     flag.FlagSet
	template string
}

func (cmd *workloadListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] workload list

List all workloads

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a 

[]struct {
	OSFLVDISABLEDDisabled  bool    // Not used
	Disk                   string  // Backing images associated with workload
	OSFLVEXTDATAEphemeral  int     // Not currently used
	OsFlavorAccessIsPublic bool    // Indicates whether the workload is available to all tenants
	ID                     string  // ID of the workload
	Links                  []Link  // Not currently used
	Name                   string  // Name of the workload
	RAM                    int     // Amount of RAM allocated to instances of this workload 
	Swap                   string  // Not currently used
	Vcpus                  int     // Number of Vcpus allocated to instances of this workload 
}
`)
	fmt.Fprintln(os.Stderr, templateFunctionHelp)
	os.Exit(2)
}

func (cmd *workloadListCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *workloadListCommand) run(args []string) error {
	if *tenantID == "" {
		fatalf("Missing required -tenant-id parameter")
	}

	var flavors compute.FlavorsDetails
	if *tenantID == "" {
		*tenantID = "faketenant"
	}

	url := buildComputeURL("%s/flavors/detail", *tenantID)

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &flavors)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.template != "" {
		return outputToTemplate("workload-list", cmd.template,
			&flavors.Flavors)
	}

	for i, flavor := range flavors.Flavors {
		fmt.Printf("Workload %d\n", i+1)
		fmt.Printf("\tName: %s\n\tUUID:%s\n\tImage UUID: %s\n\tCPUs: %d\n\tMemory: %d MB\n",
			flavor.Name, flavor.ID, flavor.Disk, flavor.Vcpus, flavor.RAM)
	}
	return nil
}

type workloadCreateCommand struct {
	Flag     flag.FlagSet
	yamlFile string
}

func (cmd *workloadCreateCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.yamlFile, "yaml", "", "filename for yaml which describes the workload")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *workloadCreateCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] workload create [flags]

Create a new workload

The create flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func getCiaoWorkloadsResource() (string, error) {
	return getCiaoResource("workloads", api.WorkloadsV1)
}

type source struct {
	Type types.SourceType `yaml:"service"`
	ID   string           `yaml:"id"`
}

type disk struct {
	ID        *string `yaml:"volume_id"`
	Size      int     `yaml:"size"`
	Bootable  bool    `yaml:"bootable"`
	Source    *source `yaml:"source"`
	Ephemeral bool    `yaml:"ephemeral"`
}

type defaultResources struct {
	VCPUs int `yaml:"vcpus"`
	MemMB int `yaml:"mem_mb"`
}

// we currently only use the first disk due to lack of support
// in types.Workload for multiple storage resources.
type workloadOptions struct {
	Description     string           `yaml:"description"`
	VMType          string           `yaml:"vm_type"`
	FWType          string           `yaml:"fw_type"`
	ImageName       string           `yaml:"image_name"`
	ImageID         string           `yaml:"image_id"`
	Defaults        defaultResources `yaml:"defaults"`
	CloudConfigFile string           `yaml:"cloud_init"`
	Disks           []disk           `yaml:"disks"`
}

func optToReqStorage(opt workloadOptions) ([]types.StorageResource, error) {
	storage := make([]types.StorageResource, 0)
	bootableCount := 0
	for _, disk := range opt.Disks {
		res := types.StorageResource{
			Size:      disk.Size,
			Bootable:  disk.Bootable,
			Ephemeral: disk.Ephemeral,
		}

		if disk.Source != nil && disk.Source.Type == "" {
			disk.Source.Type = types.Empty
		}

		if disk.ID != nil {
			res.ID = *disk.ID
		} else if disk.Size == 0 {
			// source had better exist.
			if disk.Source == nil {
				return nil, errors.New("Invalid workload yaml: disk source may not be nil")
			}

			res.SourceType = disk.Source.Type
			res.SourceID = disk.Source.ID

			if res.SourceID != "" && res.SourceType == types.Empty {
				return nil, errors.New("Invalid workload yaml: when specifying a source ID a type must also be specified")
			}

			if res.SourceType != types.Empty && res.SourceID == "" {
				return nil, errors.New("Invalid workload yaml: when specifying a source type other than empty an id must also be specified")
			}
		} else {
			if disk.Bootable == true {
				// you may not request a bootable drive
				// from an empty source
				return nil, errors.New("Invalid workload yaml: empty disk source may not be bootable")
			}

			res.SourceType = types.Empty
		}

		if disk.Bootable {
			bootableCount++
		}

		storage = append(storage, res)
	}

	if payloads.Hypervisor(opt.VMType) == payloads.QEMU && bootableCount == 0 {
		return nil, errors.New("Invalid workload yaml: no bootable disks specified for a VM")
	}

	return storage, nil
}

func optToReq(opt workloadOptions, req *types.Workload) error {
	b, err := ioutil.ReadFile(opt.CloudConfigFile)
	if err != nil {
		return err
	}

	config := string(b)

	// this is where you'd validate that the options make
	// sense.
	req.Description = opt.Description
	req.VMType = payloads.Hypervisor(opt.VMType)
	req.FWType = opt.FWType
	req.ImageName = opt.ImageName
	req.ImageID = opt.ImageID
	req.Config = config
	req.Storage, err = optToReqStorage(opt)

	if err != nil {
		return err
	}

	// all default resources are required.
	defaults := opt.Defaults

	r := payloads.RequestedResource{
		Type:  payloads.VCPUs,
		Value: defaults.VCPUs,
	}
	req.Defaults = append(req.Defaults, r)

	r = payloads.RequestedResource{
		Type:  payloads.MemMB,
		Value: defaults.MemMB,
	}
	req.Defaults = append(req.Defaults, r)

	return nil
}

func (cmd *workloadCreateCommand) run(args []string) error {
	var opt workloadOptions
	var req types.Workload

	if cmd.yamlFile == "" {
		cmd.usage()
	}

	f, err := ioutil.ReadFile(cmd.yamlFile)
	if err != nil {
		fatalf("Unable to read workload config file: %s\n", err)
	}

	err = yaml.Unmarshal(f, &opt)
	if err != nil {
		fatalf("Config file invalid: %s\n", err)
	}

	err = optToReq(opt, &req)
	if err != nil {
		fatalf(err.Error())
	}

	b, err := json.Marshal(req)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(b)

	url, err := getCiaoWorkloadsResource()
	if err != nil {
		fatalf(err.Error())
	}

	ver := api.WorkloadsV1

	resp, err := sendCiaoRequest("POST", url, nil, body, &ver)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusCreated {
		fatalf("Workload creation failed: %s", resp.Status)
	}

	var workload types.WorkloadResponse

	err = unmarshalHTTPResponse(resp, &workload)
	if err != nil {
		fatalf(err.Error())
	}

	fmt.Printf("Created new workload: %s\n", workload.Workload.ID)

	return nil
}
