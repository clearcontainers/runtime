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
// @SubApi Servers API [/v2.1/{tenant}/servers]
// @SubApi Resources API [/v2.1/{tenant}/resources]
// @SubApi Quotas API [/v2.1/{tenant}/quotas]
// @SubApi Events API [/v2.1/{tenant}/events]
// @SubApi Nodes API [/v2.1/nodes]
// @SubApi Tenants API [/v2.1/tenants]
// @SubApi CNCIs API [/v2.1/cncis]
// @SubApi Traces API [/v2.1/traces]

package main

import (
	"encoding/json"
	"net/http"

	"github.com/01org/ciao/openstack/compute"
	"github.com/01org/ciao/service"
	"github.com/gorilla/mux"
)

// APIHandler is a custom handler for the compute APIs.
// This custom handler allows us to more cleanly return an error and response,
// and pass some package level context into the handler.
type legacyAPIHandler struct {
	*controller
	Handler    func(*controller, http.ResponseWriter, *http.Request) (APIResponse, error)
	Privileged bool
}

func (h legacyAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check to see if we should send permission denied for this route.
	if h.Privileged {
		privileged := service.GetPrivilege(r.Context())
		if !privileged {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
	}

	resp, err := h.Handler(h.controller, w, r)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.status)
	w.Write(b)
}

// @Title listTenantQuotas
// @Description List the use of all resources used of a tenant from a start to end point of time.
// @Accept  json
// @Success 200 {object} types.CiaoTenantResources "Returns the limits and usage of resources of a tenant."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/{tenant}/quotas [get]
// @Resource /v2.1/{tenant}/quotas
func listTenantQuotas(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return getResources(c, w, r)
}

// @Title listTenantResources
// @Description List the use of all resources used of a tenant from a start to end point of time.
// @Accept  json
// @Success 200 {object} types.CiaoUsageHistory "Returns the usage of resouces."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/{tenant}/resources [get]
// @Resource /v2.1/{tenant}/resources
func listTenantResources(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return getUsage(c, w, r)
}

// @Title tenantServersAction
// @Description Runs the indicated action (os-start, os-stop, os-delete) in the servers.
// @Accept  json
// @Success 202 {object} string "This operation does not return a response body, returns the 202 StatusAccepted code."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/{tenant}/servers/action [post]
// @Resource /v2.1/{tenant}/servers
// tenantServersAction will apply the operation sent in POST (as os-start, os-stop, os-delete)
// to all servers of a tenant or if ServersID size is greater than zero it will be applied
// only to the subset provided that also belongs to the tenant
func tenantServersAction(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return serversAction(c, w, r)
}

// @Title legacyListTenants
// @Description List all tenants.
// @Accept  json
// @Success 200 {array} interface "Marshalled format of types.CiaoComputeTenants representing the list of all tentants."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/tenants [get]
// @Resource /v2.1/tenants
func legacyListTenants(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listTenants(c, w, r)
}

// @Title legacyListNodes
// @Description Returns a list of all nodes.
// @Accept  json
// @Success 200 {array} interface "Returns ciao-controller.nodePager with TotalInstances, TotalRunningInstances, TotalPendingInstances, TotalPausedInstances."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/nodes [get]
// @Resource /v2.1/nodes
func legacyListNodes(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listNodes(c, w, r)
}

// @Title legacyNodesSummary
// @Description A summary of all node stats.
// @Accept  json
// @Success 200 {object} interface "Returns types.CiaoClusterStatus with TotalNodesReady, TotalNodesFull, TotalNodesOffline and TotalNodesMaintenance."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/nodes/summary [get]
// @Resource /v2.1/nodes
func legacyNodesSummary(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return nodesSummary(c, w, r)
}

// @Title legacyListNodeServers
// @Description A list of servers by node id.
// @Accept  json
// @Success 200 {object} interface "Returns types.CiaoServersStats"
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/nodes/{node}/servers/detail [get]
// @Resource /v2.1/nodes
func legacyListNodeServers(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listNodeServers(c, w, r)
}

// @Title legacyListCNCIs
// @Description Lists all CNCI agents.
// @Accept  json
// @Success 200 {array} types.CiaoCNCIs "Returns all CNCI agents data as InstanceId, TenantID, IPv4 and subnets."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/cncis [get]
// @Resource /v2.1/cncis
func legacyListCNCIs(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listCNCIs(c, w, r)
}

// @Title legacyListCNCIDetails
// @Description List details of a CNCI agent.
// @Accept  json
// @Success 200 {array} types.CiaoCNCIs "Returns details of a CNCI agent as InstanceId, TenantID, IPv4 and subnets."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/cncis/{cnci}/detail [get]
// @Resource /v2.1/cncis
func legacyListCNCIDetails(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listCNCIDetails(c, w, r)
}

// @Title legacyListTraces
// @Description List all Traces.
// @Accept  json
// @Success 200 {array} types.CiaoTracesSummary "Returns a summary of each trace in the system."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/traces [get]
// @Resource /v2.1/traces
func legacyListTraces(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listTraces(c, w, r)
}

// @Title legacyListEvents
// @Description List all Events.
// @Accept  json
// @Success 200 {array} types.CiaoEvent "Returns all events from the log system."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/events [get]
// @Resource /v2.1/events
func legacyListEvents(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listEvents(c, w, r)
}

// @Title legacyListTenantEvents
// @Description List Events.
// @Accept  json
// @Success 200 {array} types.CiaoEvent "Returns the events of a tenant from the log system."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/{tenant}/events [get]
// @Resource /v2.1/events
// listTenantEvents is created with the only purpose of API documentation for method
// /v2.1/{tenant}/events
func legacyListTenantEvents(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listEvents(c, w, r)
}

// @Title legacyClearEvents
// @Description Clear Events Log.
// @Accept  json
// @Success 202 {object} string "This operation does not return a response body, returns the 202 StatusAccepted code."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/events [delete]
// @Resource /v2.1/events
func legacyClearEvents(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return clearEvents(c, w, r)
}

// @Title legacyTraceData
// @Description Trace data of a indicated trace.
// @Accept json
// @Success 200 {array} types.CiaoBatchFrameStat "Returns a summary of a trace in the system."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/traces/{label} [get]
// @Resource /v2.1/traces
func legacyTraceData(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return traceData(c, w, r)
}

// @Title listServerDetailsFlavors
// @Description Lists all servers with details for a particular flavor.
// @Accept  json
// @Success 200 {array} compute.ServerDetails "Returns a list of all servers."
// @Failure 400 {object} HTTPReturnErrorCode "The response contains the corresponding message and 40x corresponding code."
// @Failure 500 {object} HTTPReturnErrorCode "The response contains the corresponding message and 50x corresponding code."
// @Router /v2.1/flavors/{flavor}/servers/detail [get]
// @Resource /v2.1/flavors/{flavor}/servers
func listServerDetailsFlavors(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	cxt := &compute.Context{
		Service: c,
	}

	computeResp, err := compute.ListServersDetails(cxt, w, r)

	resp := APIResponse{
		status:   computeResp.Status,
		response: computeResp.Response,
	}

	return resp, err
}

func legacyComputeRoutes(ctl *controller, r *mux.Router) *mux.Router {
	r.Handle("/v2.1/{tenant}/servers/action",
		legacyAPIHandler{ctl, tenantServersAction, false}).Methods("POST")

	r.Handle("/v2.1/flavors/{flavor}/servers/detail",
		legacyAPIHandler{ctl, listServerDetailsFlavors, true}).Methods("GET")

	r.Handle("/v2.1/{tenant}/resources",
		legacyAPIHandler{ctl, listTenantResources, false}).Methods("GET")

	r.Handle("/v2.1/{tenant}/quotas",
		legacyAPIHandler{ctl, listTenantQuotas, false}).Methods("GET")

	r.Handle("/v2.1/tenants",
		legacyAPIHandler{ctl, legacyListTenants, true}).Methods("GET")

	r.Handle("/v2.1/nodes",
		legacyAPIHandler{ctl, legacyListNodes, true}).Methods("GET")
	r.Handle("/v2.1/nodes/summary",
		legacyAPIHandler{ctl, legacyNodesSummary, true}).Methods("GET")
	r.Handle("/v2.1/nodes/{node}/servers/detail",
		legacyAPIHandler{ctl, legacyListNodeServers, true}).Methods("GET")

	r.Handle("/v2.1/cncis",
		legacyAPIHandler{ctl, legacyListCNCIs, true}).Methods("GET")
	r.Handle("/v2.1/cncis/{cnci}/detail",
		legacyAPIHandler{ctl, legacyListCNCIDetails, true}).Methods("GET")

	r.Handle("/v2.1/events",
		legacyAPIHandler{ctl, legacyListEvents, true}).Methods("GET")
	r.Handle("/v2.1/events",
		legacyAPIHandler{ctl, legacyClearEvents, true}).Methods("DELETE")
	r.Handle("/v2.1/{tenant}/events",
		legacyAPIHandler{ctl, legacyListTenantEvents, false}).Methods("GET")

	r.Handle("/v2.1/traces",
		legacyAPIHandler{ctl, legacyListTraces, true}).Methods("GET")
	r.Handle("/v2.1/traces/{label}",
		legacyAPIHandler{ctl, legacyTraceData, true}).Methods("GET")

	return r
}
