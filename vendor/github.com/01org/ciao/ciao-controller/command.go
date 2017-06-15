/*
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
*/

package main

import (
	"fmt"
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/pkg/errors"
)

func (c *controller) evacuateNode(nodeID string) error {
	// should I bother to see if nodeID is valid?
	go c.client.EvacuateNode(nodeID)
	return nil
}

func (c *controller) restartInstance(instanceID string) error {
	// should I bother to see if instanceID is valid?
	i, err := c.ds.GetInstance(instanceID)
	if err != nil {
		return err
	}

	if i.State != "exited" {
		return errors.New("You may only restart paused instances")
	}

	w, err := c.ds.GetWorkload(i.TenantID, i.WorkloadID)
	if err != nil {
		return err
	}

	t, err := c.ds.GetTenant(i.TenantID)
	if err != nil {
		return err
	}

	go c.client.RestartInstance(i, &w, t)
	return nil
}

func (c *controller) stopInstance(instanceID string) error {
	// get node id.  If there is no node id we can't send a delete
	i, err := c.ds.GetInstance(instanceID)
	if err != nil {
		return err
	}

	if i.NodeID == "" {
		return types.ErrInstanceNotAssigned
	}

	if i.State == payloads.ComputeStatusPending {
		return errors.New("You may not stop a pending instance")
	}

	go c.client.StopInstance(instanceID, i.NodeID)
	return nil
}

func (c *controller) deleteInstance(instanceID string) error {
	// get node id.  If there is no node id and the instance is
	// pending we can't send a delete
	i, err := c.ds.GetInstance(instanceID)
	if err != nil {
		return err
	}

	if i.NodeID == "" && i.State == payloads.Pending {
		return types.ErrInstanceNotAssigned
	}

	// check for any external IPs
	IPs := c.ds.GetMappedIPs(&i.TenantID)
	for _, m := range IPs {
		if m.InstanceID == instanceID {
			return types.ErrInstanceMapped
		}
	}

	go c.client.DeleteInstance(instanceID, i.NodeID)
	return nil
}

func (c *controller) confirmTenantRaw(tenantID string) error {
	tenant, err := c.ds.GetTenant(tenantID)
	if err != nil {
		return err
	}

	if tenant == nil {
		tenant, err = c.ds.AddTenant(tenantID)
		if err != nil {
			return err
		}
	}

	if tenant.CNCIIP == "" && !*noNetwork {
		err := c.launchCNCI(tenantID)
		if err != nil {
			return err
		}

		tenant, err = c.ds.GetTenant(tenantID)
		if err != nil {
			return err
		}

		if tenant.CNCIIP == "" {
			return errors.New("Unable to Launch Tenant CNCI")
		}
	}

	return nil
}

func (c *controller) confirmTenant(tenantID string) error {
	c.tenantReadinessLock.Lock()
	memo := c.tenantReadiness[tenantID]
	if memo != nil {

		// Someone else has already or is in the process of confirming
		// this tenant.  We need to wait until memo.ch is closed before
		// continuing.

		c.tenantReadinessLock.Unlock()
		<-memo.ch
		if memo.err != nil {
			return memo.err
		}

		// If we get here we know that confirmTenantRaw has already
		// been successfully called for this tenant during the life
		// time of this controller invocation.

		return nil
	}

	ch := make(chan struct{})
	c.tenantReadiness[tenantID] = &tenantConfirmMemo{ch: ch}
	c.tenantReadinessLock.Unlock()
	err := c.confirmTenantRaw(tenantID)
	if err != nil {
		c.tenantReadinessLock.Lock()
		c.tenantReadiness[tenantID].err = err
		delete(c.tenantReadiness, tenantID)
		c.tenantReadinessLock.Unlock()
	}
	close(ch)
	return err
}

func (c *controller) startWorkload(w types.WorkloadRequest) ([]*types.Instance, error) {
	var e error

	if w.Instances <= 0 {
		return nil, errors.New("Missing number of instances to start")
	}

	wl, err := c.ds.GetWorkload(w.TenantID, w.WorkloadID)
	if err != nil {
		return nil, err
	}

	if !isCNCIWorkload(&wl) {
		err := c.confirmTenant(w.TenantID)
		if err != nil {
			return nil, err
		}
	}

	var newInstances []*types.Instance

	for i := 0; i < w.Instances && e == nil; i++ {
		startTime := time.Now()

		name := w.Name
		if name != "" {
			if w.Instances > 1 {
				name = fmt.Sprintf("%s-%d", name, i)
			}
		}

		instance, err := newInstance(c, w.TenantID, &wl, w.Volumes, name)
		if err != nil {
			e = errors.Wrap(err, "Error creating instance")
			continue
		}
		instance.startTime = startTime

		ok, err := instance.Allowed()
		if err != nil {
			instance.Clean()
			e = errors.Wrap(err, "Error checking if instance allowed")
			continue
		}

		if ok {
			err = instance.Add()
			if err != nil {
				instance.Clean()
				e = errors.Wrap(err, "Error adding instance")
				continue
			}

			newInstances = append(newInstances, &instance.Instance)
			if w.TraceLabel == "" {
				go c.client.StartWorkload(instance.newConfig.config)
			} else {
				go c.client.StartTracedWorkload(instance.newConfig.config, instance.startTime, w.TraceLabel)
			}
		} else {
			instance.Clean()
			// stop if we are over limits
			e = errors.New("Over quota")
			continue
		}
	}

	return newInstances, e
}

func (c *controller) launchCNCI(tenantID string) error {
	workloadID, err := c.ds.GetCNCIWorkloadID()
	if err != nil {
		return err
	}

	ch := make(chan bool)

	c.ds.AddTenantChan(ch, tenantID)

	w := types.WorkloadRequest{
		WorkloadID: workloadID,
		TenantID:   tenantID,
		Instances:  1,
	}
	_, err = c.startWorkload(w)
	if err != nil {
		return err
	}

	success := <-ch

	if success {
		return nil
	}
	msg := fmt.Sprintf("Failed to Launch CNCI for %s", tenantID)
	return errors.New(msg)
}

func (c *controller) deleteEphemeralStorage(instanceID string) error {
	attachments := c.ds.GetStorageAttachments(instanceID)
	for _, attachment := range attachments {
		if !attachment.Ephemeral {
			continue
		}
		err := c.ds.DeleteStorageAttachment(attachment.ID)
		if err != nil {
			return errors.Wrap(err, "Error deleting storage attachment from datastore")
		}
		bd, err := c.ds.GetBlockDevice(attachment.BlockID)
		if err != nil {
			return errors.Wrap(err, "Error getting block device from datastore")
		}
		err = c.ds.DeleteBlockDevice(attachment.BlockID)
		if err != nil {
			return errors.Wrap(err, "Error deleting block device from datastore")
		}
		err = c.DeleteBlockDevice(attachment.BlockID)
		if err != nil {
			return errors.Wrap(err, "Error deleting block device")
		}
		c.qs.Release(bd.TenantID,
			payloads.RequestedResource{Type: payloads.Volume, Value: 1},
			payloads.RequestedResource{Type: payloads.SharedDiskGiB, Value: bd.Size})
	}
	return nil
}
