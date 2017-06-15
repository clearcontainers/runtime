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

package main

import (
	"github.com/01org/ciao/ciao-controller/internal/datastore"
	"github.com/01org/ciao/ciao-controller/internal/quotas"
	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/pkg/errors"
)

func (c *controller) UpdateQuotas(tenantID string, qds []types.QuotaDetails) error {
	err := c.ds.UpdateQuotas(tenantID, qds)
	if err != nil {
		return errors.Wrap(err, "error updating quotas in database")
	}
	c.qs.Update(tenantID, qds)
	return nil
}

func (c *controller) ListQuotas(tenantID string) []types.QuotaDetails {
	return c.qs.DumpQuotas(tenantID)
}

func populateQuotasFromDatastore(qs *quotas.Quotas, ds *datastore.Datastore) error {
	ts, err := ds.GetAllTenants()
	if err != nil {
		return errors.Wrap(err, "error getting tenants")
	}

	for _, t := range ts {
		// Populate quotas/limits from datastore
		qds, err := ds.GetQuotas(t.ID)
		if err != nil {
			return errors.Wrapf(err, "error getting quotas for tenant %s", t.ID)
		}
		qs.Update(t.ID, qds)

		// Populate volume usage
		// TODO: populate image usage
		// TODO: populate external IP usage
		bds, err := ds.GetBlockDevices(t.ID)
		if err != nil {
			return errors.Wrapf(err, "error getting block devices for tenant %s", t.ID)
		}
		var size, count int
		for _, bd := range bds {
			size += bd.Size
			count++
		}
		// With initial population we disregard the result of consumption
		<-qs.Consume(t.ID,
			payloads.RequestedResource{Type: payloads.Volume, Value: count},
			payloads.RequestedResource{Type: payloads.SharedDiskGiB, Value: size})

		instances, err := ds.GetAllInstancesFromTenant(t.ID)
		if err != nil {
			return errors.Wrapf(err, "error getting tenant instances")
		}

		for _, instance := range instances {
			wl, err := ds.GetWorkload(t.ID, instance.WorkloadID)
			if err != nil {
				return errors.Wrapf(err, "error getting workload")
			}
			resources := []payloads.RequestedResource{{Type: payloads.Instance, Value: 1}}
			resources = append(resources, wl.Defaults...)
			<-qs.Consume(t.ID, resources...)
		}
	}

	return nil
}
