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
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

const instanceTemplateDesc = `{ "host_id" : "{{.HostID | js }}", 
    "tenant_id" : "{{.TenantID | js }}", "flavor_id" : "{{.Flavor.ID | js}}",
    "image_id" : "{{.Image.ID | js}}", "status" : "{{.Status | js}}",
    "ssh_ip" : "{{.SSHIP | js }}", "ssh_port" : {{.SSHPort}},
    "volumes" : {{tojson .OsExtendedVolumesVolumesAttached}}
    {{ $addrLen := len .Addresses.Private }}
    {{- if gt $addrLen 0 }}
      {{- with index .Addresses.Private 0 -}}
      , "private_ip" : "{{.Addr | js }}", "mac_address" : "{{.OSEXTIPSMACMacAddr | js -}}"
      {{end -}}
    {{- end }}
  }
`

// Tenant contains basic information about a tenant
type Tenant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Workload contains detailed information about a workload
type Workload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	CPUs int    `json:"vcpus"`
	Mem  int    `json:"ram"`
}

// Instance contains detailed information about an instance
type Instance struct {
	HostID     string   `json:"host_id"`
	TenantID   string   `json:"tenant_id"`
	FlavorID   string   `json:"flavor_id"`
	ImageID    string   `json:"image_id"`
	Status     string   `json:"status"`
	PrivateIP  string   `json:"private_ip"`
	MacAddress string   `json:"mac_address"`
	SSHIP      string   `json:"ssh_ip"`
	SSHPort    int      `json:"ssh_port"`
	Volumes    []string `json:"volumes"`
}

// CNCI contains information about a CNCI
type CNCI struct {
	TenantID  string   `json:"tenant_id"`
	IPv4      string   `json:"ip"`
	Geography string   `json:"geo"`
	Subnets   []string `json:"subnets"`
}

// ClusterStatus contains information about the status of a ciao cluster
type ClusterStatus struct {
	TotalNodes            int `json:"total_nodes"`
	TotalNodesReady       int `json:"total_nodes_ready"`
	TotalNodesFull        int `json:"total_nodes_full"`
	TotalNodesOffline     int `json:"total_nodes_offline"`
	TotalNodesMaintenance int `json:"total_nodes_maintenance"`
}

// NodeStatus contains information about the status of a node
type NodeStatus struct {
	ID                    string    `json:"id"`
	Timestamp             time.Time `json:"updated"`
	Status                string    `json:"status"`
	MemTotal              int       `json:"ram_total"`
	MemAvailable          int       `json:"ram_available"`
	DiskTotal             int       `json:"disk_total"`
	DiskAvailable         int       `json:"disk_available"`
	Load                  int       `json:"load"`
	OnlineCPUs            int       `json:"online_cpus"`
	TotalInstances        int       `json:"total_instances"`
	TotalRunningInstances int       `json:"total_running_instances"`
	TotalPendingInstances int       `json:"total_pending_instances"`
	TotalPausedInstances  int       `json:"total_paused_instances"`
}

func checkEnv(vars []string) error {
	for _, k := range vars {
		if os.Getenv(k) == "" {
			return fmt.Errorf("%s is not defined", k)
		}
	}
	return nil
}

// RunCIAOCLI execs the ciao-cli command with a set of arguments.  The ciao-cli
// process will be killed if the context is Done.  An error will be returned if
// the following environment are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.  On success the data written to ciao-cli on stdout
// will be returned.
func RunCIAOCLI(ctx context.Context, tenant string, args []string) ([]byte, error) {
	vars := []string{"CIAO_IDENTITY", "CIAO_CONTROLLER", "CIAO_USERNAME", "CIAO_PASSWORD"}
	if err := checkEnv(vars); err != nil {
		return nil, err
	}

	if tenant != "" {
		args = append([]string{"-tenant-id", tenant}, args...)
	}

	data, err := exec.CommandContext(ctx, "ciao-cli", args...).Output()
	if err != nil {
		var failureText string
		if err, ok := err.(*exec.ExitError); ok {
			failureText = string(err.Stderr)
		}
		return nil, fmt.Errorf("failed to launch ciao-cli %v : %v\n%s",
			args, err, failureText)
	}

	return data, nil
}

// RunCIAOCLIJS is similar to RunCIAOCLI with the exception that the output
// of the ciao-cli command is expected to be in json format.  The json is
// decoded into the jsdata parameter which should be a pointer to a type
// that corresponds to the json output.
func RunCIAOCLIJS(ctx context.Context, tenant string, args []string, jsdata interface{}) error {
	data, err := RunCIAOCLI(ctx, tenant, args)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, jsdata)
	if err != nil {
		return err
	}

	return nil
}

// RunCIAOCLIAsAdmin execs the ciao-cli command as the admin user with a set of
// provided arguments.  The ciao-cli process will be killed if the context is
// Done.  An error will be returned if the following environment are not set;
// CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.  In
// environments with multiple admin tenants CIAO_ADMIN_TENANT_NAME will be used
// to select the tenant.  On success the data written to ciao-cli on stdout will
// be returned.
func RunCIAOCLIAsAdmin(ctx context.Context, tenant string, args []string) ([]byte, error) {
	vars := []string{"CIAO_IDENTITY", "CIAO_CONTROLLER", "CIAO_ADMIN_USERNAME", "CIAO_ADMIN_PASSWORD"}
	if err := checkEnv(vars); err != nil {
		return nil, err
	}

	if tenant != "" {
		args = append([]string{"-tenant-id", tenant}, args...)
	}

	env := os.Environ()
	envCopy := make([]string, 0, len(env))
	for _, v := range env {
		if !strings.HasPrefix(v, "CIAO_USERNAME=") &&
			!strings.HasPrefix(v, "CIAO_PASSWORD=") &&
			!strings.HasPrefix(v, "CIAO_TENANT_NAME=") {
			envCopy = append(envCopy, v)
		}
	}
	envCopy = append(envCopy, fmt.Sprintf("CIAO_USERNAME=%s",
		os.Getenv("CIAO_ADMIN_USERNAME")))
	envCopy = append(envCopy, fmt.Sprintf("CIAO_PASSWORD=%s",
		os.Getenv("CIAO_ADMIN_PASSWORD")))
	if adminTenantName, ok := os.LookupEnv("CIAO_ADMIN_TENANT_NAME"); ok {
		envCopy = append(envCopy, fmt.Sprintf("CIAO_TENANT_NAME=%s",
			adminTenantName))
	}

	cmd := exec.CommandContext(ctx, "ciao-cli", args...)
	cmd.Env = envCopy
	data, err := cmd.Output()
	if err != nil {
		var failureText string
		if err, ok := err.(*exec.ExitError); ok {
			failureText = string(err.Stderr)
		}
		return nil, fmt.Errorf("failed to launch ciao-cli %v : %v\n%v",
			args, err, failureText)
	}

	return data, nil
}

// RunCIAOCLIAsAdminJS is similar to RunCIAOCLIAsAdmin with the exception that
// the output of the ciao-cli command is expected to be in json format.  The
// json is decoded into the jsdata parameter which should be a pointer to a type
// that corresponds to the json output.
func RunCIAOCLIAsAdminJS(ctx context.Context, tenant string, args []string,
	jsdata interface{}) error {
	data, err := RunCIAOCLIAsAdmin(ctx, tenant, args)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, jsdata)
	if err != nil {
		return err
	}

	return nil
}

// GetAllTenants retrieves a list of all tenants in the cluster by calling
// ciao-cli tenant list -all.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func GetAllTenants(ctx context.Context) ([]*Tenant, error) {
	var tenants []*Tenant

	args := []string{"tenant", "list", "-all", "-f", "{{tojson .}}"}
	err := RunCIAOCLIAsAdminJS(ctx, "", args, &tenants)
	if err != nil {
		return nil, err
	}

	return tenants, nil
}

// GetUserTenants retrieves a list of all the tenants the current user has
// access to. An error will be returned if the following environment variables
// are not set; CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func GetUserTenants(ctx context.Context) ([]*Tenant, error) {
	var tenants []*Tenant

	args := []string{"tenant", "list", "-f", "{{tojson .}}"}
	err := RunCIAOCLIJS(ctx, "", args, &tenants)
	if err != nil {
		return nil, err
	}

	return tenants, nil
}

// GetInstance returns an Instance structure that contains information
// about a specific instance.  The informaion is retrieved by calling
// ciao-cli show --instance.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
func GetInstance(ctx context.Context, tenant string, uuid string) (*Instance, error) {
	var instance *Instance
	args := []string{"instance", "show", "--instance", uuid, "-f", instanceTemplateDesc}
	err := RunCIAOCLIJS(ctx, tenant, args, &instance)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// GetAllInstances returns information about all instances in the specified
// tenant in a map.  The key of the map is the instance uuid.  The information
// is retrieved by calling ciao-cli instance list.  An error will be returned
// if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func GetAllInstances(ctx context.Context, tenant string) (map[string]*Instance, error) {
	var instances map[string]*Instance
	template := `
{
{{- range $i, $val := .}}
  {{- if $i }},{{end}}
  "{{$val.ID | js }}" : {{with $val}}` + instanceTemplateDesc + `{{end}}
{{- end }}
}
`
	args := []string{"instance", "list", "-f", template}
	err := RunCIAOCLIJS(ctx, tenant, args, &instances)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

// RetrieveInstanceStatus retrieve the status of a specific instance.  This
// information is retrieved using ciao-cli instance show.  An error will be
// returned if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func RetrieveInstanceStatus(ctx context.Context, tenant string, instance string) (string, error) {
	args := []string{"instance", "show", "-instance", instance, "-f", "{{.Status}}"}
	data, err := RunCIAOCLI(ctx, tenant, args)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RetrieveInstancesStatuses retrieves the statuses of a slice of specific instances.
// This information is retrieved using ciao-cli instance list.  An error will be
// returned if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func RetrieveInstancesStatuses(ctx context.Context, tenant string) (map[string]string, error) {
	var statuses map[string]string
	template := `
{
{{- range $i, $val := .}}
  {{- if $i }},{{end}} 
   "{{$val.ID | js }}" : "{{$val.Status | js }}"
{{- end }}
}
`
	args := []string{"instance", "list", "-f", template}
	err := RunCIAOCLIJS(ctx, tenant, args, &statuses)
	if err != nil {
		return nil, err
	}
	return statuses, nil
}

// StopInstance stops a ciao instance by invoking the ciao-cli instance stop command.
// An error will be returned if the following environment variables are not set;
// CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func StopInstance(ctx context.Context, tenant string, instance string) error {
	args := []string{"instance", "stop", "-instance", instance}
	_, err := RunCIAOCLI(ctx, tenant, args)
	return err
}

// StopInstanceAndWait stops a ciao instance by invoking the ciao-cli instance stop command.
// It then waits until the instance's status changes to exited.
// An error will be returned if the following environment variables are not set;
// CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func StopInstanceAndWait(ctx context.Context, tenant string, instance string) error {
	if err := StopInstance(ctx, tenant, instance); err != nil {
		return err
	}
	for {
		status, err := RetrieveInstanceStatus(ctx, tenant, instance)
		if err != nil {
			return err
		}

		if status == "exited" {
			return nil
		}

		select {
		case <-time.After(time.Second):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// RestartInstance restarts a ciao instance by invoking the ciao-cli instance restart
// command.  An error will be returned if the following environment variables are not set;
// CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func RestartInstance(ctx context.Context, tenant string, instance string) error {
	args := []string{"instance", "restart", "-instance", instance}
	_, err := RunCIAOCLI(ctx, tenant, args)
	return err
}

// RestartInstanceAndWait restarts a ciao instance by invoking the ciao-cli instance
// restart command.   It then waits until the instance's status changes to active.
// An error will be returned if the following environment variables are not set;
// CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func RestartInstanceAndWait(ctx context.Context, tenant string, instance string) error {
	if err := RestartInstance(ctx, tenant, instance); err != nil {
		return err
	}
	for {
		status, err := RetrieveInstanceStatus(ctx, tenant, instance)
		if err != nil {
			return err
		}

		if status == "active" {
			return nil
		}

		select {
		case <-time.After(time.Second):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// DeleteInstance deletes a specific instance from the cluster.  It deletes
// the instance using ciao-cli instance delete.  An error will be returned
// if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func DeleteInstance(ctx context.Context, tenant string, instance string) error {
	args := []string{"instance", "delete", "-instance", instance}
	_, err := RunCIAOCLI(ctx, tenant, args)
	return err
}

// DeleteInstanceAndWait deletes a specific instance from the cluster.  It deletes
// the instance using ciao-cli instance delete and then blocks until ciao-cli
// reports that the instance is truly deleted.  An error will be returned
// if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func DeleteInstanceAndWait(ctx context.Context, tenant string, instance string) error {
	if err := DeleteInstance(ctx, tenant, instance); err != nil {
		return err
	}

	// TODO:  The correct thing to do here is to wait for the Delete Events
	// But these do not yet contain enough information to easily identify
	// the event we're interested in.

	for {
		_, err := RetrieveInstanceStatus(ctx, tenant, instance)
		if err == nil {
			select {
			case <-time.After(time.Second):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err == context.Canceled {
			return err
		}

		return nil
	}
}

// DeleteInstances deletes a set of instances provided by the instances slice.
// If the function encounters an error deleting an instance it records the error
// and proceeds to the delete the next instance. The function returns two values,
// an error and a slice of errors.  A single error value is set if any of the
// instance deletion attempts failed. A slice of errors is also returned so that
// the caller can determine which of the deletion attempts failed. The indices
// in the error slice match the indicies in the instances slice, i.e., a non nil
// value in the first element of the error slice indicates that there was an
// error deleting the first instance in the instances slice.  An error will be
// returned if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func DeleteInstances(ctx context.Context, tenant string, instances []string) ([]error, error) {
	var err error
	errs := make([]error, len(instances))

	for i, instance := range instances {
		errs[i] = DeleteInstance(ctx, tenant, instance)
		if err == nil && errs[i] != nil {
			err = fmt.Errorf("At least one instance deletion attempt failed")
		}
	}

	return errs, err
}

// DeleteAllInstances deletes all the instances created for the specified
// tenant by calling ciao-cli instance delete -all.  It returns an error
// if the ciao-cli command fails.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
func DeleteAllInstances(ctx context.Context, tenant string) error {
	args := []string{"instance", "delete", "-all"}
	_, err := RunCIAOCLI(ctx, tenant, args)
	return err
}

func checkStatuses(instances []string, statuses map[string]string,
	mustBeActive bool) ([]string, bool, error) {

	var err error
	scheduled := make([]string, 0, len(instances))
	finished := true
	for _, instance := range instances {
		status, ok := statuses[instance]
		if !ok {
			if err == nil {
				err = fmt.Errorf("Instance %s does not exist", instance)
			}
			continue
		}

		scheduled = append(scheduled, instance)

		if status == "pending" {
			finished = false
		} else if err == nil && mustBeActive && status == "exited" {
			err = fmt.Errorf("Instance %s has exited", instance)
		}
	}

	return scheduled, finished, err
}

// WaitForInstancesLaunch waits for a slice of newly created instances to be
// scheduled.  An instance is scheduled when its status changes from pending
// to exited or active.  If mustBeActive is set to true, the function will
// fail if it sees an instance that has been scheduled but whose status is
// exited.  The function returns a slice of instance UUIDs and an error.
// In the case of success, the returned slice of UUIDs will equal the instances
// array.  In the case of error, these two slices may be different.  This
// can happen if one or more of the instances has failed to launch.  If errors
// are detected with multiple instances, e.g., mustBeActive is true and two
// instances have a status of 'exited' the error returned will refers to the
// first instance only.    An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
func WaitForInstancesLaunch(ctx context.Context, tenant string, instances []string,
	mustBeActive bool) ([]string, error) {

	scheduled := make([]string, 0, len(instances))
	for {
		statuses, err := RetrieveInstancesStatuses(ctx, tenant)
		if err != nil {
			return scheduled, err
		}

		var finished bool
		scheduled, finished, err = checkStatuses(instances, statuses, mustBeActive)
		if finished || err != nil {
			return scheduled, err
		}

		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return scheduled, ctx.Err()
		}
	}
}

// LaunchInstances launches num instances of the specified workload.  On success
// the function returns a slice of UUIDs of the successfully launched instances.
// If some instances failed to start then the error can be found in the event
// log.  The instances are launched using ciao-cli instance add. If no instances
// successfully launch then an error will be returned.  An error will be
// returned if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func LaunchInstances(ctx context.Context, tenant string, workload string, num int) ([]string, error) {
	template := `
[
{{- range $i, $val := .}}
  {{- if $i }},{{end}}"{{$val.ID | js }}"
{{- end }}
]
`
	args := []string{"instance", "add", "--workload", workload,
		"--instances", fmt.Sprintf("%d", num), "-f", template}
	var instances []string
	err := RunCIAOCLIJS(ctx, tenant, args, &instances)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

// StartRandomInstances starts a specified number of instances using
// a random workload.  The UUIDs of the started instances are returned
// to the user.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
func StartRandomInstances(ctx context.Context, tenant string, num int) ([]string, error) {
	wklds, err := GetAllWorkloads(ctx, tenant)
	if err != nil {
		return nil, err
	}

	if len(wklds) == 0 {
		return nil, fmt.Errorf("No workloads defined")
	}

	wkldUUID := wklds[rand.Intn(len(wklds))].ID
	return LaunchInstances(ctx, tenant, wkldUUID, num)
}

// GetCNCIs returns a map of the CNCIs present in the cluster.  The key
// of the map is the CNCI ID.  The CNCI information is retrieved using
// ciao-cli list -cnci command.  An error will be returned if the
// following environment are not set;  CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func GetCNCIs(ctx context.Context) (map[string]*CNCI, error) {
	var CNCIs map[string]*CNCI
	template := `
{
{{- range $i, $val := .}}
  {{- if $i }},{{end}} 
   "{{$val.ID | js }}" : {
    "tenant_id" : "{{$val.TenantID | js }}", "ip" : "{{$val.IPv4 | js}}",
    "geo": "{{$val.Geography | js }}", "subnets": [
        {{- range $j, $net := $val.Subnets -}}
              {{- if $j }},{{end -}}
              "{{- $net.Subnet -}}"
        {{- end -}}
    ]}
  {{- end }}
}
`
	args := []string{"node", "list", "-cnci", "-f", template}
	err := RunCIAOCLIAsAdminJS(ctx, "", args, &CNCIs)
	if err != nil {
		return nil, err
	}

	return CNCIs, nil
}

func getNodes(ctx context.Context, args []string) (map[string]*NodeStatus, error) {
	var nodeList []*NodeStatus
	err := RunCIAOCLIAsAdminJS(ctx, "", args, &nodeList)
	if err != nil {
		return nil, err
	}

	nodeMap := make(map[string]*NodeStatus)
	for _, n := range nodeList {
		nodeMap[n.ID] = n
	}

	return nodeMap, nil
}

// GetComputeNodes returns a map containing status information about
// each compute node in the cluster.  The key of the map is the Node ID.  The
// information is retrieved using ciao-cli list -compute command.  An
// error will be returned if the following environment are not set;
// CIAO_IDENTITY,  CIAO_CONTROLLER, CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func GetComputeNodes(ctx context.Context) (map[string]*NodeStatus, error) {
	args := []string{"node", "list", "-compute", "-f", "{{tojson .}}"}
	return getNodes(ctx, args)
}

// GetNetworkNodes returns a map containing status information about
// each network node in the cluster.  The key of the map is the Node ID.  The
// information is retrieved using ciao-cli list -network command.  An
// error will be returned if the following environment are not set;
// CIAO_IDENTITY,  CIAO_CONTROLLER, CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func GetNetworkNodes(ctx context.Context) (map[string]*NodeStatus, error) {
	args := []string{"node", "list", "-network", "-f", "{{tojson .}}"}
	return getNodes(ctx, args)
}

// GetClusterStatus returns the status of the ciao cluster.  The information
// is retrieved by calling ciao-cli node status.  An error will be returned
// if the following environment are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func GetClusterStatus(ctx context.Context) (*ClusterStatus, error) {
	var cs *ClusterStatus
	args := []string{"node", "status", "-f", "{{tojson .}}"}
	err := RunCIAOCLIAsAdminJS(ctx, "", args, &cs)
	if err != nil {
		return nil, err
	}

	return cs, nil
}
