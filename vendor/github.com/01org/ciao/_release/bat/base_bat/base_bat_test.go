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

package basebat

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/01org/ciao/bat"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp/uuid"
)

const standardTimeout = time.Second * 300

// Verify that stopping and starting an instance affects a node's instance counts
//
// Retrieve information about all the nodes in the cluster.  Then start a new instance
// and retrieve details on that instance.  If the instance is active check that the
// instance count has changed on the node on which the instance is hosted and the stop
// the instance. Check the instance counts on the host node once more.  The counts should
// match the counts taken at the start of the test.  The tests is a little fragile and
// assumes it is the only test running in the cluster at any one time.  We run it first
// as any pending delete instance operations from preceding tests could cause this
// test to fail.
//
// There should be no errors starting nodes or retrieving instance counts.  Assuming
// the started instance was active the node counts on its host node should have
// increased.  Once stopped the node counts should return to what they were before
// the instance was started.
func TestListComputeNodes(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	beforeStart, err := bat.GetComputeNodes(ctx)
	if err != nil {
		t.Fatalf("Unable to retrieve list of compute nodes")
	}

	instances, err := bat.StartRandomInstances(ctx, "", 1)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}

	defer func() {
		_, err := bat.DeleteInstances(ctx, "", instances)
		if err != nil {
			t.Errorf("Failed to delete instances: %v", err)
		}
	}()

	_, err = bat.WaitForInstancesLaunch(ctx, "", instances, false)
	if err != nil {
		t.Fatalf("Instance %s failed to launch correctly: %v", instances[0], err)
	}

	afterStart, err := bat.GetComputeNodes(ctx)
	if err != nil {
		t.Fatalf("Unable to retrieve list of compute nodes")
	}

	instanceDetails, err := bat.GetInstance(ctx, "", instances[0])
	if err != nil {
		t.Fatalf("Unable to retrieve instance[%s] details: %v",
			instances[0], err)
	}

	hostID := instanceDetails.HostID
	if instanceDetails.Status == "active" {
		if beforeStart[hostID].TotalInstances >= afterStart[hostID].TotalInstances {
			t.Fatalf("Starting an instance should increase instance count on node %v %v", beforeStart[hostID], afterStart[hostID])
		}

		if beforeStart[hostID].TotalRunningInstances >= afterStart[hostID].TotalRunningInstances {
			t.Fatalf("Starting an active instance should increase running instance count on node %v %v", beforeStart[hostID], afterStart[hostID])
		}
		err = bat.StopInstanceAndWait(ctx, "", instances[0])
		if err != nil {
			t.Fatalf("Failed to stop instance %s : %v", instances[0], err)
		}

		afterStart, err = bat.GetComputeNodes(ctx)
		if err != nil {
			t.Fatalf("Unable to retrieve list of compute nodes")
		}
	}

	if beforeStart[hostID].TotalInstances != afterStart[hostID].TotalInstances &&
		beforeStart[hostID].TotalPendingInstances != afterStart[hostID].TotalPendingInstances &&
		beforeStart[hostID].TotalRunningInstances != afterStart[hostID].TotalRunningInstances {
		t.Fatalf("Node instance counts mismatched.  Expected %v found %v",
			beforeStart[hostID], afterStart[hostID])
	}
}

// Get all tenants
//
// TestGetTenants calls ciao-cli tenant list -all.
//
// The test passes if the list of tenants defined for the cluster can
// be retrieved, even if the list is empty.
func TestGetTenants(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	_, err := bat.GetAllTenants(ctx)
	cancelFunc()
	if err != nil {
		t.Fatalf("Failed to retrieve tenant list : %v", err)
	}
}

// Confirm that the cluster is ready
//
// Retrieve the cluster status.
//
// Cluster status is retrieved successfully, the cluster contains more than one
// node and all nodes are ready.
func TestGetClusterStatus(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	status, err := bat.GetClusterStatus(ctx)
	cancelFunc()
	if err != nil {
		t.Fatalf("Failed to retrieve cluster status : %v", err)
	}

	if status.TotalNodes == 0 {
		t.Fatalf("Cluster has no nodes")
	}

	if status.TotalNodes != status.TotalNodesReady {
		t.Fatalf("Some nodes in the cluster are not ready")
	}
}

// Get all available workloads
//
// TestGetWorkloads calls ciao-cli workload list
//
// The test passes if the list of workloads defined for the cluster can
// be retrieved, even if the list is empty.
func TestGetWorkloads(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	_, err := bat.GetAllWorkloads(ctx, "")
	cancelFunc()
	if err != nil {
		t.Fatalf("Failed to retrieve workload list : %v", err)
	}
}

// Start a random workload, then make sure it's listed
//
// Retrieves the list of workloads, selects a random workload,
// creates an instance of that workload and retrieves the instance's
// status.  The instance is then deleted.
//
// The workload should be successfully launched, the status should
// be readable and the instance deleted.
func TestGetInstance(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instances, err := bat.StartRandomInstances(ctx, "", 1)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}

	_, err = bat.RetrieveInstanceStatus(ctx, "", instances[0])
	if err != nil {
		t.Errorf("Failed to retrieve instance status: %v", err)
	}

	scheduled, err := bat.WaitForInstancesLaunch(ctx, "", instances, false)
	if err != nil {
		t.Errorf("Instance %s did not launch: %v", instances[0], err)
	}

	_, err = bat.DeleteInstances(ctx, "", scheduled)
	if err != nil {
		t.Errorf("Failed to delete instance %s: %v", instances[0], err)
	}
}

// Start one instance of all workloads
//
// Retrieve list of available workloads, start one instance of each workload and
// wait for that instance to start.  Delete all started instances.
//
// The workload list should be retrieved correctly, the instances should be
// launched and achieve active status.  All instances should be deleted
// successfully.
func TestStartAllWorkloads(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	workloads, err := bat.GetAllWorkloads(ctx, "")
	if err != nil {
		t.Fatalf("Unable to retrieve workloads %v", err)
	}

	instances := make([]string, 0, len(workloads))
	for _, wkl := range workloads {
		launched, err := bat.LaunchInstances(ctx, "", wkl.ID, 1)
		if err != nil {
			t.Errorf("Unable to launch instance for workload %s : %v",
				wkl.ID, err)
			continue
		}
		scheduled, err := bat.WaitForInstancesLaunch(ctx, "", launched, true)
		if err != nil {
			t.Errorf("Instance %s did not launch correctly : %v", launched[0], err)
		}
		instances = append(instances, scheduled...)
	}

	_, err = bat.DeleteInstances(ctx, "", instances)
	if err != nil {
		t.Errorf("Failed to delete instances: %v", err)
	}
}

func checkInstances(t *testing.T, a, b *bat.Instance) {
	if !(a.TenantID == b.TenantID &&
		a.FlavorID == b.FlavorID && a.ImageID == b.ImageID &&
		a.PrivateIP == b.PrivateIP && a.MacAddress == b.MacAddress &&
		a.SSHIP == b.SSHIP && a.SSHPort == b.SSHPort) {
		t.Fatalf("Instance details do not match: %v %v", a, b)
	}
}

// Test restart and stop
//
// Start a random instance and retrieve the instance's status. We then stop
// the workload, assuming that it is running, restart it and delete it.
//
// The workload should be successfully started, stopped (if it's not running),
// restarted, and finally deleted.
func TestStopRestartInstance(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instances, err := bat.StartRandomInstances(ctx, "", 1)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}

	defer func() {
		err := bat.DeleteInstance(ctx, "", instances[0])
		if err != nil {
			t.Errorf("Failed to delete instance %s: %v", instances[0], err)
		}
	}()

	_, err = bat.WaitForInstancesLaunch(ctx, "", instances, false)
	if err != nil {
		t.Fatalf("Instance %s did not launch: %v", instances[0], err)
	}

	instanceAfterStart, err := bat.GetInstance(ctx, "", instances[0])
	if err != nil {
		t.Fatalf("Failed to retrieve instance %s status: %v", instances[0], err)
	}

	if instanceAfterStart.Status == "active" {
		err = bat.StopInstanceAndWait(ctx, "", instances[0])
		if err != nil {
			t.Fatalf("Failed to stop instance %s : %v", instances[0], err)
		}

		instanceAfterStop, err := bat.GetInstance(ctx, "", instances[0])
		if err != nil {
			t.Fatalf("Failed to retrieve instance %s status: %v", instances[0], err)
		}

		if instanceAfterStop.HostID != "" {
			t.Fatalf("Expected HostID to be \"\".  It was %s",
				instanceAfterStop.HostID)
		}

		checkInstances(t, instanceAfterStart, instanceAfterStop)
	}

	err = bat.RestartInstance(ctx, "", instances[0])
	if err != nil {
		t.Fatalf("Failed to restart instance %s : %v", instances[0], err)
	}

	// If the instance exited after startup assume that this instance is a short
	// lived container.  There's no need to wait for it to restart it may never
	// transition to the restarted state if it starts and quits faster than launcher
	// polls.

	if instanceAfterStart.Status != "active" {
		return
	}

	_, err = bat.WaitForInstancesLaunch(ctx, "", instances, true)
	if err != nil {
		t.Errorf("Timed out waiting for instance %s to restart : %v",
			instances[0], err)
	}

	instanceAfterRestart, err := bat.GetInstance(ctx, "", instances[0])
	if err != nil {
		t.Fatalf("Failed to retrieve instance %s status: %v", instances[0], err)
	}

	checkInstances(t, instanceAfterStart, instanceAfterRestart)

	if instanceAfterRestart.HostID == "" {
		t.Fatal("Expected HostID to be defined")
	}
}

// Test deletion of stopped instances
//
// Start a random instance, try to stop it and then delete it.
//
// The workload should be successfully started and stopped.  We should be able to
// delete the instance.
func TestDeleteStoppedInstance(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instances, err := bat.StartRandomInstances(ctx, "", 1)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}

	defer func() {
		if len(instances) == 0 {
			return
		}
		err := bat.DeleteInstance(ctx, "", instances[0])
		if err != nil {
			t.Errorf("Failed to delete instance %s: %v", instances[0], err)
		}
	}()

	_, err = bat.WaitForInstancesLaunch(ctx, "", instances, false)
	if err != nil {
		t.Fatalf("Instance %s did not launch: %v", instances[0], err)
	}

	instance, err := bat.GetInstance(ctx, "", instances[0])
	if err != nil {
		t.Fatalf("Failed to retrieve instance %s status: %v", instances[0], err)
	}

	if instance.Status == "active" {
		err = bat.StopInstanceAndWait(ctx, "", instances[0])
		if err != nil {
			t.Fatalf("Failed to stop instance %s : %v", instances[0], err)
		}
	}

	i := instances[0]
	instances = nil
	err = bat.DeleteInstanceAndWait(ctx, "", i)
	if err != nil {
		t.Fatalf("Failed to delete instance %s : %v", i, err)
	}
}

// Check that instances based off invalid workloads are deleted automatically.
//
// Create a new workload that launches a non-existent docker image.  Wait for
// the launch to fail and then delete the workload.
//
// The workload should be created correctly and the instance launch should
// succeed.  However, the instance will never actually be launched as the
// underlying container image doesn't really exist.  The image should be
// automatically deleted by controller as verified by calling
// WaitForInstancesLaunch and then the workload should be deleted correctly.
func TestStartBadWorkload(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	opt := bat.WorkloadOptions{
		Description: "BAD Workload test",
		VMType:      payloads.Docker,
		ImageName:   uuid.Generate().String(),
		Defaults: bat.DefaultResources{
			VCPUs: 2,
			MemMB: 128,
		},
	}

	config := `#cloud-config
`
	ID, err := bat.CreateWorkload(ctx, "", opt, config)
	if err != nil {
		t.Fatalf("Failed to create BAD workload: %v", err)
	}

	defer func() {
		err = bat.DeleteWorkload(ctx, "", ID)
		if err != nil {
			t.Errorf("Failed to delete workload")
		}
	}()

	instances, err := bat.LaunchInstances(ctx, "", ID, 1)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}

	_, err = bat.WaitForInstancesLaunch(ctx, "", instances, false)
	if err == nil {
		_, _ = bat.DeleteInstances(ctx, "", instances)
		t.Fatalf("Instance based on invalid workload should not exist")
	}
}

// Start a random workload, then get CNCI information
//
// Start a random workload and verify that there is at least one CNCI present, and that
// a CNCI exists for the tenant of the instance that has just been created.
//
// The instance should be started correctly, at least one CNCI should be returned and
// we should have a CNCI servicing the tenant to which our instance belongs.
func TestGetCNCIs(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instances, err := bat.StartRandomInstances(ctx, "", 1)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}

	defer func() {
		scheduled, err := bat.WaitForInstancesLaunch(ctx, "", instances, false)
		if err != nil {
			t.Errorf("Instance %s did not launch: %v", instances[0], err)
		}

		_, err = bat.DeleteInstances(ctx, "", scheduled)
		if err != nil {
			t.Errorf("Failed to delete instances: %v", err)
		}
	}()

	CNCIs, err := bat.GetCNCIs(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve CNCIs: %v", err)
	}

	if len(CNCIs) == 0 {
		t.Fatalf("No CNCIs found")
	}

	instanceDetails, err := bat.GetInstance(ctx, "", instances[0])
	if err != nil {
		t.Fatalf("Unable to retrieve instance[%s] details: %v",
			instances[0], err)
	}

	foundTenant := false
	for _, v := range CNCIs {
		if v.TenantID == instanceDetails.TenantID {
			foundTenant = true
			break
		}
	}

	if !foundTenant {
		t.Fatalf("Unable to locate a CNCI for instance[%s]", instances[0])
	}
}

// Start 3 random instances and make sure they're all listed
//
// Start 3 random instances, wait for them to be scheduled and retrieve
// their details.  Check some of the instance fields to ensure they're
// valid.  Finally, delete the instances.
//
// Instances should be created and scheduled.  Information about the
// instances should be successfully retrieved and this information should
// contain valid fields.
func TestGetAllInstances(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instances, err := bat.StartRandomInstances(ctx, "", 3)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}

	scheduled, err := bat.WaitForInstancesLaunch(ctx, "", instances, false)
	defer func() {
		_, err := bat.DeleteInstances(ctx, "", scheduled)
		if err != nil {
			t.Errorf("Failed to delete instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("Instance %s did not launch: %v", instances[0], err)
	}

	instanceDetails, err := bat.GetAllInstances(ctx, "")
	if err != nil {
		t.Fatalf("Failed to retrieve instances: %v", err)
	}

	for _, instance := range instances {
		instanceDetail, ok := instanceDetails[instance]
		if !ok {
			t.Fatalf("Failed to retrieve instance %s", instance)
		}

		// Check some basic information

		if instanceDetail.FlavorID == "" || instanceDetail.HostID == "" ||
			instanceDetail.TenantID == "" || instanceDetail.MacAddress == "" ||
			instanceDetail.PrivateIP == "" {
			t.Fatalf("Instance missing information: %+v", instanceDetail)
		}
	}
}

// Start a random workload, then delete it
//
// Start a random instance and wait for the instance to be scheduled.  Delete all
// the instances in the current tenant and then retrieve the list of all instances.
//
// The instance should be started and scheduled, the DeleteAllInstances command should
// succeed and GetAllInstances command should return 0 instances.
func TestDeleteAllInstances(t *testing.T) {
	const retryCount = 15

	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instances, err := bat.StartRandomInstances(ctx, "", 1)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}

	_, err = bat.WaitForInstancesLaunch(ctx, "", instances, false)
	if err != nil {
		t.Errorf("Instance %s did not launch: %v", instances[0], err)
	}

	err = bat.DeleteAllInstances(ctx, "")
	if err != nil {
		t.Fatalf("Failed to delete all instances: %v", err)
	}

	// TODO:  The correct thing to do here is to wait for the Delete Events
	// But these aren't correctly reported yet, see
	// https://github.com/01org/ciao/issues/792

	var i int
	var instancesFound int
	for ; i < retryCount; i++ {
		instanceDetails, err := bat.GetAllInstances(ctx, "")
		if err != nil {
			t.Fatalf("Failed to retrieve instances: %v", err)
		}

		instancesFound = len(instanceDetails)
		if instancesFound == 0 {
			break
		}

		time.Sleep(time.Second)
	}

	if instancesFound != 0 {
		t.Fatalf("0 instances expected.  Found %d", instancesFound)
	}
}

// TestMain ensures that all instances have been deleted when the tests finish.
// The individual tests do try to clean up after themselves but there's always
// the chance that a bug somewhere in ciao could lead to something not getting
// deleted.
func TestMain(m *testing.M) {
	flag.Parse()
	err := m.Run()

	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	_ = bat.DeleteAllInstances(ctx, "")
	cancelFunc()

	os.Exit(err)
}
