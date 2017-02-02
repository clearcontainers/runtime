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
	"errors"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
)

type identity struct {
	scV3 *gophercloud.ServiceClient
}

type identityConfig struct {
	endpoint        string
	serviceUserName string
	servicePassword string
}

func newIdentityClient(config identityConfig) (*identity, error) {
	opt := gophercloud.AuthOptions{
		IdentityEndpoint: config.endpoint + "/v3/",
		Username:         config.serviceUserName,
		Password:         config.servicePassword,
		TenantName:       "service",
		DomainID:         "default",
		AllowReauth:      true,
	}
	provider, err := openstack.AuthenticatedClient(opt)
	if err != nil {
		return nil, err
	}

	v3client := openstack.NewIdentityV3(provider)
	if v3client == nil {
		return nil, errors.New("Unable to get keystone V3 client")
	}

	id := &identity{
		scV3: v3client,
	}

	return id, err
}
