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
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/openstack/compute"
	"github.com/01org/ciao/templateutils"
)

const (
	osStart  = "os-start"
	osStop   = "os-stop"
	osDelete = "os-delete"
)

var instanceCommand = &command{
	SubCommands: map[string]subCommand{
		"add":     new(instanceAddCommand),
		"delete":  new(instanceDeleteCommand),
		"list":    new(instanceListCommand),
		"show":    new(instanceShowCommand),
		"restart": new(instanceRestartCommand),
		"stop":    new(instanceStopCommand),
	},
}

type volumeFlag struct {
	uuid      string
	bootIndex string
	swap      bool
	local     bool
	ephemeral bool
	size      int
	tag       string
}
type volumeFlagSlice []volumeFlag

func printVolumeFlagUsage() {
	msg := `
The volume flag allows specification of a volume to be attached
to a workload instance.  Sub-options include 'uuid', 'boot_index',
'swap', 'ephemeral', 'local', and 'size'.  Ephemeral volumes are
automatically removed when an instance is removed.  Local volumes
are constrained by a size which is a resource demand considered
when scheduling a workload instance.  Size is an integer number of
gigabytes.  The boot_index may be \"none\" or a negative integer to
exclude the volume from boot, otherwise use a positive integer
to indicate a relative ordering among multiple specified volumes.

Valid combinations include:
	-volume uuid=${UUID}[,boot_index=N]
	-volume uuid=${UUID},swap
	-volume uuid=${UUID},ephemeral[,boot_index=N]
	-volume size=${SIZE}
	-volume ephemeral,size=${SIZE}
	-volume local,ephemeral,size=${SIZE}
	-volume swap,size=${SIZE}
	-volume local,swap,size=${SIZE}

Multiple -volume arguments may be specified per workload instance.

`
	fmt.Fprintf(os.Stderr, msg)
}

func argMatch(patterns []string, arg string) bool {
	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, arg)
		if matched {
			return true
		}
	}
	return false
}
func isInstanceVolumeBoolArg(arg string) bool {
	if isInstanceVolumeImplicitBoolArg(arg) {
		return true
	}

	return isInstanceVolumeExplicitBoolArg(arg)
}

func isInstanceVolumeImplicitBoolArg(arg string) bool {
	patterns := []string{
		"^swap$",
		"^local$",
		"^ephemeral$",
	}
	return argMatch(patterns, arg)
}
func isInstanceVolumeExplicitBoolArg(arg string) bool {
	patterns := []string{
		"^swap=(true|false)$",
		"^local=(true|false)$",
		"^ephemeral=(true|false)$",
	}
	return argMatch(patterns, arg)
}
func isInstanceVolumeIntArg(arg string) bool {
	patterns := []string{
		"^size=[0-9]+$",
	}
	return argMatch(patterns, arg)
}
func isInstanceVolumeStringArg(arg string) bool {
	patterns := []string{
		"^uuid=.*$",
		"^boot_index=.*$",
		"^tag=.*$",
	}
	return argMatch(patterns, arg)
}
func getInstanceVolumeImplicitBoolArgs(subArg string, boolArgsMap map[string]bool) bool {
	// search for implicit affirmative bools by name, put in map
	if !isInstanceVolumeImplicitBoolArg(subArg) {
		return false
	}

	boolArgsMap[subArg] = true
	return true
}
func getInstanceVolumeExplicitBoolArgs(key string, val string, boolArgsMap map[string]bool) (bool, error) {
	// search for explicit bools by name, put in map
	if !isInstanceVolumeImplicitBoolArg(key) {
		return false, nil
	}
	fullArg := key + "=" + val
	if !isInstanceVolumeExplicitBoolArg(fullArg) {
		return false, fmt.Errorf("Invalid argument. Expected %s={true|false}, got \"%s=%s\"",
			key, key, val)
	}

	if boolArgsMap[key] != false {
		return false, fmt.Errorf("Conflicting arguments. Already had \"%s=%t\", got additional  \"%s=%s\"",
			key, boolArgsMap[key], key, val)
	}

	if val == "true" {
		boolArgsMap[key] = true
		return true, nil
	} else if val == "false" {
		boolArgsMap[key] = false
		return true, nil
	}

	return false, fmt.Errorf("Invalid argument. Expected %s={true|false}, got \"%s=%s\"",
		key, key, val)
}
func getInstanceVolumeIntegerArgs(key string, val string, intArgsMap map[string]int) (bool, error) {
	// search for integer args by name, put in map
	if key != "size" {
		return false, nil
	}

	if intArgsMap[key] != 0 {
		return false, fmt.Errorf("Conflicting arguments. Already had \"%s=%d\", got additional \"%s=%s\"",
			key, intArgsMap[key], key, val)
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		return false, fmt.Errorf("Invalid argument. Expected %s={integer}, got \"%s=%s\": %s", key, key, val, err)
	}

	intArgsMap[key] = i
	return true, nil
}
func getInstanceVolumeStringArgs(key string, val string, stringArgsMap map[string]string) (bool, error) {
	if stringArgsMap[key] != "" {
		return false, fmt.Errorf("Conflicting arguments. Already had \"%s=%s\", got additional \"%s=%s\"",
			key, stringArgsMap[key], key, val)
	}

	if val == "" {
		return false, fmt.Errorf("Invalid argument. Expected %s={string}, got \"%s=%s\"", key, key, val)
	}

	if key == "boot_index" {
		goodIndex := false
		if val == "none" {
			goodIndex = true
		} else {
			_, err := strconv.Atoi(val)
			if err == nil {
				goodIndex = true
			}
		}
		if !goodIndex {
			return false, fmt.Errorf("Invalid argument. boot_index must be \"none\" or an integer, got \"boot_index=%s\"", val)
		}
	}

	stringArgsMap[key] = val
	return true, nil
}
func processInstanceVolumeSubArg(subArg string, stringArgsMap map[string]string, boolArgsMap map[string]bool, intArgsMap map[string]int) error {
	if !isInstanceVolumeIntArg(subArg) &&
		!isInstanceVolumeStringArg(subArg) &&
		!isInstanceVolumeBoolArg(subArg) {

		return fmt.Errorf("Invalid argument \"%s\"", subArg)
	}

	ok := getInstanceVolumeImplicitBoolArgs(subArg, boolArgsMap)
	if ok {
		return nil
	}

	// split on "=", put in appropriate map
	keyValue := strings.Split(subArg, "=")
	if len(keyValue) != 2 {
		return fmt.Errorf("Invalid argument. Expected key=value, got \"%s\"", keyValue)
	}
	key := keyValue[0]
	val := keyValue[1]

	ok, err := getInstanceVolumeExplicitBoolArgs(key, val, boolArgsMap)
	if err != nil {
		return err
	} else if ok {
		return nil
	}

	ok, err = getInstanceVolumeIntegerArgs(key, val, intArgsMap)
	if err != nil {
		return err
	} else if ok {
		return nil
	}

	ok, err = getInstanceVolumeStringArgs(key, val, stringArgsMap)
	if err != nil {
		return err
	} else if ok {
		return nil
	}

	return nil
}
func validateInstanceVolumeSubArgCombo(vols *volumeFlagSlice) error {
	errPrefix := "Invalid volume argument combination:"

	for _, v := range *vols {
		if v.swap && v.uuid == "" && v.size == 0 {
			return fmt.Errorf("%s swap requires either a uuid or size argument", errPrefix)
		}

		if v.ephemeral && v.uuid == "" && v.size == 0 {
			return fmt.Errorf("%s ephemeral requires either a uuid or size argument", errPrefix)
		}

		if v.local && v.uuid == "" && v.size == 0 {
			return fmt.Errorf("%s local requires either a uuid or size argument", errPrefix)
		}

		if v.uuid != "" && v.size != 0 {
			return fmt.Errorf("%s only one of uuid or size arguments allowed", errPrefix)
		}
		if v.bootIndex != "" && v.uuid == "" {
			return fmt.Errorf("%s boot_index requires a volume uuid", errPrefix)
		}
	}
	return nil
}

// implement the flag.Value interface, eg:
// type Value interface {
// 	String() string
// 	Set(string) error
// }
func (v *volumeFlagSlice) String() string {
	var out string

	for _, vol := range *v {
		var subArgs []string
		if vol.uuid != "" {
			subArgs = append(subArgs, "uuid="+vol.uuid)
		}
		if vol.bootIndex != "" && vol.bootIndex != "none" {
			subArgs = append(subArgs, "boot_index="+vol.bootIndex)
		}
		if vol.swap {
			subArgs = append(subArgs, "swap")
		}
		if vol.local {
			subArgs = append(subArgs, "local")
		}
		if vol.ephemeral {
			subArgs = append(subArgs, "ephemeral")
		}
		if vol.size != 0 {
			subArgs = append(subArgs, "size="+fmt.Sprintf("%d", vol.size))
		}
		if vol.tag != "" {
			subArgs = append(subArgs, "tag="+vol.tag)
		}

		out += "-volume "
		subArgCount := len(subArgs)
		for subArgIdx, subArg := range subArgs {
			out += subArg
			if subArgIdx < subArgCount-1 {
				out += ","
			}
		}

		out += "\n"
	}

	return out
}
func (v *volumeFlagSlice) Set(value string) error {
	if value == "" {
		return fmt.Errorf("Invalid empty volume argument list")
	}

	stringArgsMap := make(map[string]string)
	boolArgsMap := make(map[string]bool)
	intArgsMap := make(map[string]int)

	subArgs := strings.Split(value, ",")
	for _, subArg := range subArgs {
		if subArg == "" {
			continue
		}
		err := processInstanceVolumeSubArg(subArg, stringArgsMap, boolArgsMap, intArgsMap)
		if err != nil {
			return err
		}
	}

	vol := volumeFlag{
		uuid:      stringArgsMap["uuid"],
		bootIndex: stringArgsMap["boot_index"],
		swap:      boolArgsMap["swap"],
		local:     boolArgsMap["local"],
		ephemeral: boolArgsMap["ephemeral"],
		size:      intArgsMap["size"],
		tag:       stringArgsMap["tag"],
	}
	*v = append(*v, vol)

	err := validateInstanceVolumeSubArgCombo(v)
	if err != nil {
		return err
	}

	return nil
}

type instanceAddCommand struct {
	Flag      flag.FlagSet
	workload  string
	instances int
	label     string
	volumes   volumeFlagSlice
	template  string
}

func (cmd *instanceAddCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] instance add [flags]

Launches a new instance

The add flags are:

`)
	cmd.Flag.PrintDefaults()
	printVolumeFlagUsage()
	fmt.Fprintf(os.Stderr, "\n%s", templateutils.GenerateUsageDecorated("f", []compute.ServerDetails{}, nil))
	os.Exit(2)
}

func (cmd *instanceAddCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.workload, "workload", "", "Workload UUID")
	cmd.Flag.IntVar(&cmd.instances, "instances", 1, "Number of instances to create")
	cmd.Flag.StringVar(&cmd.label, "label", "", "Set a frame label. This will trigger frame tracing")
	cmd.Flag.Var(&cmd.volumes, "volume", "volume descriptor argument list")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *instanceAddCommand) validateAddCommandArgs() {
	if *tenantID == "" {
		errorf("Missing required -tenant-id parameter")
		cmd.usage()
	}

	if cmd.workload == "" {
		errorf("Missing required -workload parameter")
		cmd.usage()
	}

	for _, volume := range cmd.volumes {
		//NOTE: volume.uuid itself may only be validated by controller as
		//only it knows which storage interface is in use and what
		//constitutes a valid uuid for that storage implementation

		if cmd.instances != 1 && volume.uuid != "" {
			errorf("Cannot attach volume by uuid (\"-volume uuid=%s\") to multiple instances (\"-instances=%d\")",
				volume.uuid, cmd.instances)
			cmd.usage()
		}
	}
}
func validateCreateServerRequest(server compute.CreateServerRequest) error {
	for _, bd := range server.Server.BlockDeviceMappings {
		if bd.DestinationType == "local" && bd.UUID != "" {
			return fmt.Errorf("Only one of \"uuid={UUID}\" or \"local\" sub-arguments may be specified")
		}

		if bd.VolumeSize != 0 && bd.UUID != "" {
			return fmt.Errorf("Only one of \"uuid={UUID}\" or \"size={SIZE}\" sub-arguments may be specificed")
		}
	}

	return nil
}

func populateCreateServerRequest(cmd *instanceAddCommand, server *compute.CreateServerRequest) {
	server.Server.Name = cmd.label
	server.Server.Flavor = cmd.workload
	server.Server.MaxInstances = cmd.instances
	server.Server.MinInstances = 1

	for _, volume := range cmd.volumes {
		bd := compute.BlockDeviceMappingV2{
			DeviceName:          "", //unsupported
			DeleteOnTermination: volume.ephemeral,
			BootIndex:           volume.bootIndex,
			Tag:                 volume.tag,
			UUID:                volume.uuid,
			VolumeSize:          volume.size,
		}

		if volume.local {
			bd.DestinationType = "local"
		} else {
			bd.DestinationType = "volume"
		}

		if bd.DestinationType == "volume" && volume.uuid != "" {
			// treat all uuid specified items as
			// volumes, ciao internals will figure out
			// if it is an image, volume or snapshot
			bd.SourceType = "volume"
		} else {
			bd.SourceType = "blank"
		}

		if volume.swap {
			bd.GuestFormat = "swap"
		} else {
			bd.GuestFormat = "ephemeral"
		}

		server.Server.BlockDeviceMappings = append(server.Server.BlockDeviceMappings, bd)
	}
}

func (cmd *instanceAddCommand) run(args []string) error {
	cmd.validateAddCommandArgs()

	var server compute.CreateServerRequest
	var servers compute.Servers

	populateCreateServerRequest(cmd, &server)

	err := validateCreateServerRequest(server)
	if err != nil {
		return err
	}

	serverBytes, err := json.Marshal(server)
	if err != nil {
		fatalf(err.Error())
	}
	body := bytes.NewReader(serverBytes)

	url := buildComputeURL("%s/servers", *tenantID)

	resp, err := sendHTTPRequest("POST", url, nil, body)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Instance creation failed: %s", resp.Status)
	}

	err = unmarshalHTTPResponse(resp, &servers)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.template != "" {
		return templateutils.OutputToTemplate(os.Stdout, "instance-add", cmd.template,
			&servers.Servers, nil)
	}

	if len(servers.Servers) < cmd.instances {
		fmt.Println("Some instances failed to start - check the event log for details.")
	}

	for _, server := range servers.Servers {
		fmt.Printf("Created new (pending) instance: %s\n", server.ID)
	}

	return nil
}

type instanceDeleteCommand struct {
	Flag     flag.FlagSet
	instance string
	all      bool
}

func (cmd *instanceDeleteCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] instance delete [flags]

Deletes a given instance

The delete flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *instanceDeleteCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.instance, "instance", "", "Instance UUID")
	cmd.Flag.BoolVar(&cmd.all, "all", false, "Delete all instances for the given tenant")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *instanceDeleteCommand) run(args []string) error {
	if cmd.all {
		return actionAllTenantInstance(*tenantID, osDelete)
	}

	if cmd.instance == "" {
		errorf("Missing required -instance parameter")
		cmd.usage()
	}

	url := buildComputeURL("%s/servers/%s", *tenantID, cmd.instance)

	resp, err := sendHTTPRequest("DELETE", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		fatalf("Instance deletion failed: %s", resp.Status)
	}

	fmt.Printf("Deleted instance: %s\n", cmd.instance)
	return nil
}

type instanceRestartCommand struct {
	Flag     flag.FlagSet
	instance string
}

func (cmd *instanceRestartCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] instance restart [flags]

Restart a stopped Ciao instance

The restart flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *instanceRestartCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.instance, "instance", "", "Instance UUID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *instanceRestartCommand) run([]string) error {
	err := startStopInstance(cmd.instance, false)
	if err != nil {
		cmd.usage()
	}
	return err
}

type instanceStopCommand struct {
	Flag     flag.FlagSet
	instance string
}

func (cmd *instanceStopCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] instance stop [flags]

Stop a Ciao instance

The stop flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *instanceStopCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.instance, "instance", "", "Instance UUID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *instanceStopCommand) run([]string) error {
	err := startStopInstance(cmd.instance, true)
	if err != nil {
		cmd.usage()
	}
	return err
}

func startStopInstance(instance string, stop bool) error {
	if *tenantID == "" {
		return errors.New("Missing required -tenant-id parameter")
	}

	if instance == "" {
		return errors.New("Missing required -instance parameter")
	}

	actionBytes := []byte(osStart)
	if stop == true {
		actionBytes = []byte(osStop)
	}

	body := bytes.NewReader(actionBytes)

	url := buildComputeURL("%s/servers/%s/action", *tenantID, instance)

	resp, err := sendHTTPRequest("POST", url, nil, body)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Instance action failed: %s", resp.Status)
	}

	if stop == true {
		fmt.Printf("Instance %s stopped\n", instance)
	} else {
		fmt.Printf("Instance %s restarted\n", instance)
	}
	return nil
}

type instanceListCommand struct {
	Flag     flag.FlagSet
	workload string
	marker   string
	offset   int
	limit    int
	cn       string
	tenant   string
	detail   bool
	template string
}

func (cmd *instanceListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] instance list [flags]

List instances for a tenant

The list flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s", templateutils.GenerateUsageDecorated("f", []compute.ServerDetails{}, nil))
	os.Exit(2)
}

func (cmd *instanceListCommand) parseArgs(args []string) []string {
	cmd.Flag.BoolVar(&cmd.detail, "detail", false, "Print detailed information about each instance")
	cmd.Flag.StringVar(&cmd.workload, "workload", "", "Workload UUID")
	cmd.Flag.StringVar(&cmd.cn, "cn", "", "Computer node to list instances from (default to all nodes when empty)")
	cmd.Flag.StringVar(&cmd.marker, "marker", "", "Show instance list starting from the next instance after marker")
	cmd.Flag.StringVar(&cmd.tenant, "tenant", "", "Specify to list instances from a tenant other than -tenant-id")
	cmd.Flag.IntVar(&cmd.offset, "offset", 0, "Show instance list starting from instance <offset>")
	cmd.Flag.IntVar(&cmd.limit, "limit", 0, "Limit list to <limit> results")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

type byCreated []compute.ServerDetails

func (ss byCreated) Len() int           { return len(ss) }
func (ss byCreated) Swap(i, j int)      { ss[i], ss[j] = ss[j], ss[i] }
func (ss byCreated) Less(i, j int) bool { return ss[i].Created.Before(ss[j].Created) }

func (cmd *instanceListCommand) run(args []string) error {
	if cmd.tenant == "" {
		cmd.tenant = *tenantID
	}

	if cmd.cn != "" {
		return listNodeInstances(cmd.cn)
	}

	var servers compute.Servers

	url := buildComputeURL("%s/servers/detail", cmd.tenant)

	var values []queryValue
	if cmd.limit > 0 {
		values = append(values, queryValue{
			name:  "limit",
			value: fmt.Sprintf("%d", cmd.limit),
		})
	}

	if cmd.offset > 0 {
		values = append(values, queryValue{
			name:  "offset",
			value: fmt.Sprintf("%d", cmd.offset),
		})
	}

	if cmd.marker != "" {
		values = append(values, queryValue{
			name:  "marker",
			value: cmd.marker,
		})
	}

	if cmd.workload != "" {
		values = append(values, queryValue{
			name:  "flavor",
			value: cmd.workload,
		})
	}

	resp, err := sendHTTPRequest("GET", url, values, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &servers)
	if err != nil {
		fatalf(err.Error())
	}

	sortedServers := []compute.ServerDetails{}
	for _, v := range servers.Servers {
		sortedServers = append(sortedServers, v)
	}
	sort.Sort(byCreated(sortedServers))

	if cmd.template != "" {
		return templateutils.OutputToTemplate(os.Stdout, "instance-list", cmd.template,
			&sortedServers, nil)
	}

	w := new(tabwriter.Writer)
	if !cmd.detail {
		w.Init(os.Stdout, 0, 1, 1, ' ', 0)
		fmt.Fprintln(w, "#\tUUID\tStatus\tPrivate IP\tSSH IP\tSSH PORT")
	}

	for i, server := range sortedServers {
		if !cmd.detail {
			fmt.Fprintf(w, "%d", i+1)
			fmt.Fprintf(w, "\t%s", server.ID)
			fmt.Fprintf(w, "\t%s", server.Status)
			fmt.Fprintf(w, "\t%s", server.Addresses.Private[0].Addr)
			if server.SSHIP != "" {
				fmt.Fprintf(w, "\t%s", server.SSHIP)
				fmt.Fprintf(w, "\t%d\n", server.SSHPort)
			} else {
				fmt.Fprintf(w, "\tN/A")
				fmt.Fprintf(w, "\tN/A\n")
			}
			w.Flush()
		} else {
			fmt.Printf("Instance #%d\n", i+1)
			dumpInstance(&server)
		}
	}
	return nil
}

type instanceShowCommand struct {
	Flag     flag.FlagSet
	instance string
	template string
}

func (cmd *instanceShowCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] instance show [flags]

Print detailed information about an instance

The show flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s", templateutils.GenerateUsageDecorated("f", compute.ServerDetails{}, nil))
	os.Exit(2)
}

func (cmd *instanceShowCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.instance, "instance", "", "Instance UUID")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *instanceShowCommand) run(args []string) error {
	if cmd.instance == "" {
		errorf("Missing required -instance parameter")
		cmd.usage()
	}

	var server compute.Server
	url := buildComputeURL("%s/servers/%s", *tenantID, cmd.instance)

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}
	err = unmarshalHTTPResponse(resp, &server)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.template != "" {
		return templateutils.OutputToTemplate(os.Stdout, "instance-show", cmd.template,
			&server.Server, nil)
	}

	dumpInstance(&server.Server)
	return nil
}

func dumpInstance(server *compute.ServerDetails) {
	fmt.Printf("\tUUID: %s\n", server.ID)
	fmt.Printf("\tStatus: %s\n", server.Status)
	fmt.Printf("\tPrivate IP: %s\n", server.Addresses.Private[0].Addr)
	fmt.Printf("\tMAC Address: %s\n", server.Addresses.Private[0].OSEXTIPSMACMacAddr)
	fmt.Printf("\tCN UUID: %s\n", server.HostID)
	fmt.Printf("\tImage UUID: %s\n", server.Image.ID)
	fmt.Printf("\tTenant UUID: %s\n", server.TenantID)
	if server.SSHIP != "" {
		fmt.Printf("\tSSH IP: %s\n", server.SSHIP)
		fmt.Printf("\tSSH Port: %d\n", server.SSHPort)
	}

	for _, vol := range server.OsExtendedVolumesVolumesAttached {
		fmt.Printf("\tVolume: %s\n", vol)
	}
}

func listNodeInstances(node string) error {
	if node == "" {
		fatalf("Missing required -cn parameter")
	}

	var servers types.CiaoServersStats
	url := buildComputeURL("nodes/%s/servers/detail", node)

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &servers)
	if err != nil {
		fatalf(err.Error())
	}

	for i, server := range servers.Servers {
		fmt.Printf("Instance #%d\n", i+1)
		fmt.Printf("\tUUID: %s\n", server.ID)
		fmt.Printf("\tStatus: %s\n", server.Status)
		fmt.Printf("\tTenant UUID: %s\n", server.TenantID)
		fmt.Printf("\tIPv4: %s\n", server.IPv4)
		fmt.Printf("\tCPUs used: %d\n", server.VCPUUsage)
		fmt.Printf("\tMemory used: %d MB\n", server.MemUsage)
		fmt.Printf("\tDisk used: %d MB\n", server.DiskUsage)
	}

	return nil
}

func actionAllTenantInstance(tenant string, osAction string) error {
	var action types.CiaoServersAction

	url := buildComputeURL("%s/servers/action", tenant)

	action.Action = osAction

	actionBytes, err := json.Marshal(action)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(actionBytes)

	resp, err := sendHTTPRequest("POST", url, nil, body)
	if err != nil {
		fatalf(err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Action %s on all instances failed: %s", osAction, resp.Status)
	}

	fmt.Printf("%s all instances for tenant %s\n", osAction, tenant)
	return nil
}
