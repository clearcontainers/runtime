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
	"github.com/01org/ciao/templateutils"

	"gopkg.in/yaml.v2"
)

var workloadCommand = &command{
	SubCommands: map[string]subCommand{
		"list":   new(workloadListCommand),
		"create": new(workloadCreateCommand),
		"delete": new(workloadDeleteCommand),
		"show":   new(workloadShowCommand),
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
	fmt.Fprintf(os.Stderr, "\n%s",
		templateutils.GenerateUsageDecorated("f", compute.FlavorsDetails{}.Flavors, nil))
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
		return templateutils.OutputToTemplate(os.Stdout, "workload-list", cmd.template,
			&flavors.Flavors, nil)
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
	ID        *string `yaml:"volume_id,omitempty"`
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
	FWType          string           `yaml:"fw_type,omitempty"`
	ImageName       string           `yaml:"image_name,omitempty"`
	ImageID         string           `yaml:"image_id,omitempty"`
	Defaults        defaultResources `yaml:"defaults"`
	CloudConfigFile string           `yaml:"cloud_init,omitempty"`
	Disks           []disk           `yaml:"disks,omitempty"`
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

func outputWorkload(w types.Workload) {
	var opt workloadOptions

	opt.Description = w.Description
	opt.VMType = string(w.VMType)
	opt.FWType = w.FWType
	opt.ImageName = w.ImageName
	opt.ImageID = w.ImageID
	for _, d := range w.Defaults {
		if d.Type == payloads.VCPUs {
			opt.Defaults.VCPUs = d.Value
		} else if d.Type == payloads.MemMB {
			opt.Defaults.MemMB = d.Value
		}
	}

	for _, s := range w.Storage {
		d := disk{
			Size:      s.Size,
			Bootable:  s.Bootable,
			Ephemeral: s.Ephemeral,
		}
		if s.ID != "" {
			d.ID = &s.ID
		}

		src := source{
			Type: s.SourceType,
			ID:   s.SourceID,
		}

		d.Source = &src

		opt.Disks = append(opt.Disks, d)
	}

	b, err := yaml.Marshal(opt)
	if err != nil {
		fatalf(err.Error())
	}

	fmt.Println(string(b))
	fmt.Println(w.Config)
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

type workloadDeleteCommand struct {
	Flag     flag.FlagSet
	workload string
}

func (cmd *workloadDeleteCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] workload delete [flags]

Deletes a given workload

The delete flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *workloadDeleteCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.workload, "workload", "", "Workload UUID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *workloadDeleteCommand) run(args []string) error {
	if cmd.workload == "" {
		cmd.usage()
	}

	url, err := getCiaoWorkloadsResource()
	if err != nil {
		fatalf(err.Error())
	}

	ver := api.WorkloadsV1

	// you should do a get first and search for the workload,
	// then use the href - but not with the currently used
	// OpenStack API. Until we support GET with a ciao API,
	// just hard code the path.
	url = fmt.Sprintf("%s/%s", url, cmd.workload)

	resp, err := sendCiaoRequest("DELETE", url, nil, nil, &ver)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusNoContent {
		fatalf("Workload deletion failed: %s", resp.Status)
	}

	return nil
}

type workloadShowCommand struct {
	Flag     flag.FlagSet
	template string
	workload string
}

func (cmd *workloadShowCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] workload show

Show workload details

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s",
		templateutils.GenerateUsageDecorated("f", types.Workload{}, nil))
	os.Exit(2)
}

func (cmd *workloadShowCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.workload, "workload", "", "Workload UUID")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *workloadShowCommand) run(args []string) error {
	var wl types.Workload

	if cmd.workload == "" {
		cmd.usage()
	}

	url, err := getCiaoWorkloadsResource()
	if err != nil {
		fatalf(err.Error())
	}

	ver := api.WorkloadsV1

	// you should do a get first and search for the workload,
	// then use the href - but not with the currently used
	// OpenStack API. Until we support GET with a ciao API,
	// just hard code the path.
	url = fmt.Sprintf("%s/%s", url, cmd.workload)

	resp, err := sendCiaoRequest("GET", url, nil, nil, &ver)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		fatalf("Workload show failed: %s", resp.Status)
	}

	err = unmarshalHTTPResponse(resp, &wl)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.template != "" {
		return templateutils.OutputToTemplate(os.Stdout, "workload-show", cmd.template, &wl, nil)
	}

	outputWorkload(wl)
	return nil
}
