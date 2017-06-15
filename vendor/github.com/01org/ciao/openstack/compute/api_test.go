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

package compute

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type test struct {
	method           string
	pattern          string
	handler          func(*Context, http.ResponseWriter, *http.Request) (APIResponse, error)
	request          string
	expectedStatus   int
	expectedResponse string
}

func myHostname() string {
	host, _ := os.Hostname()
	return host
}

var tests = []test{
	{
		"POST",
		"/v2.1/{tenant}/servers/",
		createServer,
		`{"server":{"name":"new-server-test","imageRef": "http://glance.openstack.example.com/images/70a599e0-31e7-49b7-b260-868f441e862b","flavorRef":"http://openstack.example.com/flavors/1","metadata":{"My Server Name":"Apache1"}}}`,
		http.StatusAccepted,
		`{"server":{"id":"validServerID","name":"new-server-test","imageRef":"http://glance.openstack.example.com/images/70a599e0-31e7-49b7-b260-868f441e862b","flavorRef":"http://openstack.example.com/flavors/1","max_count":0,"min_count":0,"metadata":{"My Server Name":"Apache1"}}}`,
	},
	{
		"GET",
		"/v2.1/{tenant}/servers/detail?limit=1&offset=1",
		ListServersDetails,
		"",
		http.StatusOK,
		`{"total_servers":1,"servers":[]}`,
	},
	{
		"GET",
		"/v2.1/{tenant}/servers/detail",
		ListServersDetails,
		"",
		http.StatusOK,
		`{"total_servers":1,"servers":[{"addresses":{"private":[{"addr":"192.169.0.1","OS-EXT-IPS-MAC:mac_addr":"00:02:00:01:02:03","OS-EXT-IPS:type":"","version":0}]},"created":"0001-01-01T00:00:00Z","flavor":{"id":"testFlavorUUID","links":null},"hostId":"hostUUID","id":"testUUID","image":{"id":"testImageUUID","links":null},"key_name":"","links":null,"name":"","accessIPv4":"","accessIPv6":"","config_drive":"","OS-DCF:diskConfig":"","OS-EXT-AZ:availability_zone":"","OS-EXT-SRV-ATTR:host":"","OS-EXT-SRV-ATTR:hypervisor_hostname":"","OS-EXT-SRV-ATTR:instance_name":"","OS-EXT-STS:power_state":0,"OS-EXT-STS:task_state":"","OS-EXT-STS:vm_state":"","os-extended-volumes:volumes_attached":null,"OS-SRV-USG:launched_at":"0001-01-01T00:00:00Z","OS-SRV-USG:terminated_at":"0001-01-01T00:00:00Z","progress":0,"security_groups":null,"status":"active","host_status":"","tenant_id":"","updated":"0001-01-01T00:00:00Z","user_id":"","ssh_ip":"","ssh_port":0}]}`,
	},
	{
		"GET",
		"/v2.1/{tenant}/servers/{server}",
		showServerDetails,
		"",
		http.StatusOK,
		`{"server":{"addresses":{"private":[{"addr":"192.169.0.1","OS-EXT-IPS-MAC:mac_addr":"00:02:00:01:02:03","OS-EXT-IPS:type":"","version":0}]},"created":"0001-01-01T00:00:00Z","flavor":{"id":"testFlavorUUID","links":null},"hostId":"hostUUID","id":"","image":{"id":"testImageUUID","links":null},"key_name":"","links":null,"name":"","accessIPv4":"","accessIPv6":"","config_drive":"","OS-DCF:diskConfig":"","OS-EXT-AZ:availability_zone":"","OS-EXT-SRV-ATTR:host":"","OS-EXT-SRV-ATTR:hypervisor_hostname":"","OS-EXT-SRV-ATTR:instance_name":"","OS-EXT-STS:power_state":0,"OS-EXT-STS:task_state":"","OS-EXT-STS:vm_state":"","os-extended-volumes:volumes_attached":null,"OS-SRV-USG:launched_at":"0001-01-01T00:00:00Z","OS-SRV-USG:terminated_at":"0001-01-01T00:00:00Z","progress":0,"security_groups":null,"status":"active","host_status":"","tenant_id":"","updated":"0001-01-01T00:00:00Z","user_id":"","ssh_ip":"","ssh_port":0}}`,
	},
	{
		"DELETE",
		"/v2.1/{tenant}/servers/{server}",
		deleteServer,
		"",
		http.StatusNoContent,
		"null",
	},
	{
		"POST",
		"/v2.1/{tenant}/servers/{server}/action",
		serverAction,
		`{"os-start":null}`,
		http.StatusAccepted,
		"null",
	},
	{
		"POST",
		"/v2.1/{tenant}/servers/{server}/action",
		serverAction,
		`{"os-stop":null}`,
		http.StatusAccepted,
		"null",
	},
	{
		"GET",
		"/v2.1/{tenant}/flavors/",
		listFlavors,
		"",
		http.StatusOK,
		`{"flavors":[{"id":"flavorUUID","links":null,"name":"testflavor"}]}`,
	},
	{
		"GET",
		"/v2.1/{tenant}/flavors/",
		listFlavorsDetails,
		"",
		http.StatusOK,
		`{"flavors":[{"OS-FLV-DISABLED:disabled":false,"disk":1024,"OS-FLV-EXT-DATA:ephemeral":0,"os-flavor-access:is_public":true,"id":"workloadUUID","links":null,"name":"testflavor","ram":256,"swap":"","vcpus":2}]}`,
	},
	{
		"GET",
		"/v2.1/{tenant}/flavors/",
		showFlavorDetails,
		"",
		http.StatusOK,
		`{"flavor":{"OS-FLV-DISABLED:disabled":false,"disk":1024,"OS-FLV-EXT-DATA:ephemeral":0,"os-flavor-access:is_public":true,"id":"workloadUUID","links":null,"name":"testflavor","ram":256,"swap":"","vcpus":2}}`,
	},
}

type testComputeService struct{}

// server interfaces
func (cs testComputeService) CreateServer(tenant string, req CreateServerRequest) (interface{}, error) {
	req.Server.ID = "validServerID"
	return req, nil
}

func (cs testComputeService) ListServersDetail(tenant string) ([]ServerDetails, error) {
	var servers []ServerDetails

	server := ServerDetails{
		HostID:   "hostUUID",
		ID:       "testUUID",
		TenantID: tenant,
		Flavor: FlavorLinks{
			ID: "testFlavorUUID",
		},
		Image: Image{
			ID: "testImageUUID",
		},
		Status: "active",
		Addresses: Addresses{
			Private: []PrivateAddresses{
				{
					Addr:               "192.169.0.1",
					OSEXTIPSMACMacAddr: "00:02:00:01:02:03",
				},
			},
		},
	}

	servers = append(servers, server)

	return servers, nil
}

func (cs testComputeService) ShowServerDetails(tenant string, server string) (Server, error) {
	s := ServerDetails{
		HostID:   "hostUUID",
		ID:       server,
		TenantID: tenant,
		Flavor: FlavorLinks{
			ID: "testFlavorUUID",
		},
		Image: Image{
			ID: "testImageUUID",
		},
		Status: "active",
		Addresses: Addresses{
			Private: []PrivateAddresses{
				{
					Addr:               "192.169.0.1",
					OSEXTIPSMACMacAddr: "00:02:00:01:02:03",
				},
			},
		},
	}

	return Server{Server: s}, nil
}

func (cs testComputeService) DeleteServer(tenant string, server string) error {
	return nil
}

func (cs testComputeService) StartServer(tenant string, server string) error {
	return nil
}

func (cs testComputeService) StopServer(tenant string, server string) error {
	return nil
}

//flavor interfaces
func (cs testComputeService) ListFlavors(string) (Flavors, error) {
	flavors := NewComputeFlavors()

	flavors.Flavors = append(flavors.Flavors,
		struct {
			ID    string `json:"id"`
			Links []Link `json:"links"`
			Name  string `json:"name"`
		}{
			ID:   "flavorUUID",
			Name: "testflavor",
		},
	)

	return flavors, nil
}

func (cs testComputeService) ListFlavorsDetail(string) (FlavorsDetails, error) {
	flavors := NewComputeFlavorsDetails()
	var details FlavorDetails

	details.OsFlavorAccessIsPublic = true
	details.ID = "workloadUUID"
	details.Disk = 1024
	details.Name = "testflavor"
	details.Vcpus = 2
	details.RAM = 256

	flavors.Flavors = append(flavors.Flavors, details)

	return flavors, nil
}

func (cs testComputeService) ShowFlavorDetails(string, string) (Flavor, error) {
	var details FlavorDetails
	var flavor Flavor

	details.OsFlavorAccessIsPublic = true
	details.ID = "workloadUUID"
	details.Disk = 1024
	details.Name = "testflavor"
	details.Vcpus = 2
	details.RAM = 256

	flavor.Flavor = details

	return flavor, nil
}

func TestAPIResponse(t *testing.T) {
	var cs testComputeService

	// TBD: add context to test definition so it can be created per
	// endpoint with either a pass testComputeService or a failure
	// one.
	context := &Context{8774, cs}

	for _, tt := range tests {
		req, err := http.NewRequest(tt.method, tt.pattern, bytes.NewBuffer([]byte(tt.request)))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := APIHandler{context, tt.handler}

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
	var cs testComputeService
	config := APIConfig{8774, cs}

	r := Routes(config)
	if r == nil {
		t.Fatalf("No routes returned")
	}
}

func TestPager(t *testing.T) {
	req, err := http.NewRequest("GET", "/v2.1/{tenant}/servers/detail?limit=2&offset=2", bytes.NewBuffer([]byte("")))

	if err != nil {
		t.Fatal(err)
	}
	limit, offset, _ := pagerQueryParse(req)
	if limit != 2 {
		t.Fatalf("Invalid limit registered")
	}
	if offset != 2 {
		t.Fatalf("Invalid offset registered")
	}
}
