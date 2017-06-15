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

package identity

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/01org/ciao/testutil"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gorilla/mux"
)

var validServices = []ValidService{
	{"compute", "ciao"},
	{"compute", "nova"},
}

var validAdmins = []ValidAdmin{
	{"service", "admin"},
	{"admin", "admin"},
}

var invalidAdmins = []ValidAdmin{
	{"test", "test"},
}

var invalidServices = []ValidService{
	{"test", "test"},
}

var anyOldComputeServices = []ValidService{
	{"compute", "whatevahs"},
}

type test struct {
	URL              string
	route            string
	validServices    []ValidService
	validAdmins      []ValidAdmin
	expectedResponse int
}

var tests = []test{
	{fmt.Sprintf("/v2/%s/volumes", testutil.ComputeUser), "/v2/{tenant}/volumes", validServices, validAdmins, 200},
	{"/v2.1/tenants", "/v2.1/tenants", validServices, validAdmins, 200},
	{"/v2.1/tenants", "/v2.1/tenants", validServices, invalidAdmins, 200},
	{fmt.Sprintf("/v2/%s/volumes", testutil.ComputeUser), "/v2/{tenant}/volumes", invalidServices, invalidAdmins, 401},
	{fmt.Sprintf("/v2/%s/volumes", testutil.ComputeUser), "/v2/{tenant}/volumes", anyOldComputeServices, invalidAdmins, 200},
	{fmt.Sprintf("/v2/%s/volumes", "unknowntenantid"), "/v2/{tenant}/volumes", validServices, invalidAdmins, 401},
}

func getIdentityClient(endpoint string) (*gophercloud.ServiceClient, error) {
	opt := gophercloud.AuthOptions{
		IdentityEndpoint: endpoint + "v3/",
		Username:         "ciao",
		Password:         "iheartciao",
		TenantName:       "service",
		DomainID:         "default",
		AllowReauth:      true,
	}
	provider, err := openstack.AuthenticatedClient(opt)
	if err != nil {
		return nil, err
	}

	v3client, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("Unable to get keystone V3 client : %v", err)
	}

	return v3client, nil
}

func TestNoToken(t *testing.T) {
	testIdentityConfig := testutil.IdentityConfig{
		ComputeURL: testutil.ComputeURL,
		ProjectID:  testutil.ComputeUser,
	}

	id := testutil.StartIdentityServer(testIdentityConfig)
	if id == nil {
		t.Fatal("Could not start test identity server")
	}

	defer id.Close()

	client, err := getIdentityClient(id.URL + "/")
	if err != nil {
		t.Fatal(err)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("hello")
	})

	validServices := []ValidService{
		{"compute", "ciao"},
		{"compute", "nova"},
	}

	validAdmins := []ValidAdmin{
		{"service", "admin"},
		{"admin", "admin"},
	}

	h := Handler{
		Client:        client,
		Next:          &testHandler,
		ValidServices: validServices,
		ValidAdmins:   validAdmins,
	}

	for _, tt := range tests {
		req, err := http.NewRequest("GET", tt.URL, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		r := mux.NewRouter()

		r.Handle(tt.route, h).Methods("GET")

		r.ServeHTTP(rr, req)

		status := rr.Code
		if status != http.StatusUnauthorized {
			t.Errorf("got %v: expected %v", status, http.StatusUnauthorized)
		}
	}
}

func TestHandler(t *testing.T) {
	testIdentityConfig := testutil.IdentityConfig{
		ComputeURL: testutil.ComputeURL,
		ProjectID:  testutil.ComputeUser,
	}

	id := testutil.StartIdentityServer(testIdentityConfig)
	if id == nil {
		t.Fatal("Could not start test identity server")
	}

	defer id.Close()

	client, err := getIdentityClient(id.URL + "/")
	if err != nil {
		t.Fatal(err)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		return
	})

	for _, tt := range tests {
		h := Handler{
			Client:        client,
			Next:          &testHandler,
			ValidServices: tt.validServices,
			ValidAdmins:   tt.validAdmins,
		}

		req, err := http.NewRequest("GET", tt.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Auth-Token", "imaninvalidtoken")

		rr := httptest.NewRecorder()

		r := mux.NewRouter()

		r.Handle(tt.route, h).Methods("GET")

		r.ServeHTTP(rr, req)

		status := rr.Code
		if status != tt.expectedResponse {
			t.Errorf("got %v: expected %v", status, tt.expectedResponse)
		}
	}
}
