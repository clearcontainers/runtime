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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

const ciaoAPIPort = 8889

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

// APIResponse contains the http status and any response struct to be marshalled.
type APIResponse struct {
	status   int
	response interface{}
}

func errorResponse(err error) APIResponse {
	switch err {
	case types.ErrQuota:
		return APIResponse{http.StatusForbidden, nil}
	case types.ErrTenantNotFound,
		types.ErrInstanceNotFound:
		return APIResponse{http.StatusNotFound, nil}
	default:
		return APIResponse{http.StatusInternalServerError, nil}
	}
}

// APIHandler is a custom handler for the compute APIs.
// This custom handler allows us to more cleanly return an error and response,
// and pass some package level context into the handler.
type APIHandler struct {
	*controller
	Handler     func(*controller, http.ResponseWriter, *http.Request) (APIResponse, error)
	ContentType string
}

func (h APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", h.ContentType)
	w.WriteHeader(resp.status)
	w.Write(b)
}

type pagerFilterType uint8

const (
	none           pagerFilterType = 0
	workloadFilter                 = 0x1
	statusFilter                   = 0x2
)

type pager interface {
	filter(filterType pagerFilterType, filter string, item interface{}) bool
	nextPage(filterType pagerFilterType, filter string, r *http.Request) ([]byte, error)
}

func pagerQueryParse(r *http.Request) (int, int, string) {
	values := r.URL.Query()
	limit := 0
	offset := 0
	marker := ""
	if values["limit"] != nil {
		l, err := strconv.ParseInt(values["limit"][0], 10, 32)
		if err != nil {
			limit = 0
		} else {
			limit = (int)(l)
		}
	}

	if values["marker"] != nil {
		marker = values["marker"][0]
	} else if values["offset"] != nil {
		o, err := strconv.ParseInt(values["offset"][0], 10, 32)
		if err != nil {
			offset = 0
		} else {
			offset = (int)(o)
		}
	}

	return limit, offset, marker
}

type nodePager struct {
	ctl   *controller
	nodes []types.CiaoComputeNode
}

func (pager *nodePager) getNodes(filterType pagerFilterType, filter string, nodes []types.CiaoComputeNode, limit int, offset int) (types.CiaoComputeNodes, error) {
	computeNodes := types.NewCiaoComputeNodes()

	pageLength := 0

	glog.V(2).Infof("Get nodes limit [%d] offset [%d]", limit, offset)

	if nodes == nil || offset >= len(nodes) {
		return computeNodes, nil
	}

	for _, node := range nodes[offset:] {
		computeNodes.Nodes = append(computeNodes.Nodes, node)

		pageLength++
		if limit > 0 && pageLength >= limit {
			break
		}
	}

	return computeNodes, nil
}

func (pager *nodePager) filter(filterType pagerFilterType, filter string, node types.CiaoComputeNode) bool {
	return false
}

func (pager *nodePager) nextPage(filterType pagerFilterType, filter string, r *http.Request) (types.CiaoComputeNodes, error) {
	limit, offset, lastSeen := pagerQueryParse(r)

	if lastSeen == "" {
		if limit != 0 {
			return pager.getNodes(filterType, filter, pager.nodes,
				limit, offset)
		}

		return pager.getNodes(filterType, filter, pager.nodes, 0,
			offset)
	}

	for i, node := range pager.nodes {
		if node.ID == lastSeen {
			if i >= len(pager.nodes)-1 {
				return pager.getNodes(filterType, filter, nil,
					limit, 0)
			}

			return pager.getNodes(filterType, filter,
				pager.nodes[i+1:], limit, 0)
		}
	}

	return types.CiaoComputeNodes{}, fmt.Errorf("Item %s not found", lastSeen)
}

type nodeServerPager struct {
	ctl       *controller
	instances []types.CiaoServerStats
}

func (pager *nodeServerPager) getNodeServers(filterType pagerFilterType, filter string, instances []types.CiaoServerStats,
	limit int, offset int) (types.CiaoServersStats, error) {
	servers := types.NewCiaoServersStats()

	servers.TotalServers = len(instances)
	pageLength := 0

	glog.V(2).Infof("Get nodes limit [%d] offset [%d]", limit, offset)

	if instances == nil || offset >= len(instances) {
		return servers, nil
	}

	for _, instance := range instances[offset:] {
		servers.Servers = append(servers.Servers, instance)

		pageLength++
		if limit > 0 && pageLength >= limit {
			break
		}
	}

	return servers, nil
}

func (pager *nodeServerPager) filter(filterType pagerFilterType, filter string, instance types.CiaoServerStats) bool {
	return false
}

func (pager *nodeServerPager) nextPage(filterType pagerFilterType, filter string, r *http.Request) (types.CiaoServersStats, error) {
	limit, offset, lastSeen := pagerQueryParse(r)

	glog.V(2).Infof("Next page marker [%s] limit [%d] offset [%d]",
		lastSeen, limit, offset)

	if lastSeen == "" {
		if limit != 0 {
			return pager.getNodeServers(filterType, filter,
				pager.instances, limit, offset)
		}

		return pager.getNodeServers(filterType, filter,
			pager.instances, 0, offset)
	}

	for i, instance := range pager.instances {
		if instance.ID == lastSeen {
			if i >= len(pager.instances)-1 {
				return pager.getNodeServers(filterType, filter,
					nil, limit, 0)
			}

			return pager.getNodeServers(filterType, filter,
				pager.instances[i+1:], limit, 0)
		}
	}

	return types.CiaoServersStats{}, fmt.Errorf("Item %s not found", lastSeen)
}

const (
	instances int = 1
	vcpu          = 2
	memory        = 3
	disk          = 4
)

func getResources(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	var tenantResource types.CiaoTenantResources

	vars := mux.Vars(r)
	tenant := vars["tenant"]

	err := c.confirmTenant(tenant)
	if err != nil {
		return errorResponse(err), err
	}

	t, err := c.ds.GetTenant(tenant)
	if err != nil || t == nil {
		return errorResponse(types.ErrTenantNotFound), types.ErrTenantNotFound
	}

	resources := t.Resources

	tenantResource.ID = t.ID

	for _, resource := range resources {
		switch resource.Rtype {
		case instances:
			tenantResource.InstanceLimit = resource.Limit
			tenantResource.InstanceUsage = resource.Usage

		case vcpu:
			tenantResource.VCPULimit = resource.Limit
			tenantResource.VCPUUsage = resource.Usage

		case memory:
			tenantResource.MemLimit = resource.Limit
			tenantResource.MemUsage = resource.Usage

		case disk:
			tenantResource.DiskLimit = resource.Limit
			tenantResource.DiskUsage = resource.Usage
		}
	}

	return APIResponse{http.StatusOK, tenantResource}, nil
}

func tenantQueryParse(r *http.Request) (time.Time, time.Time, error) {
	values := r.URL.Query()
	var startTime, endTime time.Time

	if values["start_date"] == nil || values["end_date"] == nil {
		return startTime, endTime, fmt.Errorf("Missing date")
	}

	startTime, err := time.Parse(time.RFC3339, values["start_date"][0])
	if err != nil {
		return startTime, endTime, err
	}

	endTime, err = time.Parse(time.RFC3339, values["end_date"][0])
	if err != nil {
		return startTime, endTime, err
	}

	return startTime, endTime, nil
}

func getUsage(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	var usage types.CiaoUsageHistory

	start, end, err := tenantQueryParse(r)
	if err != nil {
		return errorResponse(err), err
	}

	glog.V(2).Infof("Start %v\n", start)
	glog.V(2).Infof("End %v\n", end)

	usage.Usages, err = c.ds.GetTenantUsage(tenant, start, end)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusOK, usage}, nil
}

type instanceAction func(string) error

func serversAction(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	var servers types.CiaoServersAction
	var actionFunc instanceAction
	var statusFilter string

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errorResponse(err), err
	}

	err = json.Unmarshal(body, &servers)
	if err != nil {
		return errorResponse(err), err
	}

	if servers.Action == "os-start" {
		actionFunc = c.restartInstance
		statusFilter = payloads.Exited
	} else if servers.Action == "os-stop" {
		actionFunc = c.stopInstance
		statusFilter = payloads.Running
	} else if servers.Action == "os-delete" {
		actionFunc = c.deleteInstance
		statusFilter = ""
	} else {
		return APIResponse{http.StatusServiceUnavailable, nil},
			errors.New("Unsupported action")
	}

	if len(servers.ServerIDs) > 0 {
		for _, instanceID := range servers.ServerIDs {
			// make sure the instance belongs to the tenant
			instance, err := c.ds.GetInstance(instanceID)

			if err != nil {
				return errorResponse(err), err
			}

			if instance.TenantID != tenant {
				return errorResponse(err), err
			}
			actionFunc(instanceID)
		}
	} else {
		/* We want to act on all relevant instances */
		instances, err := c.ds.GetAllInstancesFromTenant(tenant)
		if err != nil {
			return errorResponse(err), err
		}

		for _, instance := range instances {
			if statusFilter != "" &&
				instance.State != statusFilter {
				continue
			}

			actionFunc(instance.ID)
		}
	}

	return APIResponse{http.StatusAccepted, nil}, nil
}

func listTenants(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	var computeTenants types.CiaoComputeTenants

	tenants, err := c.ds.GetAllTenants()
	if err != nil {
		return errorResponse(err), err
	}

	for _, tenant := range tenants {
		computeTenants.Tenants = append(computeTenants.Tenants,
			struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				ID:   tenant.ID,
				Name: tenant.Name,
			},
		)
	}

	return APIResponse{http.StatusOK, computeTenants}, nil
}

func listNodes(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	computeNodes := c.ds.GetNodeLastStats()

	nodeSummary, err := c.ds.GetNodeSummary()
	if err != nil {
		return errorResponse(err), err
	}

	for _, node := range nodeSummary {
		for i := range computeNodes.Nodes {
			if computeNodes.Nodes[i].ID != node.NodeID {
				continue
			}

			computeNodes.Nodes[i].TotalInstances =
				node.TotalInstances
			computeNodes.Nodes[i].TotalRunningInstances =
				node.TotalRunningInstances
			computeNodes.Nodes[i].TotalPendingInstances =
				node.TotalPendingInstances
			computeNodes.Nodes[i].TotalPausedInstances =
				node.TotalPausedInstances
		}
	}

	sort.Sort(types.SortedComputeNodesByID(computeNodes.Nodes))

	pager := nodePager{
		ctl:   c,
		nodes: computeNodes.Nodes,
	}

	resp, err := pager.nextPage(none, "", r)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusOK, resp}, nil
}

func nodesSummary(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	var nodesStatus types.CiaoClusterStatus

	computeNodes := c.ds.GetNodeLastStats()

	glog.V(2).Infof("nodesSummary %d nodes", len(computeNodes.Nodes))

	nodesStatus.Status.TotalNodes = len(computeNodes.Nodes)
	for _, node := range computeNodes.Nodes {
		if node.Status == ssntp.READY.String() {
			nodesStatus.Status.TotalNodesReady++
		} else if node.Status == ssntp.FULL.String() {
			nodesStatus.Status.TotalNodesFull++
		} else if node.Status == ssntp.OFFLINE.String() {
			nodesStatus.Status.TotalNodesOffline++
		} else if node.Status == ssntp.MAINTENANCE.String() {
			nodesStatus.Status.TotalNodesMaintenance++
		}
	}

	return APIResponse{http.StatusOK, nodesStatus}, nil
}

func listNodeServers(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	nodeID := vars["node"]

	serversStats := c.ds.GetInstanceLastStats(nodeID)

	instances, err := c.ds.GetAllInstancesByNode(nodeID)
	if err != nil {
		return errorResponse(err), err
	}

	for _, instance := range instances {
		for i := range serversStats.Servers {
			if serversStats.Servers[i].ID != instance.ID {
				continue
			}

			serversStats.Servers[i].TenantID = instance.TenantID
			serversStats.Servers[i].IPv4 = instance.IPAddress
		}
	}

	pager := nodeServerPager{
		ctl:       c,
		instances: serversStats.Servers,
	}

	resp, err := pager.nextPage(none, "", r)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusOK, resp}, nil
}

func listCNCIs(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	var ciaoCNCIs types.CiaoCNCIs

	cncis, err := c.ds.GetTenantCNCISummary("")
	if err != nil {
		return errorResponse(err), err
	}

	var subnets []types.CiaoCNCISubnet

	for _, cnci := range cncis {
		if cnci.InstanceID == "" {
			continue
		}

		for _, subnet := range cnci.Subnets {
			subnets = append(subnets,
				types.CiaoCNCISubnet{
					Subnet: subnet,
				},
			)
		}

		ciaoCNCIs.CNCIs = append(ciaoCNCIs.CNCIs,
			types.CiaoCNCI{
				ID:       cnci.InstanceID,
				TenantID: cnci.TenantID,
				IPv4:     cnci.IPAddress,
				Subnets:  subnets,
			},
		)
	}

	return APIResponse{http.StatusOK, ciaoCNCIs}, nil
}

func listCNCIDetails(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	cnciID := vars["cnci"]
	var ciaoCNCI types.CiaoCNCI

	cncis, err := c.ds.GetTenantCNCISummary(cnciID)
	if err != nil {
		return errorResponse(err), err
	}

	if len(cncis) > 0 {
		var subnets []types.CiaoCNCISubnet
		cnci := cncis[0]

		for _, subnet := range cnci.Subnets {
			subnets = append(subnets,
				types.CiaoCNCISubnet{
					Subnet: subnet,
				},
			)
		}

		ciaoCNCI = types.CiaoCNCI{
			ID:       cnci.InstanceID,
			TenantID: cnci.TenantID,
			IPv4:     cnci.IPAddress,
			Subnets:  subnets,
		}
	}

	return APIResponse{http.StatusOK, ciaoCNCI}, err
}

func listTraces(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	var traces types.CiaoTracesSummary

	summaries, err := c.ds.GetBatchFrameSummary()
	if err != nil {
		return errorResponse(err), err
	}

	for _, s := range summaries {
		summary := types.CiaoTraceSummary{
			Label:     s.BatchID,
			Instances: s.NumInstances,
		}
		traces.Summaries = append(traces.Summaries, summary)
	}

	return APIResponse{http.StatusOK, traces}, err
}

func listEvents(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]

	events := types.NewCiaoEvents()

	logs, err := c.ds.GetEventLog()
	if err != nil {
		return errorResponse(err), err
	}

	for _, l := range logs {
		if tenant != "" && tenant != l.TenantID {
			continue
		}

		event := types.CiaoEvent{
			Timestamp: l.Timestamp,
			TenantID:  l.TenantID,
			EventType: l.EventType,
			Message:   l.Message,
		}
		events.Events = append(events.Events, event)
	}

	return APIResponse{http.StatusOK, events}, err
}

func clearEvents(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	err := c.ds.ClearLog()
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusAccepted, nil}, nil
}

func traceData(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	label := vars["label"]
	var traceData types.CiaoTraceData

	batchStats, err := c.ds.GetBatchFrameStatistics(label)
	if err != nil {
		return errorResponse(err), err
	}

	traceData.Summary = types.CiaoBatchFrameStat{
		NumInstances:             batchStats[0].NumInstances,
		TotalElapsed:             batchStats[0].TotalElapsed,
		AverageElapsed:           batchStats[0].AverageElapsed,
		AverageControllerElapsed: batchStats[0].AverageControllerElapsed,
		AverageLauncherElapsed:   batchStats[0].AverageLauncherElapsed,
		AverageSchedulerElapsed:  batchStats[0].AverageSchedulerElapsed,
		VarianceController:       batchStats[0].VarianceController,
		VarianceLauncher:         batchStats[0].VarianceLauncher,
		VarianceScheduler:        batchStats[0].VarianceScheduler,
	}

	return APIResponse{http.StatusOK, traceData}, nil
}
