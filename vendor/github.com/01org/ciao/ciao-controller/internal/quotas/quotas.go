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

package quotas

import (
	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
)

type quota struct {
	limit    int
	consumed int
}

type tenantData struct {
	quotas map[payloads.Resource]*quota

	perInstanceVCPUs  int
	perInstanceMemory int
	perVolumeSize     int
}

// Quotas provides a quota and limit service
type Quotas struct {
	ch chan interface{}
}

// Result provides a method for querying the result of a Consume operation.
type Result interface {
	Allowed() bool
	Reason() string
	Resources() []payloads.RequestedResource
}

type consumeOp struct {
	tenantID  string
	resources []payloads.RequestedResource
	ch        chan Result
}

type releaseOp struct {
	tenantID  string
	resources []payloads.RequestedResource
}

type updateOp struct {
	tenantID string
	quotas   []types.QuotaDetails
	doneCh   chan struct{}
}

type dumpOp struct {
	tenantID string
	ch       chan []types.QuotaDetails
}

type result struct {
	allowed   bool
	reason    string
	resources []payloads.RequestedResource
}

var supportedResources = [...]payloads.Resource{
	payloads.VCPUs,
	payloads.MemMB,
	payloads.Volume,
	payloads.SharedDiskGiB,
	payloads.Instance,
	payloads.Image,
	payloads.ExternalIP,
}

func makeTentantData() *tenantData {
	td := tenantData{}
	td.quotas = make(map[payloads.Resource]*quota)

	for _, resource := range supportedResources {
		td.quotas[resource] = &quota{-1, 0}
	}

	td.perInstanceMemory = -1
	td.perInstanceVCPUs = -1
	td.perVolumeSize = -1

	return &td
}

func getTenantData(tenantDetails map[string]*tenantData, tenantID string) *tenantData {
	td, ok := tenantDetails[tenantID]
	if !ok {
		td = makeTentantData()
		tenantDetails[tenantID] = td
	}

	return td
}

func consumeQuota(tenantDetails map[string]*tenantData, op *consumeOp) Result {
	td := getTenantData(tenantDetails, op.tenantID)
	allowed := true

	for _, r := range op.resources {
		q, ok := td.quotas[r.Type]

		if ok {
			q.consumed += r.Value
			if q.limit > -1 && q.consumed > q.limit {
				allowed = false
			}
		}
	}

	res := &result{resources: op.resources}
	res.allowed = allowed
	if !allowed {
		// TODO: produce more precise reason
		res.reason = "Over quota"
	}
	return res
}

func checkLimit(tenantDetails map[string]*tenantData, op *consumeOp) Result {
	td := getTenantData(tenantDetails, op.tenantID)

	allowed := true
	for _, r := range op.resources {
		switch r.Type {
		case payloads.VCPUs:
			if td.perInstanceVCPUs > -1 && r.Value > td.perInstanceVCPUs {
				allowed = false
			}
		case payloads.MemMB:
			if td.perInstanceMemory > -1 && r.Value > td.perInstanceMemory {
				allowed = false
			}
		case payloads.SharedDiskGiB:
			if td.perVolumeSize > -1 && r.Value > td.perVolumeSize {
				allowed = false
			}
		}
	}
	res := &result{resources: op.resources}
	res.allowed = allowed
	if !allowed {
		// TODO: produce more precise reason
		res.reason = "Over limit"
	}
	return res
}

func release(tenantDetails map[string]*tenantData, op *releaseOp) {
	td := getTenantData(tenantDetails, op.tenantID)

	for _, r := range op.resources {
		q, ok := td.quotas[r.Type]

		if ok {
			q.consumed -= r.Value
			if q.consumed < 0 {
				q.consumed = 0
			}
		}
	}
}

func quotaNameToResource(name string) payloads.Resource {
	switch name {
	case "tenant-vcpu-quota":
		return payloads.VCPUs
	case "tenant-mem-quota":
		return payloads.MemMB
	case "tenant-storage-quota":
		return payloads.SharedDiskGiB
	case "tenant-volumes-quota":
		return payloads.Volume
	case "tenant-instances-quota":
		return payloads.Instance
	case "tenant-images-quota":
		return payloads.Image
	case "tenant-external-ips-quota":
		return payloads.ExternalIP
	}

	return ""
}

func resourceToQuotaName(r payloads.Resource) string {
	switch r {
	case payloads.VCPUs:
		return "tenant-vcpu-quota"
	case payloads.MemMB:
		return "tenant-mem-quota"
	case payloads.Volume:
		return "tenant-volumes-quota"
	case payloads.SharedDiskGiB:
		return "tenant-storage-quota"
	case payloads.Instance:
		return "tenant-instances-quota"
	case payloads.Image:
		return "tenant-images-quota"
	case payloads.ExternalIP:
		return "tenant-external-ips-quota"
	}
	return ""
}

func update(tenantDetails map[string]*tenantData, op *updateOp) {
	td := getTenantData(tenantDetails, op.tenantID)

	for _, q := range op.quotas {
		r := quotaNameToResource(q.Name)

		if r != "" {
			td.quotas[r].limit = q.Value
		}

		switch q.Name {
		case "tenant-vcpu-per-instance-limit":
			td.perInstanceVCPUs = q.Value
		case "tenant-mem-per-instance-limit":
			td.perInstanceMemory = q.Value
		case "tenant-volume-size-limit":
			td.perVolumeSize = q.Value
		}
	}
}

func dump(tenantDetails map[string]*tenantData, op *dumpOp) []types.QuotaDetails {
	td := getTenantData(tenantDetails, op.tenantID)

	qds := []types.QuotaDetails{}

	for r, q := range td.quotas {
		name := resourceToQuotaName(r)
		if name != "" {
			qd := types.QuotaDetails{
				Name:  name,
				Value: q.limit,
				Usage: q.consumed,
			}
			qds = append(qds, qd)
		}
	}

	qd := types.QuotaDetails{
		Name:  "tenant-vcpu-per-instance-limit",
		Value: td.perInstanceVCPUs,
	}
	qds = append(qds, qd)
	qd = types.QuotaDetails{
		Name:  "tenant-mem-per-instance-limit",
		Value: td.perInstanceMemory,
	}
	qds = append(qds, qd)
	qd = types.QuotaDetails{
		Name:  "tenant-volume-size-limit",
		Value: td.perVolumeSize,
	}
	qds = append(qds, qd)

	return qds
}

// Init is used to initialise the quota service.
func (qs *Quotas) Init() {
	qs.ch = make(chan interface{})

	go func() {
		tenantDetails := make(map[string]*tenantData)

		for {
			data, more := <-qs.ch
			if !more {
				return
			}

			switch data.(type) {

			case *consumeOp:
				consumeData := data.(*consumeOp)
				res := consumeQuota(tenantDetails, consumeData)
				if !res.Allowed() {
					consumeData.ch <- res
					close(consumeData.ch)
					continue
				}
				res = checkLimit(tenantDetails, consumeData)
				consumeData.ch <- res
				close(consumeData.ch)

			case *releaseOp:
				releaseData := data.(*releaseOp)
				release(tenantDetails, releaseData)

			case *updateOp:
				updateData := data.(*updateOp)
				update(tenantDetails, updateData)
				close(updateData.doneCh)

			case *dumpOp:
				dumpData := data.(*dumpOp)
				dumpData.ch <- dump(tenantDetails, dumpData)
				close(dumpData.ch)
			}
		}

	}()
}

func copyResources(resources []payloads.RequestedResource) []payloads.RequestedResource {
	copy := make([]payloads.RequestedResource, len(resources))

	for i := range resources {
		copy[i] = resources[i]
	}

	return copy
}

// Consume will update the quota records to indicate that the tenant is using
// all the resources specified. This method should usually be used on a
// per-instance/volume/image basis as it will also check against the limits.
// The exception to this is for initial import when disregarding the result.
//
// This method returns a Result channel indicating whether the consumption is
// allowed. The result of the Consume() is indicated by
// Result.Allowed(). The caller can choose to ignore this or reclaim the
// resources used (e.g. by terminating an instance.) If the caller chooses to
// reclaim the resource they must call Quotas.Release(). The resources used in
// the original request are available in the result by calling
// Result.Resources(). If Result.Allowed() returns false then then
// Result.Reason() returns an explanation that can be shared with the user.
func (qs *Quotas) Consume(tenantID string, resources ...payloads.RequestedResource) chan Result {
	ch := make(chan Result, 1)
	data := &consumeOp{tenantID, copyResources(resources), ch}
	qs.ch <- data

	return ch
}

// Release will update the quota records for a tenant to indicate that it is no
// longer using the supplied resources.
func (qs *Quotas) Release(tenantID string, resources ...payloads.RequestedResource) {
	data := &releaseOp{tenantID, copyResources(resources)}
	qs.ch <- data
}

// Shutdown will stop the quota service and should be called when it is no
// longer needed.
func (qs *Quotas) Shutdown() {
	close(qs.ch)
}

// Update will populate the quota service with quota and limits information.
func (qs *Quotas) Update(tenantID string, quotas []types.QuotaDetails) {
	ch := make(chan struct{})
	op := &updateOp{tenantID, quotas, ch}
	qs.ch <- op
	<-ch
}

// DumpQuotas provides the list of quotas and limits along with usage
// for a given tenant
func (qs *Quotas) DumpQuotas(tenantID string) []types.QuotaDetails {
	ch := make(chan []types.QuotaDetails, 1)
	op := &dumpOp{tenantID, ch}
	qs.ch <- op
	qds := <-ch
	return qds
}

// Allowed indicates whether the desired consumption should be permitted.
func (r *result) Allowed() bool {
	return r.allowed
}

// Reason gives an explanation for why the request should be denied.
func (r *result) Reason() string {
	return r.reason
}

// Resources gives the set of resources that made up the consumption request
// that this Result was associated with.
func (r *result) Resources() []payloads.RequestedResource {
	return r.resources
}
