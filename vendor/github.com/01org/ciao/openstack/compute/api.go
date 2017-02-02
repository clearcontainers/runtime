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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

// APIPort is the OpenStack compute port
const APIPort = 8774

// PrivateAddresses contains information about a single instance network
// interface.
type PrivateAddresses struct {
	Addr               string `json:"addr"`
	OSEXTIPSMACMacAddr string `json:"OS-EXT-IPS-MAC:mac_addr"`
	OSEXTIPSType       string `json:"OS-EXT-IPS:type"`
	Version            int    `json:"version"`
}

// Addresses contains information about an instance's networks.
type Addresses struct {
	Private []PrivateAddresses `json:"private"`
}

// Link contains the address to a compute resource, like e.g. a Flavor or an
// Image.
type Link struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

// FlavorLinks provides links to a specific flavor ID.
type FlavorLinks struct {
	ID    string `json:"id"`
	Links []Link `json:"links"`
}

// Image identifies the base image of the instance.
type Image struct {
	ID    string `json:"id"`
	Links []Link `json:"links"`
}

// SecurityGroup represents the security group of an instance.
type SecurityGroup struct {
	Name string `json:"name"`
}

// These errors can be returned by the Service interface
var (
	ErrQuota                = errors.New("Tenant over quota")
	ErrTenantNotFound       = errors.New("Tenant not found")
	ErrServerNotFound       = errors.New("Server not found")
	ErrServerOwner          = errors.New("You are not server owner")
	ErrInstanceNotAvailable = errors.New("Instance not currently available for this operation")
)

// errorResponse maps service error responses to http responses.
// this helper function can help functions avoid having to switch
// on return values all the time.
func errorResponse(err error) APIResponse {
	switch err {
	case ErrTenantNotFound, ErrServerNotFound:
		return APIResponse{http.StatusNotFound, nil}

	case ErrQuota, ErrServerOwner, ErrInstanceNotAvailable:
		return APIResponse{http.StatusForbidden, nil}

	default:
		return APIResponse{http.StatusInternalServerError, nil}
	}
}

// ServerDetails contains information about a specific instance.
type ServerDetails struct {
	Addresses                        Addresses       `json:"addresses"`
	Created                          time.Time       `json:"created"`
	Flavor                           FlavorLinks     `json:"flavor"`
	HostID                           string          `json:"hostId"`
	ID                               string          `json:"id"`
	Image                            Image           `json:"image"`
	KeyName                          string          `json:"key_name"`
	Links                            []Link          `json:"links"`
	Name                             string          `json:"name"`
	AccessIPv4                       string          `json:"accessIPv4"`
	AccessIPv6                       string          `json:"accessIPv6"`
	ConfigDrive                      string          `json:"config_drive"`
	OSDCFDiskConfig                  string          `json:"OS-DCF:diskConfig"`
	OSEXTAZAvailabilityZone          string          `json:"OS-EXT-AZ:availability_zone"`
	OSEXTSRVATTRHost                 string          `json:"OS-EXT-SRV-ATTR:host"`
	OSEXTSRVATTRHypervisorHostname   string          `json:"OS-EXT-SRV-ATTR:hypervisor_hostname"`
	OSEXTSRVATTRInstanceName         string          `json:"OS-EXT-SRV-ATTR:instance_name"`
	OSEXTSTSPowerState               int             `json:"OS-EXT-STS:power_state"`
	OSEXTSTSTaskState                string          `json:"OS-EXT-STS:task_state"`
	OSEXTSTSVMState                  string          `json:"OS-EXT-STS:vm_state"`
	OsExtendedVolumesVolumesAttached []string        `json:"os-extended-volumes:volumes_attached"`
	OSSRVUSGLaunchedAt               time.Time       `json:"OS-SRV-USG:launched_at"`
	OSSRVUSGTerminatedAt             time.Time       `json:"OS-SRV-USG:terminated_at"`
	Progress                         int             `json:"progress"`
	SecurityGroups                   []SecurityGroup `json:"security_groups"`
	Status                           string          `json:"status"`
	HostStatus                       string          `json:"host_status"`
	TenantID                         string          `json:"tenant_id"`
	Updated                          time.Time       `json:"updated"`
	UserID                           string          `json:"user_id"`
	SSHIP                            string          `json:"ssh_ip"`
	SSHPort                          int             `json:"ssh_port"`
}

// Servers represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/servers/detail response.  It contains information about a
// set of instances within a ciao cluster.
// http://developer.openstack.org/api-ref/compute/?expanded=list-servers-detailed-detail
// BUG - TotalServers is not specified by the openstack api. We are going
// to pretend it is for now.
type Servers struct {
	TotalServers int             `json:"total_servers"`
	Servers      []ServerDetails `json:"servers"`
}

// NewServers allocates a Servers structure.
// It allocates the Servers slice as well so that the marshalled
// JSON is an empty array and not a nil pointer for, as
// specified by the OpenStack APIs.
func NewServers() (servers Servers) {
	servers.Servers = []ServerDetails{}
	return
}

// Server represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/servers/{server} response.  It contains information about a
// specific instance within a ciao cluster.
type Server struct {
	Server ServerDetails `json:"server"`
}

// Flavors represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/flavors response.  It contains information about all the
// flavors in a cluster.
type Flavors struct {
	Flavors []struct {
		ID    string `json:"id"`
		Links []Link `json:"links"`
		Name  string `json:"name"`
	} `json:"flavors"`
}

// NewComputeFlavors allocates a ComputeFlavors structure.
// It allocates the Flavors slice as well so that the marshalled
// JSON is an empty array and not a nil pointer, as specified
// by the OpenStack APIs.
func NewComputeFlavors() (flavors Flavors) {
	flavors.Flavors = []struct {
		ID    string `json:"id"`
		Links []Link `json:"links"`
		Name  string `json:"name"`
	}{}
	return
}

// FlavorDetails contains information about a specific flavor.
type FlavorDetails struct {
	OSFLVDISABLEDDisabled  bool   `json:"OS-FLV-DISABLED:disabled"`
	Disk                   string `json:"disk"` /* OpenStack API says this is an int */
	OSFLVEXTDATAEphemeral  int    `json:"OS-FLV-EXT-DATA:ephemeral"`
	OsFlavorAccessIsPublic bool   `json:"os-flavor-access:is_public"`
	ID                     string `json:"id"`
	Links                  []Link `json:"links"`
	Name                   string `json:"name"`
	RAM                    int    `json:"ram"`
	Swap                   string `json:"swap"`
	Vcpus                  int    `json:"vcpus"`
}

// BlockDeviceMappingV2 represents an optional block_device_mapping_v2
// object within a /v2.1/{tenant}/servers request POST to "Create Server"
// array of block_device_mapping_v2 objects.
// NOTE: the OpenStack api-ref currently indicates in text that this is an
// object not an array, but given the implementation/usage it is clearly in
// fact an array.  Also volume size and uuid are not documented in the API
// reference, but logically must be included.
type BlockDeviceMappingV2 struct {
	// DeviceName: the name the hypervisor should assign to the block
	// device, eg: "vda"
	DeviceName string `json:"device_name,omitempty"`

	// SourceType: blank, snapshot, volume, or image
	SourceType string `json:"source_type"`

	// DestinationType: optional flag to indicate whether the block
	// device is backed locally or from the volume service
	DestinationType string `json:"destination_type,omitempty"`

	// DeleteOnTermination: optional flag to indicate the volume should
	// autodelete upon termination of the instance
	DeleteOnTermination bool `json:"delete_on_termination,omitempty"`

	// GuestFormat: optionally format a created volume as "swap" or
	// leave "ephemeral" (unformatted) for any use by the instance
	GuestFormat string `json:"guest_format,omitempty"`

	// BootIndex: hint to hypervisor for boot order among multiple
	// bootable devices, eg: floppy, cdrom, disk.  Default "none".
	// Disable booting via negative number or "none"
	BootIndex string `json:"boot_index"`

	// Tag: optional arbitrary text identifier for the block device, useful
	// for human identification or programmatic searching/sorting
	Tag string `json:"tag,omitempty"`

	// UUID: the volume/image/snapshot to attach
	UUID string `json:"uuid,omitempty"`

	// VolumeSize: integer number of gigabytes for ephemeral or swap
	VolumeSize int `json:"volume_size,omitempty"`
}

// Flavor represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/flavors/{flavor} response.  It contains information about a
// specific flavour.
type Flavor struct {
	Flavor FlavorDetails `json:"flavor"`
}

// FlavorsDetails represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/flavors/detail response. It contains detailed information about
// all flavour for a given tenant.
type FlavorsDetails struct {
	Flavors []FlavorDetails `json:"flavors"`
}

// NewComputeFlavorsDetails allocates a ComputeFlavorsDetails structure.
// It allocates the Flavors slice as well so that the marshalled
// JSON is an empty array and not a nil pointer, as specified by the
// OpenStack APIs.
func NewComputeFlavorsDetails() (flavors FlavorsDetails) {
	flavors.Flavors = []FlavorDetails{}
	return
}

// CreateServerRequest represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/servers request.  It contains the information needed to start
// one or more instances.
type CreateServerRequest struct {
	Server struct {
		ID                  string                 `json:"id"`
		Name                string                 `json:"name"`
		Image               string                 `json:"imageRef"`
		Flavor              string                 `json:"flavorRef"`
		MaxInstances        int                    `json:"max_count"`
		MinInstances        int                    `json:"min_count"`
		BlockDeviceMappings []BlockDeviceMappingV2 `json:"block_device_mapping_v2,omitempty"`
	} `json:"server"`
}

// APIConfig contains information needed to start the compute api service.
type APIConfig struct {
	Port           int     // the https port of the compute api service
	ComputeService Service // the service interface
}

// Service defines the interface required by the compute service.
type Service interface {
	// server interfaces
	CreateServer(string, CreateServerRequest) (interface{}, error)
	ListServersDetail(tenant string) ([]ServerDetails, error)
	ShowServerDetails(tenant string, server string) (Server, error)
	DeleteServer(tenant string, server string) error
	StartServer(tenant string, server string) error
	StopServer(tenant string, server string) error

	//flavor interfaces
	ListFlavors(string) (Flavors, error)
	ListFlavorsDetail(string) (FlavorsDetails, error)
	ShowFlavorDetails(string, string) (Flavor, error)
}

type pagerFilterType uint8

const (
	none pagerFilterType = iota
	changesSinceFilter
	imageFilter
	flavorFilter
	nameFilter
	statusFilter
	hostFilter
	limit
	marker
)

type pager interface {
	filter(filterType pagerFilterType, filter string, item interface{}) bool
	nextPage(filterType pagerFilterType, filter string, r *http.Request) ([]byte, error)
}

type serverPager struct {
	servers []ServerDetails
}

func pagerQueryParse(r *http.Request) (int, int, string) {
	values := r.URL.Query()
	limit := 0
	offset := 0
	marker := ""

	// we only support marker and offset for now.
	if values["marker"] != nil {
		marker = values["marker"][0]
	} else {
		if values["offset"] != nil {
			o, err := strconv.ParseInt(values["offset"][0], 10, 32)
			if err != nil {
				offset = 0
			} else {
				offset = (int)(o)
			}
		}
		if values["limit"] != nil {
			l, err := strconv.ParseInt(values["limit"][0], 10, 32)
			if err != nil {
				limit = 0
			} else {
				limit = (int)(l)
			}
		}
	}

	return limit, offset, marker
}

func (pager *serverPager) getServers(filterType pagerFilterType, filter string, servers []ServerDetails, limit int, offset int) (Servers, error) {
	newServers := NewServers()

	newServers.TotalServers = len(servers)
	pageLength := 0

	glog.V(2).Infof("Get servers limit [%d] offset [%d]", limit, offset)

	if servers == nil || offset >= len(servers) {
		return newServers, nil
	}

	for _, server := range servers[offset:] {
		if filterType != none &&
			pager.filter(filterType, filter, server) {
			continue
		}

		newServers.Servers = append(newServers.Servers, server)
		pageLength++
		if limit > 0 && pageLength >= limit {
			break
		}
	}

	return newServers, nil
}

func (pager *serverPager) filter(filterType pagerFilterType, filter string, server ServerDetails) bool {
	// we only support filtering by flavor right now
	switch filterType {
	case flavorFilter:
		if server.Flavor.ID != filter {
			return true
		}
	}

	return false
}

func (pager *serverPager) nextPage(filterType pagerFilterType, filter string, r *http.Request) (Servers, error) {
	limit, offset, lastSeen := pagerQueryParse(r)

	glog.V(2).Infof("Next page marker [%s] limit [%d] offset [%d]",
		lastSeen, limit, offset)

	if lastSeen == "" {
		if limit != 0 {
			return pager.getServers(filterType, filter,
				pager.servers, limit, offset)
		}

		return pager.getServers(filterType, filter, pager.servers,
			0, offset)
	}

	for i, server := range pager.servers {
		if server.ID == lastSeen {
			if i >= len(pager.servers)-1 {
				return pager.getServers(filterType, filter,
					nil, limit, 0)
			}

			return pager.getServers(filterType, filter,
				pager.servers[i+1:], limit, 0)
		}
	}

	return Servers{}, fmt.Errorf("Item %s not found", lastSeen)
}

type action uint8

const (
	computeActionStart action = iota
	computeActionStop
	computeActionDelete
)

func dumpRequestBody(r *http.Request, body bool) {
	if glog.V(2) {
		dump, err := httputil.DumpRequest(r, body)
		if err != nil {
			glog.Errorf("HTTP request dump error %s", err)
		}

		glog.Infof("HTTP request [%q]", dump)
	}
}

// DumpRequest will dump an http request if log level is 2
func DumpRequest(r *http.Request) {
	dumpRequestBody(r, false)
}

// HTTPErrorData represents the HTTP response body for
// a compute API request error.
type HTTPErrorData struct {
	Code    int    `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// HTTPReturnErrorCode represents the unmarshalled version for Return codes
// when a API call is made and you need to return explicit data of
// the call as OpenStack format
// http://developer.openstack.org/api-guide/compute/faults.html
type HTTPReturnErrorCode struct {
	Error HTTPErrorData `json:"error"`
}

// Context contains information needed by the compute API service
type Context struct {
	port int
	Service
}

// APIResponse is returned from all compute API functions.
// It contains the http status and response to be marshalled if needed.
type APIResponse struct {
	Status   int
	Response interface{}
}

// APIHandler is a custom handler for the compute APIs.
// This custom handler allows us to more cleanly return an error and response,
// and pass some package level context into the handler.
type APIHandler struct {
	*Context
	Handler func(*Context, http.ResponseWriter, *http.Request) (APIResponse, error)
}

// ServeHTTP satisfies the interface for the http Handler.
// If the individual handler returns an error, then it will marshal
// an error response.
func (h APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Handler(h.Context, w, r)
	if err != nil {
		data := HTTPErrorData{
			Code:    resp.Status,
			Name:    http.StatusText(resp.Status),
			Message: err.Error(),
		}

		code := HTTPReturnErrorCode{
			Error: data,
		}

		b, err := json.Marshal(code)
		if err != nil {
			http.Error(w, http.StatusText(resp.Status), resp.Status)
		}

		http.Error(w, string(b), resp.Status)
	}

	b, err := json.Marshal(resp.Response)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Status)
	w.Write(b)
}

// @Title createServer
// @Description Creates a server.
// @Accept  json
// @Success 202 {object} Servers "Returns Servers and CreateServerRequest with data of the created server."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/{tenant}/servers [post]
// @Resource /v2.1/{tenant}/servers
func createServer(c *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {

	vars := mux.Vars(r)
	tenant := vars["tenant"]

	DumpRequest(r)

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return APIResponse{http.StatusBadRequest, nil}, err
	}

	var req CreateServerRequest

	err = json.Unmarshal(body, &req)
	if err != nil {
		return APIResponse{http.StatusBadRequest, nil}, err
	}

	resp, err := c.CreateServer(tenant, req)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusAccepted, resp}, nil
}

// ListServersDetails provides server details by tenant or by flavor.
// This function is exported for use by ciao-controller due to legacy
// endpoint using the "flavor" option. It is simpler to just overload
// this function than to reimplement the legacy code.
//
// @Title ListServerDetails
// @Description Lists all servers with details.
// @Accept  json
// @Success 200 {array} ServerDetails "Returns details of all servers."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/{tenant}/servers/detail [get]
// @Resource /v2.1/{tenant}/servers
func ListServersDetails(c *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]

	// flavor will never have a valid value. This is left over from
	// when this function could be used by a different endpoint.
	// the function needs to be rewritten with no pager.
	flavor := vars["flavor"]

	DumpRequest(r)

	servers, err := c.ListServersDetail(tenant)
	if err != nil {
		return errorResponse(err), err
	}

	pager := serverPager{servers: servers}
	filterType := none
	filter := ""
	if flavor != "" {
		filterType = flavorFilter
		filter = flavor
	}

	resp, err := pager.nextPage(filterType, filter, r)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusOK, resp}, nil
}

// @Title showServerDetails
// @Description Shows details for a server.
// @Accept  json
// @Success 200 {object} Server "Returns details for a server."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/{tenant}/servers/{server} [get]
// @Resource /v2.1/{tenant}/servers
func showServerDetails(c *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	server := vars["server"]

	DumpRequest(r)

	resp, err := c.ShowServerDetails(tenant, server)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusOK, resp}, nil
}

// @Title deleteServer
// @Description Deletes a server.
// @Accept  json
// @Success 204 {object} string "This operation does not return a response body, returns the 204 StatusNoContent code."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/{tenant}/servers/{server} [delete]
// @Resource /v2.1/{tenant}/servers
func deleteServer(c *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	server := vars["server"]

	DumpRequest(r)

	err := c.DeleteServer(tenant, server)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusNoContent, nil}, nil
}

// @Title serverAction
// @Description Runs the indicated action (os-start, os-stop) in the a server.
// @Accept  json
// @Success 202 {object} string "This operation does not return a response body, returns the 202 StatusAccepted code."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/{tenant}/servers/{server}/action [post]
// @Resource /v2.1/{tenant}/servers
func serverAction(c *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	server := vars["server"]

	DumpRequest(r)

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return APIResponse{http.StatusBadRequest, nil}, err
	}

	bodyString := string(body)

	var action action

	if strings.Contains(bodyString, "os-start") {
		action = computeActionStart
	} else if strings.Contains(bodyString, "os-stop") {
		action = computeActionStop
	} else {
		return APIResponse{http.StatusServiceUnavailable, nil},
			errors.New("Unsupported Action")
	}

	switch action {
	case computeActionStart:
		err = c.StartServer(tenant, server)
	case computeActionStop:
		err = c.StopServer(tenant, server)
	}

	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusAccepted, nil}, nil
}

// @Title listFlavors
// @Description Lists flavors.
// @Accept  json
// @Success 200 {object} Flavors "Returns Flavors with the corresponding available flavors for the tenant."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/{tenant}/flavors [get]
// @Resource /v2.1/{tenant}/flavors
func listFlavors(c *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]

	DumpRequest(r)

	resp, err := c.ListFlavors(tenant)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusOK, resp}, nil
}

// @Title listFlavorsDetails
// @Description Lists flavors with details.
// @Accept  json
// @Success 200 {object} FlavorsDetails "Returns FlavorsDetails with flavor details."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/{tenant}/flavors/detail [get]
// @Resource /v2.1/{tenant}/flavors
func listFlavorsDetails(c *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]

	DumpRequest(r)

	resp, err := c.ListFlavorsDetail(tenant)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusOK, resp}, nil
}

// @Title showFlavorDetails
// @Description Shows details for a flavor.
// @Accept  json
// @Success 200 {object} Flavor "Returns details for a flavor."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/{tenant}/flavors/{flavor} [get]
// @Resource /v2.1/{tenant}/flavors
func showFlavorDetails(c *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	flavor := vars["flavor"]

	DumpRequest(r)

	resp, err := c.ShowFlavorDetails(tenant, flavor)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusOK, resp}, nil
}

// Routes returns a gorilla mux router for the compute endpoints.
func Routes(config APIConfig) *mux.Router {
	context := &Context{config.Port, config.ComputeService}

	r := mux.NewRouter()

	// servers endpoints
	r.Handle("/v2.1/{tenant}/servers",
		APIHandler{context, createServer}).Methods("POST")
	r.Handle("/v2.1/{tenant}/servers/detail",
		APIHandler{context, ListServersDetails}).Methods("GET")
	r.Handle("/v2.1/{tenant}/servers/{server}",
		APIHandler{context, showServerDetails}).Methods("GET")
	r.Handle("/v2.1/{tenant}/servers/{server}",
		APIHandler{context, deleteServer}).Methods("DELETE")
	r.Handle("/v2.1/{tenant}/servers/{server}/action",
		APIHandler{context, serverAction}).Methods("POST")

	// flavor related endpoints
	r.Handle("/v2.1/{tenant}/flavors",
		APIHandler{context, listFlavors}).Methods("GET")
	r.Handle("/v2.1/{tenant}/flavors/detail",
		APIHandler{context, listFlavorsDetails}).Methods("GET")
	r.Handle("/v2.1/{tenant}/flavors/{flavor}",
		APIHandler{context, showFlavorDetails}).Methods("GET")

	return r
}
