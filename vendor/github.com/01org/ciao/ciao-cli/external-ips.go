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
	"net"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/01org/ciao/ciao-controller/api"
	"github.com/01org/ciao/ciao-controller/types"
)

const (
	externalSubnetDesc = `struct {
	ID	string			// ID of the external subnet
	CIDR	string			// CIDR representation of the subnet
}`
	externalIPDesc = `struct {
	ID	string			// ID of the external IP
	Address	string			// the IPv4 Address
}`
	poolTemplateDesc = `struct {
	ID       string                 // ID of the pool (Admin only)
	Name     string                 // Name of the pool
	TotalIPs int                    // Total IPs in pool (Admin only)
	Free	 int                    // Total Free IPs in pool (Admin only)
}`
	poolShowTemplateDesc = `struct {
	ID	string			// ID of the pool
	Name	string			// name of the pool
	Free	int			// Total free IPs in pool
	TotalIPs int			// Total IPs in pool
	Subnets []ExternalSubnet	// Subnets in this pool
	IPs	[]ExternalIP		// Individual IPs in this pool
}`
	externalIPTemplate = `struct {
	ID		string		// ID of the mapped IP
	ExternalIP 	string		// External IP address
	InternalIP	string		// Internal IP address
	InstanceID	string		// ID of the instance that is mapped (Admin only)
	TenantID	string		// ID of the tenant (Admin only)
	PoolID		string		// ID of the allocation pool (Admin only)
	PoolName	string		// Name of the allocation pool (Admin only)
}`
)

func getCiaoExternalIPsResource() (string, string, error) {
	url, err := getCiaoResource("external-ips", api.ExternalIPsV1)
	return url, api.ExternalIPsV1, err
}

// TBD: in an ideal world, we'd modify the GET to take a query.
func getExternalIPRef(address string) (string, error) {
	var IPs []types.MappedIP

	url, ver, err := getCiaoExternalIPsResource()
	if err != nil {
		return "", err
	}

	resp, err := sendCiaoRequest("GET", url, nil, nil, &ver)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("External IP list failed: %s", resp.Status)
	}

	err = unmarshalHTTPResponse(resp, &IPs)
	if err != nil {
		return "", err
	}

	for _, IP := range IPs {
		if IP.ExternalIP == address {
			url := getRef("self", IP.Links)
			if url != "" {
				return url, nil
			}
		}
	}

	return "", types.ErrAddressNotFound
}

var externalIPCommand = &command{
	SubCommands: map[string]subCommand{
		"map":   new(externalIPMapCommand),
		"list":  new(externalIPListCommand),
		"unmap": new(externalIPUnMapCommand),
	},
}

type externalIPMapCommand struct {
	Flag       flag.FlagSet
	instanceID string
	poolName   string
}

func (cmd *externalIPMapCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] external-ip map [flags]

Map an external IP from a given pool to an instance.

The map flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *externalIPMapCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.instanceID, "instance", "", "ID of the instance to map IP to.")
	cmd.Flag.StringVar(&cmd.poolName, "pool", "", "Name of the pool to map from.")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *externalIPMapCommand) run(args []string) error {
	if cmd.instanceID == "" {
		errorf("Missing required -instance parameter")
		cmd.usage()
	}

	req := types.MapIPRequest{
		InstanceID: cmd.instanceID,
	}

	if cmd.poolName != "" {
		req.PoolName = &cmd.poolName
	}

	b, err := json.Marshal(req)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(b)

	url, ver, err := getCiaoExternalIPsResource()
	if err != nil {
		fatalf(err.Error())
	}

	resp, err := sendCiaoRequest("POST", url, nil, body, &ver)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusNoContent {
		fatalf("External IP map failed: %s", resp.Status)
	}

	fmt.Printf("Requested external IP for: %s\n", cmd.instanceID)

	return nil
}

type externalIPListCommand struct {
	Flag     flag.FlagSet
	template string
}

func (cmd *externalIPListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] external-ip list [flags]

List all mapped external IPs.

The list flags are:

`)
	cmd.Flag.PrintDefaults()

	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a

[]%s
`, externalIPTemplate)
	fmt.Fprintln(os.Stderr, templateFunctionHelp)
	os.Exit(2)
}

func (cmd *externalIPListCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *externalIPListCommand) run(args []string) error {
	var IPs []types.MappedIP

	url, ver, err := getCiaoExternalIPsResource()
	if err != nil {
		fatalf(err.Error())
	}

	resp, err := sendCiaoRequest("GET", url, nil, nil, &ver)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		fatalf("External IP list failed: %s", resp.Status)
	}

	err = unmarshalHTTPResponse(resp, &IPs)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.template != "" {
		return outputToTemplate("external-ip-list", cmd.template,
			&IPs)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 1, 1, ' ', 0)
	fmt.Fprintf(w, "#\tExternalIP\tInternalIP")
	if checkPrivilege() {
		fmt.Fprintf(w, "\tInstanceID\tTenantID\tPoolName\n")
	} else {
		fmt.Fprintf(w, "\n")
	}

	for i, IP := range IPs {
		fmt.Fprintf(w, "%d", i+1)
		fmt.Fprintf(w, "\t%s", IP.ExternalIP)
		fmt.Fprintf(w, "\t%s", IP.InternalIP)
		if IP.InstanceID != "" {
			fmt.Fprintf(w, "\t%s", IP.InstanceID)
		}

		if IP.TenantID != "" {
			fmt.Fprintf(w, "\t%s", IP.TenantID)
		}

		if IP.PoolName != "" {
			fmt.Fprintf(w, "\t%s", IP.PoolName)
		}

		fmt.Fprintf(w, "\n")
	}

	w.Flush()

	return nil
}

type externalIPUnMapCommand struct {
	address string
	Flag    flag.FlagSet
}

func (cmd *externalIPUnMapCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] external-ip unmap [flags]

Unmap a given external IP.

The unmap flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *externalIPUnMapCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.address, "address", "", "External IP to unmap.")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *externalIPUnMapCommand) run(args []string) error {
	if cmd.address == "" {
		errorf("Missing required -instance parameter")
		cmd.usage()
	}

	url, err := getExternalIPRef(cmd.address)
	if err != nil {
		fatalf(err.Error())
	}

	ver := api.ExternalIPsV1

	resp, err := sendCiaoRequest("DELETE", url, nil, nil, &ver)
	if err != nil {
		fatalf(err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Unmap of address failed: %s", resp.Status)
	}

	fmt.Printf("Requested unmap of: %s\n", cmd.address)

	return nil
}

var poolCommand = &command{
	SubCommands: map[string]subCommand{
		"create": new(poolCreateCommand),
		"list":   new(poolListCommand),
		"show":   new(poolShowCommand),
		"delete": new(poolDeleteCommand),
		"add":    new(poolAddCommand),
		"remove": new(poolRemoveCommand),
	},
}

type poolCreateCommand struct {
	Flag flag.FlagSet
	name string
}

func getCiaoPoolsResource() (string, error) {
	return getCiaoResource("pools", api.PoolsV1)
}

func getCiaoPoolRef(name string) (string, error) {
	var pools types.ListPoolsResponse

	query := queryValue{
		name:  "name",
		value: name,
	}

	url, err := getCiaoPoolsResource()
	if err != nil {
		return "", err
	}

	ver := api.PoolsV1

	resp, err := sendCiaoRequest("GET", url, []queryValue{query}, nil, &ver)
	if err != nil {
		return "", err
	}

	err = unmarshalHTTPResponse(resp, &pools)
	if err != nil {
		return "", err
	}

	// we have now the pool ID
	if len(pools.Pools) != 1 {
		return "", errors.New("No pool by that name found")
	}

	links := pools.Pools[0].Links
	url = getRef("self", links)
	if url == "" {
		return url, errors.New("Invalid Link returned from controller")
	}

	return url, nil
}

func getCiaoPool(name string) (types.Pool, error) {
	var pool types.Pool

	url, err := getCiaoPoolRef(name)
	if err != nil {
		return pool, nil
	}

	ver := api.PoolsV1

	resp, err := sendCiaoRequest("GET", url, nil, nil, &ver)
	if err != nil {
		return pool, err
	}

	err = unmarshalHTTPResponse(resp, &pool)
	if err != nil {
		return pool, err
	}

	return pool, nil
}

func (cmd *poolCreateCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] pool create [flags]

Creates a new external IP pool.

The create flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

// TBD: add support for specifying a subnet or []ip addresses.
func (cmd *poolCreateCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Name of pool")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *poolCreateCommand) run(args []string) error {
	if cmd.name == "" {
		errorf("Missing required -name parameter")
		cmd.usage()
	}

	req := types.NewPoolRequest{
		Name: cmd.name,
	}

	b, err := json.Marshal(req)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(b)

	url, err := getCiaoPoolsResource()
	if err != nil {
		fatalf(err.Error())
	}

	ver := api.PoolsV1

	resp, err := sendCiaoRequest("POST", url, nil, body, &ver)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusNoContent {
		fatalf("Pool creation failed: %s", resp.Status)
	}

	fmt.Printf("Created new pool: %s\n", cmd.name)

	return nil
}

type poolListCommand struct {
	Flag     flag.FlagSet
	template string
}

func (cmd *poolListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] pool list [flags]

List all ciao external IP pools.

The list flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a

[]%s
`, poolTemplateDesc)
	fmt.Fprintln(os.Stderr, templateFunctionHelp)

	os.Exit(2)
}

func (cmd *poolListCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

// change this command to return different output depending
// on the privilege level of user. Check privilege, then
// if not privileged, build non-privileged URL.
func (cmd *poolListCommand) run(args []string) error {
	var pools types.ListPoolsResponse

	url, err := getCiaoPoolsResource()
	if err != nil {
		fatalf(err.Error())
	}

	ver := api.PoolsV1

	resp, err := sendCiaoRequest("GET", url, nil, nil, &ver)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &pools)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.template != "" {
		return outputToTemplate("pool-list", cmd.template,
			&pools.Pools)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 1, 1, ' ', 0)
	fmt.Fprintf(w, "#\tName")
	if checkPrivilege() {
		fmt.Fprintf(w, "\tTotalIPs\tFreeIPs\n")
	} else {
		fmt.Fprintf(w, "\n")
	}

	for i, pool := range pools.Pools {
		fmt.Fprintf(w, "%d", i+1)
		fmt.Fprintf(w, "\t%s", pool.Name)

		if pool.TotalIPs != nil {
			fmt.Fprintf(w, "\t%d", *pool.TotalIPs)
		}

		if pool.Free != nil {
			fmt.Fprintf(w, "\t%d", *pool.Free)
		}

		fmt.Fprintf(w, "\n")
	}

	w.Flush()

	return nil
}

type poolShowCommand struct {
	Flag     flag.FlagSet
	name     string
	template string
}

func (cmd *poolShowCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] pool show [flags]

Show ciao external IP pool details.

The show flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a

%s
`, poolShowTemplateDesc)
	fmt.Fprintf(os.Stderr, `
The externalSubnets are described by

%s
`, externalSubnetDesc)

	fmt.Fprintf(os.Stderr, `
The externalIPs are described by

%s
`, externalIPDesc)
	fmt.Fprintln(os.Stderr, templateFunctionHelp)

	os.Exit(2)
}

func (cmd *poolShowCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Name of pool")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func dumpPool(pool types.Pool) {
	fmt.Printf("\tUUID: %s\n", pool.ID)
	fmt.Printf("\tName: %s\n", pool.Name)
	fmt.Printf("\tFree IPs: %d\n", pool.Free)
	fmt.Printf("\tTotal IPs: %d\n", pool.TotalIPs)

	for _, sub := range pool.Subnets {
		fmt.Printf("\tSubnet: %s\n", sub.CIDR)
	}

	for _, ip := range pool.IPs {
		fmt.Printf("\tIP Address: %s\n", ip.Address)
	}
}

func (cmd *poolShowCommand) run(args []string) error {
	var pool types.Pool

	if cmd.name == "" {
		errorf("Missing required -name parameter")
		cmd.usage()
	}

	pool, err := getCiaoPool(cmd.name)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.template != "" {
		return outputToTemplate("pool-show", cmd.template,
			&pool)
	}

	dumpPool(pool)

	return nil
}

type poolDeleteCommand struct {
	Flag flag.FlagSet
	name string
}

func (cmd *poolDeleteCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] pool delete [flags]

Delete an unused ciao external IP pool.

The delete flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *poolDeleteCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Name of pool")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *poolDeleteCommand) run(args []string) error {
	if cmd.name == "" {
		errorf("Missing required -name parameter")
		cmd.usage()
	}

	url, err := getCiaoPoolRef(cmd.name)
	if err != nil {
		fatalf(err.Error())
	}

	ver := api.PoolsV1

	resp, err := sendCiaoRequest("DELETE", url, nil, nil, &ver)
	if err != nil {
		fatalf(err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		fatalf("Pool deletion failed: %s", resp.Status)
	}

	fmt.Printf("Deleted pool: %s\n", cmd.name)

	return nil
}

type poolAddCommand struct {
	Flag   flag.FlagSet
	name   string
	subnet string
	ips    []string
}

func (cmd *poolAddCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] pool add [flags] [ip1 ip2...]

Add external IPs to a pool.

The add flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *poolAddCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Name of pool")
	cmd.Flag.StringVar(&cmd.subnet, "subnet", "", "Subnet in CIDR format")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *poolAddCommand) run(args []string) error {
	var req types.NewAddressRequest

	if cmd.name == "" {
		errorf("Missing required -name parameter")
		cmd.usage()
	}

	url, err := getCiaoPoolRef(cmd.name)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.subnet != "" {
		// verify it's a good address.
		_, _, err := net.ParseCIDR(cmd.subnet)
		if err != nil {
			fatalf(err.Error())
		}

		req.Subnet = &cmd.subnet
	} else if len(args) < 1 {
		errorf("Missing any addresses to add")
		cmd.usage()
	} else {
		for _, addr := range args {
			// verify it's a good address
			IP := net.ParseIP(addr)
			if IP == nil {
				fatalf("Invalid IP address")
			}

			newAddr := types.NewIPAddressRequest{
				IP: addr,
			}

			req.IPs = append(req.IPs, newAddr)
		}
	}

	b, err := json.Marshal(req)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(b)

	ver := api.PoolsV1

	resp, err := sendCiaoRequest("POST", url, nil, body, &ver)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusNoContent {
		fatalf("Adding address failed: %s", resp.Status)
	}

	fmt.Printf("Added new address to: %s\n", cmd.name)

	return nil
}

type poolRemoveCommand struct {
	Flag   flag.FlagSet
	name   string
	subnet string
	ip     string
}

func (cmd *poolRemoveCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] pool remove [flags]

Remove unmapped external IPs from a pool.

The remove flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *poolRemoveCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Name of pool")
	cmd.Flag.StringVar(&cmd.subnet, "subnet", "", "Subnet in CIDR format")
	cmd.Flag.StringVar(&cmd.ip, "ip", "", "IPv4 Address")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func getSubnetRef(pool types.Pool, cidr string) string {
	for _, sub := range pool.Subnets {
		if sub.CIDR == cidr {
			return getRef("self", sub.Links)
		}
	}

	return ""
}

func getIPRef(pool types.Pool, address string) string {
	for _, ip := range pool.IPs {
		if ip.Address == address {
			return getRef("self", ip.Links)
		}
	}

	return ""
}

func (cmd *poolRemoveCommand) run(args []string) error {
	if cmd.name == "" {
		errorf("Missing required -name parameter")
		cmd.usage()
	}

	if cmd.subnet == "" && cmd.ip == "" {
		errorf("You must specify subnet or ip address to remove")
		cmd.usage()
	}

	if cmd.subnet != "" && cmd.ip != "" {
		errorf("You can only remove one item at a time")
		cmd.usage()
	}

	pool, err := getCiaoPool(cmd.name)
	if err != nil {
		fatalf(err.Error())
	}

	var url string

	if cmd.subnet != "" {
		url = getSubnetRef(pool, cmd.subnet)
	}

	if cmd.ip != "" {
		url = getIPRef(pool, cmd.ip)
	}

	if url == "" {
		fatalf("Address not present")
	}

	ver := api.PoolsV1

	resp, err := sendCiaoRequest("DELETE", url, nil, nil, &ver)
	if err != nil {
		fatalf(err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		fatalf("Address removal failed: %s", resp.Status)
	}

	fmt.Printf("Removed address from pool: %s\n", cmd.name)
	return nil
}
