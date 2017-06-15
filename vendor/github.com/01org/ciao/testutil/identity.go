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

package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gorilla/mux"
)

// ComputeAPIPort is the compute service port the testutil identity service will use by default
const ComputeAPIPort = "8774"

// VolumeAPIPort is the volume service port the testutil identity service will use by default
const VolumeAPIPort = "8776"

// ImageAPIPort is the image service port the testutil identity service will use by default
const ImageAPIPort = "9292"

// ComputeURL is the compute service URL the testutil identity service will use by default
var ComputeURL = "https://localhost:" + ComputeAPIPort

// VolumeURL is the volume service URL the testutil identity service will use by default
var VolumeURL = "https://localhost:" + VolumeAPIPort

// ImageURL is the image service URL the testutil identity service will use by default
var ImageURL = "https://localhost:" + ImageAPIPort

// IdentityURL is the URL for the testutil identity service
var IdentityURL string

// ComputeUser is the test user/tenant name the testutil identity service will use by default
var ComputeUser = "f452bbc7-5076-44d5-922c-3b9d2ce1503f"

func authHandler(w http.ResponseWriter, r *http.Request) {
	cinderv2URL := VolumeURL + "/v2/" + ComputeUser
	token := `
		{
			"token": {
				"methods": [
					"password"
				],
				"roles": [
					{
						"id" : "12345",
						"name" : "admin"
					}
				],
				"expires_at": "%s",
				"project": {
					"domain": {
						"id": "default",
						"name": "Default"
					},
					"id": "%s",
					"name": "admin"
				},
				"catalog": [
					{
						"endpoints": [
							{
								"region_id": "RegionOne",
								"url": "%[3]s/v3",
								"region": "RegionOne",
								"interface": "public",
								"id": "068d1b359ee84b438266cb736d81de97"
							},
							{
								"region_id": "RegionOne",
								"url": "%[3]s/v3",
								"region": "RegionOne",
								"interface": "admin",
								"id": "8bfc846841ab441ca38471be6d164ced"
							},
							{
								"region_id": "RegionOne",
								"url": "%[3]s/v3",
								"region": "RegionOne",
								"interface": "internal",
								"id": "beb6d358c3654b4bada04d4663b640b9"
							}
						],
						"type": "identity",
						"id": "050726f278654128aba89757ae25950c",
						"name": "keystone"
					},
					{
						"endpoints": [
							{
								"region_id": "RegionOne",
								"url": "%[4]s",
								"region": "RegionOne",
								"interface": "internal",
								"id": "823c65e659d64733b86a609a96bcc48f"
							},
							{
								"region_id": "RegionOne",
								"url": "%[4]s",
								"region": "RegionOne",
								"id": "65796adae7b24663a2b9114fa34314c7",
								"interface": "admin"
							},
							{
								"region_id": "RegionOne",
								"url": "%[4]s",
								"region": "RegionOne",
								"id": "0eddcb06eac24b9bae7d9db516a40fdb",
								"interface": "public"
							}
						],
						"id": "fbd0a99c-01c0-4fc2-96b9-c76e01def567",
						"name": "cinderv2",
						"type": "volumev2"
					},
					{
						"endpoints": [
							{
								"region_id": "RegionOne",
								"url": "%[5]s",
								"region": "RegionOne",
								"id": "a4989bcbc54a4b4ca47f91ffb8adf5f7",
								"interface": "internal"
							},
							{
								"region_id": "RegionOne",
								"url": "%[5]s",
								"region": "RegionOne",
								"id": "025515ea9f664368a26eeefd56f5cab5",
								"interface": "admin"
							},
							{
								"region_id": "RegionOne",
								"url": "%[5]s",
								"region": "RegionOne",
								"id": "dd9cdb516d5745d9a8d7215bc712543a",
								"interface": "public"
							}
						],
						"id": "ecbff61f-92d9-48b6-bbb5-73153e7bfe26",
						"name": "glance",
						"type": "image"
					}
				],
			       "extras": {},
			       "user": {
				       "domain": {
				               "id": "default",
				               "name": "Default"
				        },
				       "id": "ee4dfb6e5540447cb3741905149d9b6e",
			               "name": "admin"
			        },
			        "audit_ids": [
				        "3T2dc1CGQxyJsHdDu1xkcw"
			        ],
			        "issued_at": "%[6]s"
			}
		}`

	t := []byte(fmt.Sprintf(token,
		time.Now().Add(1*time.Hour).Format(gophercloud.RFC3339Milli),
		ComputeUser, IdentityURL, cinderv2URL, ImageURL,
		time.Now().Format(gophercloud.RFC3339Milli)))
	w.Header().Set("X-Subject-Token", "imavalidtoken")
	w.WriteHeader(http.StatusCreated)
	w.Write(t)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	tenantURL := ComputeURL + "/v2.1/" + ComputeUser
	token := `
	{
		"token": {
		        "methods": [
		                "token"
		        ],
		        "expires_at": "%s",
		        "extras": {},
		        "user": {
				"domain": {
					"id": "default",
					"name": "Default"
				},
				"id": "10a2e6e717a245d9acad3e5f97aeca3d",
				"name": "admin"
			},
			"roles": [
				{
					"id" : "12345",
					"name" : "admin"
				}
			],
			"project": {
				"domain": {
					"id": "default",
					"name": "Default"
				},
				"id": "%s",
				"name": "admin"
			},
			"catalog": [
				{
					"endpoints": [
						{
							"region_id": "RegionOne",
							"url": "%[3]s/v3",
							"region": "RegionOne",
							"interface": "public",
							"id": "068d1b359ee84b438266cb736d81de97"
						},
						{
							"region_id": "RegionOne",
							"url": "%[3]s/v3",
							"region": "RegionOne",
							"interface": "admin",
							"id": "8bfc846841ab441ca38471be6d164ced"
						},
						{
							"region_id": "RegionOne",
							"url": "%[3]s/v3",
							"region": "RegionOne",
							"interface": "internal",
							"id": "beb6d358c3654b4bada04d4663b640b9"
						}
					],
					"type": "identity",
					"id": "050726f278654128aba89757ae25950c",
					"name": "keystone"
				},
				{
			                "endpoints": [
					         {
							"region_id": "RegionOne",
							"url": "%[4]s",
							"region": "RegionOne",
							"interface": "admin",
							"id": "2511589f262a407bb0071a814a480af4"
						},
						{
							"region_id": "RegionOne",
							"url": "%[4]s",
							"region": "RegionOne",
							"interface": "internal",
							"id": "9cf9209ae4fc4673a7295611001cf0ae"
						},
						{
							"region_id": "RegionOne",
							"url": "%[4]s",
							"region": "RegionOne",
							"interface": "public",
							"id": "d200b2509e1343e3887dcc465b4fa534"
						}
					],
					"type": "compute",
					"id": "a226b3eeb5594f50bf8b6df94636ed28",
					"name": "ciao"
				}
			],
			"audit_ids": [
			        "mAjXQhiYRyKwkB4qygdLVg"
			],
			"issued_at": "%[5]s"
		}
	}`

	t := []byte(fmt.Sprintf(token,
		time.Now().Add(1*time.Hour).Format(gophercloud.RFC3339Milli),
		ComputeUser, IdentityURL, tenantURL,
		time.Now().Format(gophercloud.RFC3339Milli)))
	w.WriteHeader(http.StatusOK)
	w.Write(t)
}

func projectsHandler(w http.ResponseWriter, r *http.Request) {
	response := `
	{
		"projects": [
			{
				"description": "fake project1",
				"domain_id": "default",
				"enabled": true,
				"id": "456788",
				"parent_id": "212223",
				"links": {
					"self": "%s/v3/projects/456788"
				},
				"name": "ilovepuppies"
			}
		],
		"links": {
			"self": "%s/v3/users/10a2e6e717a245d9acad3e5f97aeca3d/projects",
			"previous": null,
			"next": null
		}
	}`

	p := []byte(fmt.Sprintf(response, IdentityURL, IdentityURL))
	w.WriteHeader(http.StatusOK)
	w.Write(p)
}

// IdentityHandlers creates a mux.Router for identity POST and GET handlers
func IdentityHandlers() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/v3/auth/tokens", authHandler).Methods("POST")
	r.HandleFunc("/v3/auth/tokens", validateHandler).Methods("GET")
	r.HandleFunc("/v3/users/10a2e6e717a245d9acad3e5f97aeca3d/projects", projectsHandler).Methods("GET")
	r.HandleFunc("/v3/projects", projectsHandler).Methods("GET")

	return r
}

// IdentityConfig contains the URL of the ciao compute and volume services, and the
// TenantID of the tenant you want tokens to be sent for.  The test Identity service
// only supports authentication of a single tenant, and gives the token an admin role.
type IdentityConfig struct {
	VolumeURL  string
	ImageURL   string
	ComputeURL string
	ProjectID  string
}

// StartIdentityServer starts a fake keystone service for unit testing ciao.
// Caller must call Close() on the returned *httptest.Server.
func StartIdentityServer(config IdentityConfig) *httptest.Server {
	id := httptest.NewServer(IdentityHandlers())
	if id == nil {
		return nil
	}

	if config.VolumeURL != "" {
		VolumeURL = config.VolumeURL
	}
	if config.ImageURL != "" {
		ImageURL = config.ImageURL
	}
	if config.ComputeURL != "" {
		ComputeURL = config.ComputeURL
	}
	if config.ProjectID != "" {
		ComputeUser = config.ProjectID
	}
	IdentityURL = id.URL

	return id
}
