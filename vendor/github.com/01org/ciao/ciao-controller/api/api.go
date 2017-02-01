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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/ssntp/uuid"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

// Port is the default port number for the ciao API.
const Port = 8889

const (
	// PoolsV1 is the content-type string for v1 of our pools resource
	PoolsV1 = "x.ciao.pools.v1"

	// ExternalIPsV1 is the content-type string for v1 of our external-ips resource
	ExternalIPsV1 = "x.ciao.external-ips.v1"

	// WorkloadsV1 is the content-type string for v1 of our workloads resource
	WorkloadsV1 = "x.ciao.workloads.v1"
)

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

// Response contains the http status and any response struct to be marshalled.
type Response struct {
	status   int
	response interface{}
}

func errorResponse(err error) Response {
	switch err {
	case types.ErrPoolNotFound,
		types.ErrTenantNotFound,
		types.ErrAddressNotFound,
		types.ErrInstanceNotFound:
		return Response{http.StatusNotFound, nil}

	case types.ErrQuota,
		types.ErrInstanceNotAssigned,
		types.ErrDuplicateSubnet,
		types.ErrDuplicateIP,
		types.ErrInvalidIP,
		types.ErrPoolNotEmpty,
		types.ErrInvalidPoolAddress,
		types.ErrBadRequest,
		types.ErrPoolEmpty,
		types.ErrDuplicatePoolName:
		return Response{http.StatusForbidden, nil}

	default:
		return Response{http.StatusInternalServerError, nil}
	}
}

// Handler is a custom handler for the compute APIs.
// This custom handler allows us to more cleanly return an error and response,
// and pass some package level context into the handler.
type Handler struct {
	*Context
	Handler func(*Context, http.ResponseWriter, *http.Request) (Response, error)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// set the content type to whatever was requested.
	contentType := r.Header.Get("Content-Type")

	resp, err := h.Handler(h.Context, w, r)
	if err != nil {
		data := HTTPErrorData{
			Code:    resp.status,
			Name:    http.StatusText(resp.status),
			Message: err.Error(),
		}

		code := HTTPReturnErrorCode{
			Error: data,
		}

		b, err := json.Marshal(code)
		if err != nil {
			http.Error(w, http.StatusText(resp.status), resp.status)
		}

		http.Error(w, string(b), resp.status)
	}

	b, err := json.Marshal(resp.response)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(resp.status)
	w.Write(b)
}

func listResources(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	var links []types.APILink
	vars := mux.Vars(r)
	tenantID, ok := vars["tenant"]

	// we support the "pools" resource.
	link := types.APILink{
		Rel:        "pools",
		Version:    PoolsV1,
		MinVersion: PoolsV1,
	}

	if !ok {
		link.Href = fmt.Sprintf("%s/pools", c.URL)
	} else {
		link.Href = fmt.Sprintf("%s/%s/pools", c.URL, tenantID)
	}

	links = append(links, link)

	// we support the "external-ips" resource
	link = types.APILink{
		Rel:        "external-ips",
		Version:    ExternalIPsV1,
		MinVersion: ExternalIPsV1,
	}

	if !ok {
		link.Href = fmt.Sprintf("%s/external-ips", c.URL)
	} else {
		link.Href = fmt.Sprintf("%s/%s/external-ips", c.URL, tenantID)
	}

	links = append(links, link)

	// we support the "workloads" resource
	link = types.APILink{
		Rel:        "workloads",
		Version:    WorkloadsV1,
		MinVersion: WorkloadsV1,
	}

	if !ok {
		link.Href = fmt.Sprintf("%s/workloads", c.URL)
	} else {
		link.Href = fmt.Sprintf("%s/%s/workloads", c.URL, tenantID)
	}

	links = append(links, link)

	return Response{http.StatusOK, links}, nil
}

func dumpPool(pool types.Pool) {
	glog.V(2).Info("Pool")
	glog.V(2).Info("-----------------------")
	glog.V(2).Infof("ID: %s\n", pool.ID)
	glog.V(2).Infof("Name: %s\n", pool.Name)
	glog.V(2).Infof("TotalIPs: %d\n", pool.TotalIPs)
	glog.V(2).Infof("Free: %d\n", pool.Free)
	glog.V(2).Infof("Links: %v\n", pool.Links)
	glog.V(2).Infof("Subnets:\n")

	for _, sub := range pool.Subnets {
		glog.V(2).Infof("%v", sub)
	}
}

func showPool(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	ID := vars["pool"]

	pool, err := c.ShowPool(ID)
	if err != nil {
		return errorResponse(err), err
	}

	dumpPool(pool)

	return Response{http.StatusOK, pool}, nil
}

func listPools(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	var resp types.ListPoolsResponse
	vars := mux.Vars(r)
	_, ok := vars["tenant"]

	pools, err := c.ListPools()
	if err != nil {
		return errorResponse(err), err
	}

	queries := r.URL.Query()

	names, returnNamedPool := queries["name"]

	for i, p := range pools {
		dumpPool(p)

		var match bool

		if returnNamedPool == true {
			for _, name := range names {
				if name == p.Name {
					match = true
				}
			}
		} else {
			match = true
		}

		if match {
			summary := types.PoolSummary{
				ID:   p.ID,
				Name: p.Name,
			}

			if !ok {
				summary.TotalIPs = &pools[i].TotalIPs
				summary.Free = &pools[i].Free
				summary.Links = pools[i].Links
			}

			resp.Pools = append(resp.Pools, summary)
		}
	}

	return Response{http.StatusOK, resp}, err
}

func addPool(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	var req types.NewPoolRequest

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errorResponse(err), err
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		return errorResponse(err), err
	}

	var ips []string

	for _, ip := range req.IPs {
		ips = append(ips, ip.IP)
	}

	_, err = c.AddPool(req.Name, req.Subnet, ips)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func deletePool(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	ID := vars["pool"]

	err := c.DeletePool(ID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func addToPool(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	ID := vars["pool"]

	var req types.NewAddressRequest

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errorResponse(err), err
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		return errorResponse(err), err
	}

	var ips []string

	for _, ip := range req.IPs {
		ips = append(ips, ip.IP)
	}

	err = c.AddAddress(ID, req.Subnet, ips)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func deleteSubnet(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	poolID := vars["pool"]
	subnetID := vars["subnet"]

	err := c.RemoveAddress(poolID, &subnetID, nil)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func deleteExternalIP(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	poolID := vars["pool"]
	IPID := vars["ip_id"]

	err := c.RemoveAddress(poolID, nil, &IPID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func listMappedIPs(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	tenantID, ok := vars["tenant"]
	var IPs []types.MappedIP
	var short []types.MappedIPShort

	if !ok {
		IPs = c.ListMappedAddresses(nil)
		return Response{http.StatusOK, IPs}, nil
	}

	IPs = c.ListMappedAddresses(&tenantID)
	for _, IP := range IPs {
		s := types.MappedIPShort{
			ID:         IP.ID,
			ExternalIP: IP.ExternalIP,
			InternalIP: IP.InternalIP,
			Links:      IP.Links,
		}
		short = append(short, s)
	}

	return Response{http.StatusOK, short}, nil
}

func mapExternalIP(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	var req types.MapIPRequest

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errorResponse(err), err
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		return errorResponse(err), err
	}

	err = c.MapAddress(req.PoolName, req.InstanceID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func unmapExternalIP(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	tenantID, ok := vars["tenant"]
	mappingID := vars["mapping_id"]

	var IPs []types.MappedIP

	if !ok {
		IPs = c.ListMappedAddresses(nil)
	} else {
		IPs = c.ListMappedAddresses(&tenantID)
	}

	for _, m := range IPs {
		if m.ID == mappingID {
			err := c.UnMapAddress(m.ExternalIP)
			if err != nil {
				return errorResponse(err), err
			}

			return Response{http.StatusAccepted, nil}, nil
		}
	}

	return errorResponse(types.ErrAddressNotFound), types.ErrAddressNotFound
}

func addWorkload(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	var req types.Workload

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errorResponse(err), err
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		return errorResponse(err), err
	}

	wl, err := c.CreateWorkload(req)
	if err != nil {
		return errorResponse(err), err
	}

	ref := fmt.Sprintf("%s/workloads/%s", c.URL, wl.ID)

	link := types.Link{
		Rel:  "self",
		Href: ref,
	}

	resp := types.WorkloadResponse{
		Workload: wl,
		Link:     link,
	}

	return Response{http.StatusCreated, resp}, nil
}

// Service is an interface which must be implemented by the ciao API context.
type Service interface {
	AddPool(name string, subnet *string, ips []string) (types.Pool, error)
	ListPools() ([]types.Pool, error)
	ShowPool(id string) (types.Pool, error)
	DeletePool(id string) error
	AddAddress(poolID string, subnet *string, IPs []string) error
	RemoveAddress(poolID string, subnetID *string, IPID *string) error
	ListMappedAddresses(tenantID *string) []types.MappedIP
	MapAddress(poolName *string, instanceID string) error
	UnMapAddress(ID string) error
	CreateWorkload(req types.Workload) (types.Workload, error)
}

// Context is used to provide the services and current URL to the handlers.
type Context struct {
	URL string
	Service
}

// Config is used to setup the Context for the ciao API.
type Config struct {
	URL         string
	CiaoService Service
}

// Routes returns the supported ciao API endpoints.
// A plain application/json request will return v1 of the resource
// since we only have one version of this api so far, that means
// most routes will match both json as well as our custom
// content type.
func Routes(config Config) *mux.Router {
	// make new Context
	context := &Context{config.URL, config.CiaoService}

	r := mux.NewRouter()

	// external IP pools
	route := r.Handle("/", Handler{context, listResources})
	route.Methods("GET")

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}", Handler{context, listResources})
	route.Methods("GET")

	matchContent := fmt.Sprintf("application/(%s|json)", PoolsV1)

	route = r.Handle("/pools", Handler{context, listPools})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}", Handler{context, listPools})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/pools", Handler{context, addPool})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/pools/{pool:"+uuid.UUIDRegex+"}", Handler{context, showPool})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/pools/{pool:"+uuid.UUIDRegex+"}", Handler{context, deletePool})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/pools/{pool:"+uuid.UUIDRegex+"}", Handler{context, addToPool})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/pools/{pool:"+uuid.UUIDRegex+"}/subnets/{subnet:"+uuid.UUIDRegex+"}", Handler{context, deleteSubnet})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/pools/{pool:"+uuid.UUIDRegex+"}/external-ips/{ip_id:"+uuid.UUIDRegex+"}", Handler{context, deleteExternalIP})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	// mapped external IPs
	matchContent = fmt.Sprintf("application/(%s|json)", ExternalIPsV1)

	route = r.Handle("/external-ips", Handler{context, listMappedIPs})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}/external-ips", Handler{context, listMappedIPs})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/external-ips", Handler{context, mapExternalIP})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}/external-ips", Handler{context, mapExternalIP})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/external-ips/{mapping_id:"+uuid.UUIDRegex+"}", Handler{context, unmapExternalIP})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}/external-ips/{mapping_id:"+uuid.UUIDRegex+"}", Handler{context, unmapExternalIP})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	// workloads
	matchContent = fmt.Sprintf("application/(%s|json)", WorkloadsV1)

	route = r.Handle("/workloads", Handler{context, addWorkload})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)
	return r
}
