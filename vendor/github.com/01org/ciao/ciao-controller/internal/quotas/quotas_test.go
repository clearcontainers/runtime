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
	"reflect"
	"testing"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
)

func TestConsumeAndRelease(t *testing.T) {
	qs := &Quotas{}
	qs.Init()

	quotas := []types.QuotaDetails{{Name: "tenant-vcpu-quota", Value: 10}}

	qs.Update("test-tenant-1", quotas)

	// "first instance""
	ch := qs.Consume("test-tenant-1", payloads.RequestedResource{Type: payloads.VCPUs, Value: 10})
	res := <-ch

	if !res.Allowed() {
		t.Fatal("Expected to be allowed")
	}

	// "second instance"
	ch = qs.Consume("test-tenant-1", payloads.RequestedResource{Type: payloads.VCPUs, Value: 10})
	res2 := <-ch

	if res2.Allowed() {
		t.Fatal("Expected to be denied")
	}
	// If denied we are responsible for releasing
	qs.Release("test-tenant-1", res2.Resources()...)

	// Now release "first instance"
	qs.Release("test-tenant-1", res.Resources()...)

	ch = qs.Consume("test-tenant-1", payloads.RequestedResource{Type: payloads.VCPUs, Value: 10})
	res3 := <-ch

	if !res3.Allowed() {
		t.Fatal("Expected to be allowed")
	}

	qs.Shutdown()
}

func testHasQuota(t *testing.T, qds []types.QuotaDetails, qd types.QuotaDetails) {
	for i := range qds {
		if reflect.DeepEqual(qd, qds[i]) {
			return
		}
	}
	t.Fatalf("Quota not found: %+v", qd)
}

func TestDumpQuotas(t *testing.T) {
	qs := &Quotas{}
	qs.Init()

	quotas := []types.QuotaDetails{
		{Name: "tenant-vcpu-quota", Value: 10},
		{Name: "tenant-mem-quota", Value: 100},
	}

	qs.Update("test-tenant-1", quotas)

	dumpedQuotas := qs.DumpQuotas("test-tenant-1")

	testHasQuota(t, dumpedQuotas, quotas[0])
	testHasQuota(t, dumpedQuotas, quotas[1])

	qs.Shutdown()
}

func TestTenantSeparation(t *testing.T) {
	qs := &Quotas{}
	qs.Init()

	t1Quotas := []types.QuotaDetails{
		{Name: "tenant-vcpu-quota", Value: 10},
		{Name: "tenant-mem-quota", Value: 100},
	}

	qs.Update("test-tenant-1", t1Quotas)

	t2Quotas := []types.QuotaDetails{
		{Name: "tenant-vcpu-quota", Value: 20},
		{Name: "tenant-mem-quota", Value: 40},
	}

	qs.Update("test-tenant-2", t2Quotas)

	dumpedQuotas := qs.DumpQuotas("test-tenant-1")

	testHasQuota(t, dumpedQuotas, t1Quotas[0])
	testHasQuota(t, dumpedQuotas, t1Quotas[1])

	dumpedQuotas = qs.DumpQuotas("test-tenant-2")

	testHasQuota(t, dumpedQuotas, t2Quotas[0])
	testHasQuota(t, dumpedQuotas, t2Quotas[1])

	qs.Shutdown()
}

func TestResourceQuotaMapping(t *testing.T) {
	resources := []payloads.Resource{
		payloads.VCPUs,
		payloads.MemMB,
		payloads.SharedDiskGiB,
		payloads.Volume,
		payloads.Instance,
		payloads.Image,
		payloads.ExternalIP,
	}

	for _, resource := range resources {
		qn := resourceToQuotaName(resource)
		r := quotaNameToResource(qn)

		if r != resource {
			t.Fatal("Expected resources to be equal")
		}
	}
}

func TestAllLimits(t *testing.T) {
	qs := &Quotas{}
	qs.Init()

	limits := []types.QuotaDetails{
		{Name: "tenant-vcpu-per-instance-limit", Value: 4},
		{Name: "tenant-mem-per-instance-limit", Value: 128},
		{Name: "tenant-volume-size-limit", Value: 10},
	}

	qs.Update("test-tenant-1", limits)

	dumpedQuotas := qs.DumpQuotas("test-tenant-1")

	for _, qd := range limits {
		testHasQuota(t, dumpedQuotas, qd)
	}

	// Over limit tests
	rrs := []payloads.RequestedResource{
		{
			Type:  payloads.VCPUs,
			Value: 8,
		},
		{
			Type:  payloads.MemMB,
			Value: 512,
		},
		{
			Type:  payloads.SharedDiskGiB,
			Value: 20,
		},
	}

	for _, rr := range rrs {
		r := <-qs.Consume("test-tenant-1", rr)
		if r.Allowed() && r.Reason() != "Over limit" {
			t.Fatalf("Expected to be over limit for: %s", rr.Type)
		}
	}

	// Under limit tests
	rrs = []payloads.RequestedResource{
		{
			Type:  payloads.VCPUs,
			Value: 4,
		},
		{
			Type:  payloads.MemMB,
			Value: 128,
		},
		{
			Type:  payloads.SharedDiskGiB,
			Value: 10,
		},
	}
	for _, rr := range rrs {
		r := <-qs.Consume("test-tenant-1", rr)
		if !r.Allowed() {
			t.Fatalf("Expected to be under limit for: %s", rr.Type)
		}
	}
}
