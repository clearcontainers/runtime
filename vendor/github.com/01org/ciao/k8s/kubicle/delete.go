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

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/01org/ciao/bat"
)

type destroyer struct {
	instances   map[string]*bat.Instance
	wDeleted    []string
	iDeleted    []string
	ipsDeleted  []string
	poolName    string
	poolDeleted string
	ipsMapped   bool
}

func (d *destroyer) deleteExternalIPs(ctx context.Context) error {
	externalIPs, err := bat.ListExternalIPs(ctx, "")
	if err != nil {
		return ctx.Err()
	}

	for _, ip := range externalIPs {
		if ip.PoolName != d.poolName {
			continue
		}
		if _, ok := d.instances[ip.InstanceID]; ok {
			d.ipsMapped = true
			err := bat.UnmapExternalIP(ctx, "", ip.ExternalIP)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to unmap %s\n", ip.ExternalIP)
				if ctx.Err() != nil {
					return ctx.Err()
				}
				continue
			}
			d.ipsDeleted = append(d.ipsDeleted, ip.ExternalIP)
		}
	}

	for i := 0; i < 5; i++ {
		err := bat.DeleteExternalIPPool(ctx, "", d.poolName)
		if err == nil {
			d.poolDeleted = d.poolName
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}

	return nil
}

func (d *destroyer) deleteWorkloadAndInstances(ctx context.Context, workload string) error {
	for instanceID, instance := range d.instances {
		if workload == instance.FlavorID {
			err := deleteInstance(ctx, instanceID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to delete instance %s\n", instanceID)
				if ctx.Err() != nil {
					return ctx.Err()
				}
			} else {
				d.iDeleted = append(d.iDeleted, instanceID)
			}
		}
	}

	err := bat.DeleteWorkload(ctx, "", workload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Unable to delete workload %s: %v\n", workload, err)
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	d.wDeleted = append(d.wDeleted, workload)

	return nil
}

func (d *destroyer) status() {
	if len(d.ipsDeleted) > 0 {
		fmt.Println("External-ips deleted:")
		for _, ips := range d.ipsDeleted {
			fmt.Println(ips)
		}
		fmt.Println("")
	}

	if d.poolDeleted != "" {
		fmt.Println("Pools Deleted:")
		fmt.Println(d.poolDeleted)
		fmt.Println("")
	}

	fmt.Println("Workloads deleted:")
	for _, w := range d.wDeleted {
		fmt.Println(w)
	}

	fmt.Println("\nInstances deleted:")
	for _, i := range d.iDeleted {
		fmt.Println(i)
	}
}

func (d *destroyer) destroyCluster(ctx context.Context) error {
	var err error
	d.instances, err = bat.GetAllInstances(ctx, "")
	if err != nil {
		return fmt.Errorf("Failed to retrieve instance list")
	}

	defer d.status()

	masterWkldUUID, err := getWorkloadUUID(ctx, masterWorkloadName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to determine master workload: %v\n", err)
		if ctx.Err() != nil {
			return err
		}
	} else {
		d.poolName = fmt.Sprintf("%s-%s", externalIPPool, masterWkldUUID)
		err = d.deleteExternalIPs(ctx)
		if err != nil {
			return err
		}
	}

	workerWkldUUID, err := getWorkloadUUID(ctx, workerWorkloadName)
	if err == nil {
		err := d.deleteWorkloadAndInstances(ctx, workerWkldUUID)
		if err != nil {
			return err
		}
	} else {
		fmt.Fprintf(os.Stderr, "Warning: Failed to determine worker workload: %v\n", err)
		if ctx.Err() != nil {
			return err
		}
	}

	if masterWkldUUID != "" {
		err := d.deleteWorkloadAndInstances(ctx, masterWkldUUID)
		if err != nil {
			return err
		}
	}

	if len(d.wDeleted) == 0 || len(d.iDeleted) == 0 ||
		(d.ipsMapped && (len(d.ipsDeleted) == 0 || d.poolDeleted == "")) {
		return fmt.Errorf("Not all parts of the cluster were deleted")
	}

	return nil
}

func destroy(ctx context.Context, errCh chan error) {
	d := &destroyer{}
	errCh <- d.destroyCluster(ctx)
}
