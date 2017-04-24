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

package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/01org/ciao/ciao-controller/types"
)

type test struct {
	method           string
	pattern          string
	handler          func(*Context, http.ResponseWriter, *http.Request) (Response, error)
	request          string
	media            string
	expectedStatus   int
	expectedResponse string
}

func myHostname() string {
	host, _ := os.Hostname()
	return host
}

var tests = []test{
	{
		"GET",
		"/",
		listResources,
		"",
		"application/text",
		http.StatusOK,
		`[{"rel":"pools","href":"/pools","version":"x.ciao.pools.v1","minimum_version":"x.ciao.pools.v1"},{"rel":"external-ips","href":"/external-ips","version":"x.ciao.external-ips.v1","minimum_version":"x.ciao.external-ips.v1"},{"rel":"workloads","href":"/workloads","version":"x.ciao.workloads.v1","minimum_version":"x.ciao.workloads.v1"},{"rel":"tenants","href":"/tenants","version":"x.ciao.tenants.v1","minimum_version":"x.ciao.tenants.v1"}]`,
	},
	{
		"GET",
		"/pools",
		listPools,
		"",
		"application/x.ciao.v1.pools",
		http.StatusOK,
		`{"pools":[{"id":"validID","name":"testpool","free":0,"total_ips":0,"links":[{"rel":"self","href":"/pools/validID"}]}]}`,
	},
	{
		"GET",
		"/pools?name=testpool",
		listPools,
		"",
		"application/x.ciao.v1.pools",
		http.StatusOK,
		`{"pools":[{"id":"validID","name":"testpool","free":0,"total_ips":0,"links":[{"rel":"self","href":"/pools/validID"}]}]}`,
	},
	{
		"POST",
		"/pools",
		addPool,
		`{"name":"testpool"}`,
		"application/x.ciao.v1.pools",
		http.StatusNoContent,
		"null",
	},
	{
		"GET",
		"/pools/validID",
		showPool,
		"",
		"application/x.ciao.v1.pools",
		http.StatusOK,
		`{"id":"validID","name":"testpool","free":0,"total_ips":0,"links":[{"rel":"self","href":"/pools/validID"}],"subnets":[],"ips":[]}`,
	},
	{
		"DELETE",
		"/pools/validID",
		deletePool,
		"",
		"application/x.ciao.v1.pools",
		http.StatusNoContent,
		"null",
	},
	{
		"POST",
		"/pools/validID",
		addToPool,
		`{"subnet":"192.168.0.0/24"}`,
		"application/x.ciao.v1.pools",
		http.StatusNoContent,
		"null",
	},
	{
		"DELETE",
		"/pools/validID/subnets/validID",
		deleteSubnet,
		"",
		"application/x.ciao.v1.pools",
		http.StatusNoContent,
		"null",
	},
	{
		"DELETE",
		"/pools/validID/external-ips/validID",
		deleteExternalIP,
		"",
		"application/x.ciao.v1.pools",
		http.StatusNoContent,
		"null",
	},
	{
		"GET",
		"/external-ips",
		listMappedIPs,
		"",
		ExternalIPsV1,
		http.StatusOK,
		`[{"mapping_id":"validID","external_ip":"192.168.0.1","internal_ip":"172.16.0.1","instance_id":"","tenant_id":"validtenant","pool_id":"validpool","pool_name":"mypool","links":[{"rel":"self","href":"/external-ips/validID"},{"rel":"pool","href":"/pools/validpool"}]}]`,
	},
	{
		"POST",
		"/validID/external-ips",
		mapExternalIP,
		`{"pool_name":"apool","instance_id":"validinstanceID"}`,
		"application/x.ciao.v1.pools",
		http.StatusNoContent,
		"null",
	},
	{
		"POST",
		"/workloads",
		addWorkload,
		`{"id":"","description":"testWorkload","fw_type":"legacy","vm_type":"qemu","image_id":"73a86d7e-93c0-480e-9c41-ab42f69b7799","image_name":"","config":"this will totally work!","defaults":[]}`,
		"application/x.ciao.v1.workloads",
		http.StatusCreated,
		`{"workload":{"id":"ba58f471-0735-4773-9550-188e2d012941","description":"testWorkload","fw_type":"legacy","vm_type":"qemu","image_id":"73a86d7e-93c0-480e-9c41-ab42f69b7799","image_name":"","config":"this will totally work!","defaults":[],"storage":null},"link":{"rel":"self","href":"/workloads/ba58f471-0735-4773-9550-188e2d012941"}}`,
	},
	{
		"GET",
		"/tenants/test-tenant-id/quotas/",
		listQuotas,
		"",
		"application/x.ciao.v1.tenants",
		http.StatusOK,
		`{"quotas":[{"name":"test-quota-1","value":"10","usage":"3"},{"name":"test-quota-2","value":"unlimited","usage":"10"},{"name":"test-limit","value":"123"}]}`,
	},
}

type testCiaoService struct{}

func (ts testCiaoService) ListPools() ([]types.Pool, error) {
	self := types.Link{
		Rel:  "self",
		Href: "/pools/validID",
	}

	resp := types.Pool{
		ID:       "validID",
		Name:     "testpool",
		Free:     0,
		TotalIPs: 0,
		Subnets:  []types.ExternalSubnet{},
		IPs:      []types.ExternalIP{},
		Links:    []types.Link{self},
	}

	return []types.Pool{resp}, nil
}

func (ts testCiaoService) AddPool(name string, subnet *string, ips []string) (types.Pool, error) {
	return types.Pool{}, nil
}

func (ts testCiaoService) ShowPool(id string) (types.Pool, error) {
	self := types.Link{
		Rel:  "self",
		Href: "/pools/validID",
	}

	resp := types.Pool{
		ID:       "validID",
		Name:     "testpool",
		Free:     0,
		TotalIPs: 0,
		Subnets:  []types.ExternalSubnet{},
		IPs:      []types.ExternalIP{},
		Links:    []types.Link{self},
	}

	return resp, nil
}

func (ts testCiaoService) DeletePool(id string) error {
	return nil
}

func (ts testCiaoService) AddAddress(poolID string, subnet *string, ips []string) error {
	return nil
}

func (ts testCiaoService) RemoveAddress(poolID string, subnet *string, extIP *string) error {
	return nil
}

func (ts testCiaoService) ListMappedAddresses(tenant *string) []types.MappedIP {
	var ref string

	m := types.MappedIP{
		ID:         "validID",
		ExternalIP: "192.168.0.1",
		InternalIP: "172.16.0.1",
		TenantID:   "validtenant",
		PoolID:     "validpool",
		PoolName:   "mypool",
	}

	if tenant != nil {
		ref = fmt.Sprintf("%s/external-ips/%s", *tenant, m.ID)
	} else {
		ref = fmt.Sprintf("/external-ips/%s", m.ID)
	}

	link := types.Link{
		Rel:  "self",
		Href: ref,
	}

	m.Links = []types.Link{link}

	if tenant == nil {
		ref := fmt.Sprintf("/pools/%s", m.PoolID)

		link := types.Link{
			Rel:  "pool",
			Href: ref,
		}

		m.Links = append(m.Links, link)
	}

	return []types.MappedIP{m}
}

func (ts testCiaoService) MapAddress(name *string, instanceID string) error {
	return nil
}

func (ts testCiaoService) UnMapAddress(string) error {
	return nil
}

func (ts testCiaoService) CreateWorkload(req types.Workload) (types.Workload, error) {
	req.ID = "ba58f471-0735-4773-9550-188e2d012941"
	return req, nil
}

func (ts testCiaoService) ListQuotas(tenantID string) []types.QuotaDetails {
	return []types.QuotaDetails{
		{Name: "test-quota-1", Value: 10, Usage: 3},
		{Name: "test-quota-2", Value: -1, Usage: 10},
		{Name: "test-limit", Value: 123, Usage: 0},
	}
}

func (ts testCiaoService) UpdateQuotas(tenantID string, qds []types.QuotaDetails) error {
	return nil
}

func TestResponse(t *testing.T) {
	var ts testCiaoService

	context := &Context{"", ts}

	for _, tt := range tests {
		req, err := http.NewRequest(tt.method, tt.pattern, bytes.NewBuffer([]byte(tt.request)))
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", tt.media)

		rr := httptest.NewRecorder()
		handler := Handler{context, tt.handler}

		handler.ServeHTTP(rr, req)

		status := rr.Code
		if status != tt.expectedStatus {
			t.Errorf("got %v, expected %v", status, tt.expectedStatus)
		}

		if rr.Body.String() != tt.expectedResponse {
			t.Errorf("%s: failed\ngot: %v\nexp: %v", tt.pattern, rr.Body.String(), tt.expectedResponse)
		}
	}
}

func TestRoutes(t *testing.T) {
	var ts testCiaoService
	config := Config{"", ts}

	r := Routes(config)
	if r == nil {
		t.Fatalf("No routes returned")
	}
}
