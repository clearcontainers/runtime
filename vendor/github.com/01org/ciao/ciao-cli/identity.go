//
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
//

package main

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/identity/v3/tokens"
)

// Project represents a tenant UUID and friendly name.
type Project struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
}

type getResult struct {
	tokens.GetResult
}

// Domain is a collection of users, groups and projects.
// Here we only need to fetch a domain UUID and friendly name.
type Domain struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
}

// User represents an Openstack identity (e.g. Keystone) user.
// We fetch the user UUID, friendly name and the domain it belongs
// to.
type User struct {
	ValidDomain Domain `mapstructure:"domain"`
	ID          string `mapstructure:"id"`
	Name        string `mapstructure:"name"`
}

// UserProjects represents the list of projects a user has access to.
type UserProjects struct {
	Projects []struct {
		Description string `json:"description"`
		DomainID    string `json:"domain_id"`
		Enabled     bool   `json:"enabled"`
		ID          string `json:"id"`
		ParentID    string `json:"parent_id"`
		Links       struct {
			Self string `json:"self"`
		} `json:"links"`
		Name string `json:"name"`
	} `json:"projects"`

	Links struct {
		Self     string      `json:"self"`
		Previous interface{} `json:"previous"`
		Next     interface{} `json:"next"`
	} `json:"links"`
}

func (r getResult) extractUserID() (string, error) {
	if r.Err != nil {
		return "", r.Err
	}

	var response struct {
		Token struct {
			ValidUser User `mapstructure:"user"`
		} `mapstructure:"token"`
	}

	err := mapstructure.Decode(r.Body, &response)
	if err != nil {
		return "", err
	}

	return response.Token.ValidUser.ID, nil
}

func (r getResult) extractProject() (string, error) {
	if r.Err != nil {
		return "", r.Err
	}

	var response struct {
		Token struct {
			ValidProject Project `mapstructure:"project"`
		} `mapstructure:"token"`
	}

	err := mapstructure.Decode(r.Body, &response)
	if err != nil {
		return "", err
	}

	return response.Token.ValidProject.ID, nil
}

func getScopedToken(username string, password string, projectScope string) (string, string, string, error) {
	var scope *tokens.Scope

	opt := gophercloud.AuthOptions{
		IdentityEndpoint: *identityURL + "/v3/",
		Username:         username,
		Password:         password,
		DomainID:         "default",
		AllowReauth:      true,
	}

	provider, err := newAuthenticatedClient(opt)
	if err != nil {
		return "", "", "", errors.Wrap(err, "Failed to create an AuthenticatedClient")
	}

	client := openstack.NewIdentityV3(provider)
	if client == nil {
		return "", "", "", errors.Wrap(err, "something went wrong")
	}

	scope = nil
	if projectScope != "" {
		scope = &tokens.Scope{
			ProjectName: projectScope,
			DomainName:  "default",
		}
	}

	token, err := tokens.Create(client, opt, scope).Extract()
	if err != nil {
		return "", "", "", errors.Wrap(err, "Could not extract token")
	}

	r := tokens.Get(client, token.ID)
	result := getResult{r}
	tenantID, err := result.extractProject()
	if err != nil {
		return "", "", "", errors.Wrap(err, "Could not extract tenant ID")
	}

	userID, err := result.extractUserID()
	if err != nil {
		return "", "", "", errors.Wrap(err, "Could not extract user ID")
	}

	infof("Got token %s for tenant %s, user %s (%s, %s, %s)\n", token.ID, tenantID, userID, username, password, projectScope)

	return token.ID, tenantID, userID, nil
}

func getUnscopedToken(username string, password string) (string, string, string, error) {
	return getScopedToken(username, password, "")
}

func getUserProjects(username string, password string) ([]Project, error) {
	var projects UserProjects
	var userProjects []Project

	token, _, user, err := getUnscopedToken(username, password)
	if err != nil {
		return nil, err
	}

	identity := fmt.Sprintf("%s/v3/users/%s/projects", *identityURL, user)

	resp, err := sendHTTPRequestToken("GET", identity, nil, token, nil, nil)
	if err != nil {
		return nil, err
	}

	err = unmarshalHTTPResponse(resp, &projects)
	if err != nil {
		return nil, err
	}

	for _, project := range projects.Projects {
		newProject := Project{
			ID:   project.ID,
			Name: project.Name,
		}
		userProjects = append(userProjects, newProject)
	}

	return userProjects, nil
}

// IdentityProjects represents the list of all existing projects.
type IdentityProjects struct {
	Links struct {
		Next     interface{} `json:"next"`
		Previous interface{} `json:"previous"`
		Self     string      `json:"self"`
	} `json:"links"`

	Projects []struct {
		Description interface{} `json:"description"`
		DomainID    string      `json:"domain_id"`
		Enabled     bool        `json:"enabled"`
		ID          string      `json:"id"`
		Links       struct {
			Self string `json:"self"`
		} `json:"links"`
		Name     string      `json:"name"`
		ParentID interface{} `json:"parent_id"`
	} `json:"projects"`
}

func getAllProjects(username string, password string) (*IdentityProjects, error) {
	var projects IdentityProjects

	token, _, _, err := getUnscopedToken(username, password)
	if err != nil {
		return nil, err
	}

	identity := fmt.Sprintf("%s/v3/auth/projects", *identityURL)

	resp, err := sendHTTPRequestToken("GET", identity, nil, token, nil, nil)
	if err != nil {
		return nil, err
	}

	err = unmarshalHTTPResponse(resp, &projects)
	if err != nil {
		return nil, err
	}

	return &projects, nil
}

func getTenant(username string, password string, tenantID string) (string, string, error) {
	projects, err := getUserProjects(username, password)
	if err != nil {
		return "", "", err
	}

	if len(projects) <= 0 {
		return "", tenantID, fmt.Errorf("No tenant name for %s", username)
	}

	if tenantID == "" {
		if len(projects) == 1 {
			tenantName := projects[0].Name
			tenantID = projects[0].ID
			return tenantName, tenantID, nil
		}
		fmt.Printf("Available projects for %s:\n", *identityUser)
		for i, p := range projects {
			fmt.Printf("\t Project[%d]: %s (%s)\n", i+1, p.Name, p.ID)
		}
		return "", "", fmt.Errorf("Please specify a project to use with -tenant-name or -tenant-id")
	}

	for _, p := range projects {
		if p.ID == tenantID {
			return p.Name, tenantID, nil
		}
	}
	return "", tenantID, fmt.Errorf("No tenant name for %s", tenantID)
}
