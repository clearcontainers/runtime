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
	"flag"
	"fmt"
	"os"
	"text/template"

	"github.com/01org/ciao/ciao-controller/types"
)

const cnciTemplateDesc = `struct {
	ID        string // UUID of network node
	TenantID  string // UUID of tenant to which this CNCI pertains
	IPv4      string // IP address of CNCI
	Geography string // Physical location of the network node
	Subnets   []struct {
		Subnet string // Subnet used by the CNCI
	}
}
`

const computeTemplateDesc = `struct {
	ID                    string // UUID of the compute node
	Timestamp             time.Time
	Status                string // Node status, e.g., READY, FULL
	MemTotal              int    // Total amount of RAM on Compute Node in MB
	MemAvailable          int    // Memory available on Compute Node in MB
	DiskTotal             int    // Total amount of Disk Space on Compute Node in MB
	DiskAvailable         int    // Disk Space available on Compute Node in MB
	Load                  int    // Compute node load
	OnlineCPUs            int    // Number of CPUs
	TotalInstances        int    // Number of instances hosted by the Compute Node
	TotalRunningInstances int    // Number of running instances
	TotalPendingInstances int    // Number of pending instances
	TotalPausedInstances  int    // Number of paused instances
}
`

var nodeCommand = &command{
	SubCommands: map[string]subCommand{
		"list":   new(nodeListCommand),
		"status": new(nodeStatusCommand),
		"show":   new(nodeShowCommand),
	},
}

type nodeListCommand struct {
	Flag     flag.FlagSet
	compute  bool
	cnci     bool
	nodeID   bool
	template string
}

func (cmd *nodeListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] node list

List nodes

The list flags are:
`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on one of the following types:

--cnci

[]%s

--compute

[]%s
`, cnciTemplateDesc, computeTemplateDesc)
	fmt.Fprintln(os.Stderr, templateFunctionHelp)
	os.Exit(2)
}

func (cmd *nodeListCommand) parseArgs(args []string) []string {
	cmd.Flag.BoolVar(&cmd.compute, "compute", false, "List all compute nodes")
	cmd.Flag.BoolVar(&cmd.cnci, "cnci", false, "List all CNCIs")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *nodeListCommand) run(args []string) error {
	t := createTemplate("node-list", cmd.template)

	if cmd.compute {
		return listComputeNodes(t)
	}
	if cmd.cnci {
		return listCNCINodes(t)
	}
	cmd.usage()
	return nil
}

func listComputeNodes(t *template.Template) error {
	var nodes types.CiaoComputeNodes

	url := buildComputeURL("nodes")

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &nodes)
	if err != nil {
		fatalf(err.Error())
	}

	if t != nil {
		if err := t.Execute(os.Stdout, &nodes.Nodes); err != nil {
			fatalf(err.Error())
		}
		return nil
	}

	for i, node := range nodes.Nodes {
		fmt.Printf("Compute Node %d\n", i+1)
		fmt.Printf("\tUUID: %s\n", node.ID)
		fmt.Printf("\tStatus: %s\n", node.Status)
		fmt.Printf("\tLoad: %d\n", node.Load)
		fmt.Printf("\tAvailable/Total memory: %d/%d MB\n", node.MemAvailable, node.MemTotal)
		fmt.Printf("\tAvailable/Total disk: %d/%d MB\n", node.DiskAvailable, node.DiskTotal)
		fmt.Printf("\tTotal Instances: %d\n", node.TotalInstances)
		fmt.Printf("\t\tRunning Instances: %d\n", node.TotalRunningInstances)
		fmt.Printf("\t\tPending Instances: %d\n", node.TotalPendingInstances)
		fmt.Printf("\t\tPaused Instances: %d\n", node.TotalPausedInstances)
	}
	return nil
}

func listCNCINodes(t *template.Template) error {
	var cncis types.CiaoCNCIs

	url := buildComputeURL("cncis")

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &cncis)
	if err != nil {
		fatalf(err.Error())
	}

	if t != nil {
		if err := t.Execute(os.Stdout, &cncis.CNCIs); err != nil {
			fatalf(err.Error())
		}
		return nil
	}

	for i, cnci := range cncis.CNCIs {
		fmt.Printf("CNCI %d\n", i+1)
		fmt.Printf("\tCNCI UUID: %s\n", cnci.ID)
		fmt.Printf("\tTenant UUID: %s\n", cnci.TenantID)
		fmt.Printf("\tIPv4: %s\n", cnci.IPv4)
		fmt.Printf("\tSubnets:\n")
		for _, subnet := range cnci.Subnets {
			fmt.Printf("\t\t%s\n", subnet.Subnet)
		}
	}
	return nil
}

type nodeStatusCommand struct {
	Flag     flag.FlagSet
	template string
}

func (cmd *nodeStatusCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] node status

Show cluster status
`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a

struct {
	TotalNodes            int
	TotalNodesReady       int
	TotalNodesFull        int
	TotalNodesOffline     int
	TotalNodesMaintenance int
}
`)
	fmt.Fprintln(os.Stderr, templateFunctionHelp)
	os.Exit(2)
}

func (cmd *nodeStatusCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *nodeStatusCommand) run(args []string) error {
	var status types.CiaoClusterStatus
	url := buildComputeURL("nodes/summary")

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &status)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.template != "" {
		return outputToTemplate("node-status", cmd.template,
			&status.Status)
	}

	fmt.Printf("Total Nodes %d\n", status.Status.TotalNodes)
	fmt.Printf("\tReady %d\n", status.Status.TotalNodesReady)
	fmt.Printf("\tFull %d\n", status.Status.TotalNodesFull)
	fmt.Printf("\tOffline %d\n", status.Status.TotalNodesOffline)
	fmt.Printf("\tMaintenance %d\n", status.Status.TotalNodesMaintenance)

	return nil
}

type nodeShowCommand struct {
	Flag     flag.FlagSet
	cnci     bool
	nodeID   string
	template string
}

func (cmd *nodeShowCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] node show

Show info about a node

The show flags are:
`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on one of the following types:

--cnci

%s
`, cnciTemplateDesc)
	fmt.Fprintln(os.Stderr, templateFunctionHelp)
	os.Exit(2)
}

func (cmd *nodeShowCommand) parseArgs(args []string) []string {
	cmd.Flag.BoolVar(&cmd.cnci, "cnci", false, "Show info about a cnci node")
	cmd.Flag.StringVar(&cmd.nodeID, "node-id", "", "Node ID")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *nodeShowCommand) run(args []string) error {
	if cmd.cnci {
		return showCNCINode(cmd)
	}

	cmd.usage()
	return nil
}

func showCNCINode(cmd *nodeShowCommand) error {
	if cmd.nodeID == "" {
		fatalf("Missing required -cnci-id parameter")
	}

	var cnci types.CiaoCNCI

	url := buildComputeURL("cncis/%s/detail", cmd.nodeID)

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &cnci)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.template != "" {
		return outputToTemplate("node-show", cmd.template,
			&cnci)
	}

	fmt.Printf("\tCNCI UUID: %s\n", cnci.ID)
	fmt.Printf("\tTenant UUID: %s\n", cnci.TenantID)
	fmt.Printf("\tIPv4: %s\n", cnci.IPv4)
	fmt.Printf("\tSubnets:\n")
	for _, subnet := range cnci.Subnets {
		fmt.Printf("\t\t%s\n", subnet.Subnet)
	}
	return nil
}
