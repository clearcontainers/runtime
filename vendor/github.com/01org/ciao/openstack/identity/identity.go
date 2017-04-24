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
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
	"github.com/rackspace/gophercloud"
	v3tokens "github.com/rackspace/gophercloud/openstack/identity/v3/tokens"
)

// Project holds project information extracted from the keystone response.
type Project struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
}

// RoleEntry contains the name of a role extracted from the keystone response.
type RoleEntry struct {
	Name string `mapstructure:"name"`
}

// Roles contains a list of role names extracted from the keystone response.
type Roles struct {
	Entries []RoleEntry
}

// Endpoint contains endpoint information extracted from the keystone response.
type Endpoint struct {
	ID        string `mapstructure:"id"`
	Region    string `mapstructure:"region"`
	Interface string `mapstructure:"interface"`
	URL       string `mapstructure:"url"`
}

// ServiceEntry contains information about a service extracted from the keystone response.
type ServiceEntry struct {
	ID        string     `mapstructure:"id"`
	Name      string     `mapstructure:"name"`
	Type      string     `mapstructure:"type"`
	Endpoints []Endpoint `mapstructure:"endpoints"`
}

// Services is a list of ServiceEntry structs
// These structs contain information about the services keystone knows about.
type Services struct {
	Entries []ServiceEntry
}

type getResult struct {
	v3tokens.GetResult
}

// extractProject
// Ideally we would actually contribute this functionality
// back to the gophercloud project, but for now we extend
// their object to allow us to get project information out
// of the response from the GET token validation request.
func (r getResult) extractProject() (*Project, error) {
	if r.Err != nil {
		glog.V(2).Info(r.Err)
		return nil, r.Err
	}

	// can there be more than one project?  You need to test.
	var response struct {
		Token struct {
			ValidProject Project `mapstructure:"project"`
		} `mapstructure:"token"`
	}

	err := mapstructure.Decode(r.Body, &response)
	if err != nil {
		glog.V(2).Info(err)
		return nil, err
	}

	return &Project{
		ID:   response.Token.ValidProject.ID,
		Name: response.Token.ValidProject.Name,
	}, nil
}

func (r getResult) extractServices() (*Services, error) {
	if r.Err != nil {
		glog.V(2).Info(r.Err)
		return nil, r.Err
	}

	var response struct {
		Token struct {
			Entries []ServiceEntry `mapstructure:"catalog"`
		} `mapstructure:"token"`
	}

	err := mapstructure.Decode(r.Body, &response)
	if err != nil {
		glog.Errorf(err.Error())
		return nil, err
	}

	return &Services{Entries: response.Token.Entries}, nil
}

// extractRole
// Ideally we would actually contribute this functionality
// back to the gophercloud project, but for now we extend
// their object to allow us to get project information out
// of the response from the GET token validation request.
func (r getResult) extractRoles() (*Roles, error) {
	if r.Err != nil {
		glog.V(2).Info(r.Err)
		return nil, r.Err
	}

	var response struct {
		Token struct {
			ValidRoles []RoleEntry `mapstructure:"roles"`
		} `mapstructure:"token"`
	}

	err := mapstructure.Decode(r.Body, &response)
	if err != nil {
		glog.V(2).Info(err)
		return nil, err
	}

	return &Roles{Entries: response.Token.ValidRoles}, nil
}

// validateServices
// Validates that a given user belonging to a tenant
// can access a service specified by its type and name.
func validateService(tokenResult getResult, tenantID string, serviceType string, serviceName string) bool {

	p, err := tokenResult.extractProject()
	if err != nil {
		return false
	}

	if p.ID != tenantID {
		glog.Errorf("expected %s got %s\n", tenantID, p.ID)
		return false
	}

	services, err := tokenResult.extractServices()
	if err != nil {
		return false
	}

	for _, e := range services.Entries {
		if e.Type == serviceType {
			if serviceName == "" {
				return true
			}

			if e.Name == serviceName {
				return true
			}
		}
	}

	return false
}

func validateProjectRole(tokenResult getResult, project string, role string) bool {
	p, err := tokenResult.extractProject()
	if err != nil {
		return false
	}

	if project != "" && p.Name != project {
		return false
	}

	roles, err := tokenResult.extractRoles()
	if err != nil {
		return false
	}

	for i := range roles.Entries {
		if roles.Entries[i].Name == role {
			return true
		}
	}
	return false
}

// checkToken verifies that given the token, the request is performed as
// a valid admin and that such token is consistent with the services
// attempted to be used in the received request
func (h Handler) checkToken(r *http.Request, tenant string, tokenGetResult getResult) bool {

	/* TODO Caching or PKI */
	for _, a := range h.ValidAdmins {
		if validateProjectRole(tokenGetResult, a.Project, a.Role) == true {
			return true
		}
	}

	for _, s := range h.ValidServices {
		if validateService(tokenGetResult, tenant, s.ServiceType, s.ServiceName) == true {
			return true
		} else if validateService(tokenGetResult, tenant, s.ServiceType, "") == true {
			return true
		}

	}

	glog.V(2).Infof("Invalid token for [%s]", tenant)
	return false
}

func (h Handler) validateToken(r *http.Request) bool {

	token := r.Header["X-Auth-Token"]
	if len(token) == 0 {
		return false
	}

	vars := mux.Vars(r)
	tenantFromVars := vars["tenant"]

	// tenant may not exists on vars due to API endpoints URI lacking of
	// tenantID (such as the image service), for this case we need to
	// retrieve tenant from the given X-Auth-Token.
	res := v3tokens.Get(h.Client, token[0])
	tokenResult := getResult{res}
	p, err := tokenResult.extractProject()
	if err != nil {
		glog.V(2).Infof("Unable to retrieve tenant from token [%s]", token)
		return false
	}
	tenantFromToken := p.ID

	if tenantFromVars == "" {
		glog.V(2).Infof("Token validation for [%s]", tenantFromToken)
		return h.checkToken(r, tenantFromToken, tokenResult)
	}
	// verify that tenant from token is consistent with the tenant
	// obtained from the URI endpoint request
	if tenantFromVars != tenantFromToken {
		glog.Errorf("expected tenant %v, got %v\n", tenantFromToken, tenantFromVars)
		return false
	}

	glog.V(2).Infof("Token validation for [%s]", tenantFromVars)
	return h.checkToken(r, tenantFromVars, tokenResult)
}

// ValidService defines service name and type of the api service
type ValidService struct {
	ServiceType string
	ServiceName string
}

// ValidAdmin defines which project and roles are considered valid for admin.
type ValidAdmin struct {
	Project string
	Role    string
}

// Handler is a custom handler for APIs which would like keystone validation.
// This custom handler allows us to more cleanly return an error and response,
// and pass some package level context into the handler.
type Handler struct {
	Client        *gophercloud.ServiceClient
	Next          http.Handler
	ValidServices []ValidService
	ValidAdmins   []ValidAdmin
}

// ServeHTTP satisfies the http handler interface.
// It will check to make sure that the api caller is validated with
// keystone before allowing the next handler in the chain to be called.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.validateToken(r) == false {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	h.Next.ServeHTTP(w, r)
}
