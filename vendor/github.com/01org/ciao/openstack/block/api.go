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

package block

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

// APIPort is the standard OpenStack Volume port
const APIPort = 8776

// VersionStatus defines whether a reported version is supported or not.
type VersionStatus string

const (
	// Deprecated indicates the api deprecates the spec version.
	Deprecated VersionStatus = "DEPRECATED"

	// Supported indicates a spec version is supported by the api
	Supported VersionStatus = "SUPPORTED"

	// Current indicates the current spec version of the api
	// TBD: can this be eliminated? do we need both supported & current?
	Current VersionStatus = "CURRENT"
)

// Link is used by the API to create the link json strings.
type Link struct {
	Href string `json:"href"`
	Type string `json:"type,omitempty"`
	Rel  string `json:"rel,omitempty"`
}

// MediaType indicates whether the api supports json or xml
type MediaType struct {
	Base string `json:"base"`
	Type string `json:"type"`
}

// Version contains information about the api version that is supported.
type Version struct {
	Status     VersionStatus `json:"status"`
	Updated    string        `json:"updated"`
	Links      []Link        `json:"links"`
	MinVersion string        `json:"min_version"`
	Version    string        `json:"version"`
	MediaTypes []MediaType   `json:"media-types"`
	ID         string        `json:"id"`
}

// Versions is used to create the version api strings.
type Versions struct {
	Versions []Version `json:"versions"`
}

// Choice is used to indicate the api choices that are supported.
type Choice struct {
	Status     VersionStatus `json:"status"`
	MediaTypes []MediaType   `json:"media-types"`
	ID         string        `json:"id"`
	Links      []Link        `json:"links"`
}

// Choices is used to create the choices json strings.
type Choices struct {
	Choices []Choice `json:"choices"`
}

// AbsoluteLimits implements the block api absolute limits object.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#showAbsoluteLimits
// Note that an absolute limit value of -1 indicates that the limit is infinite.
type AbsoluteLimits struct {
	TotalSnapshotsUsed       int `json:"totalSnapshotsUsed"`
	MaxTotalBackups          int `json:"maxTotalBackups"`
	MaxTotalVolumeGigabytes  int `json:"maxTotalVolumeGigabytes"`
	MaxTotalSnapshots        int `json:"maxTotalSnapshots"`
	MaxTotalBackupGigabytes  int `json:"maxTotalBackupGigabytes"`
	TotalBackupGigabytesUsed int `json:"totalBackupGigabytesUsed"`
	MaxTotalVolumes          int `json:"maxTotalVolumes"`
	TotalVolumesUsed         int `json:"totalVolumesUsed"`
	TotalBackupsUsed         int `json:"totalBackupsUsed"`
	TotalGigabytesUsed       int `json:"totalGigabytesUsed"`
}

// Limit implements the limit object.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#showAbsoluteLimits
type Limit struct {
	Rate     []string       `json:"rate"`
	Absolute AbsoluteLimits `json:"absolute"`
}

// Limits implements the limits object.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#showAbsoluteLimits
type Limits struct {
	Limits Limit `json:"limits"`
}

// VolumeStatus is the status of create, list, update, or delete volumes.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#volumes-v2-volumes
type VolumeStatus string

const (
	// Creating indicates that a volume is being created.
	Creating VolumeStatus = "creating"

	// Available indicates that a volume is ready to be used.
	Available VolumeStatus = "available"

	// Attaching indicates that a volume is being attached.
	Attaching VolumeStatus = "attaching"

	// InUse indicates that a volume is attached to an instance.
	InUse VolumeStatus = "in-use"

	// Deleting indicates that a volume is being deleted.
	Deleting VolumeStatus = "deleting"

	// Error indicates that a volume creation error occurred.
	Error VolumeStatus = "error"

	// ErrorDeleting indicates that a volume deletion error occurred.
	ErrorDeleting VolumeStatus = "error_deleting"

	// BackingUp indicates that the volume is being backed up.
	BackingUp VolumeStatus = "backing-up"

	// RestoringBackup indicates that a backup is being restored
	// to the volume.
	RestoringBackup VolumeStatus = "restoring-backup"

	// ErrorRestoring indicates that a backup restoration error occurred.
	ErrorRestoring VolumeStatus = "error_restoring"

	// ErrorExtending indicates that an error occurred
	// while attempting to extend a volume.
	ErrorExtending VolumeStatus = "error_extending"
)

// MetaData is defined as a set of arbitrary key value structs.
type MetaData interface{}

// RequestedVolume contains information about a volume to be created.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#createVolume
type RequestedVolume struct {
	Size               int      `json:"size"`
	AvailabilityZone   string   `json:"availability_zone"`
	SourceVolID        *string  `json:"source_volid"`
	Description        *string  `json:"description"`
	MultiAttach        bool     `json:"multiattach"`
	SnapshotID         *string  `json:"snapshot_id"`
	Name               *string  `json:"name"`
	ImageRef           *string  `json:"imageRef"`
	VolumeType         *string  `json:"volume_type"`
	MetaData           MetaData `json:"metadata"`
	SourceReplica      *string  `json:"source_replica"`
	ConsistencyGroupID *string  `json:"consistencygroup_id"`
}

// VolumeCreateRequest is the json request for the createVolume endpoint.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#createVolume
type VolumeCreateRequest struct {
	Volume RequestedVolume `json:"volume"`
}

// Attachment contains instance attachment information.
// If this volume is attached to a server, the attachment contains
// information about the attachment.
type Attachment struct {
	ServerUUID     string  `json:"server_id"`
	AttachmentUUID string  `json:"attachment_id"`
	HostName       *string `json:"host_name"`
	VolumeUUID     string  `json:"volume_id"`
	Device         string  `json:"device"`
	DeviceUUID     string  `json:"id"`
}

const (
	// ReplicationDisabled indicates that replication is not enabled.
	ReplicationDisabled string = "disabled"
)

// Volume contains information about a volume that has been created or updated.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#createVolume
// http://developer.openstack.org/api-ref-blockstorage-v2.html#updateVolume
type Volume struct {
	Status             VolumeStatus `json:"status"`
	MigrationStatus    *string      `json:"migration_status"`
	UserID             string       `json:"user_id"`
	Attachments        []Attachment `json:"attachments"`
	Links              []Link       `json:"links"`
	AvailabilityZone   *string      `json:"availability_zone"`
	Bootable           string       `json:"bootable"`
	Encrypted          bool         `json:"encrypted"`
	CreatedAt          *time.Time   `json:"created_at"`
	Description        *string      `json:"description"`
	UpdatedAt          *time.Time   `json:"updated_at"`
	VolumeType         *string      `json:"volume_type"`
	Name               *string      `json:"name"`
	ReplicationStatus  string       `json:"replication_status"`
	ConsistencyGroupID *string      `json:"consistencygroup_id"`
	SourceVolID        *string      `json:"source_volid"`
	SnapshotID         *string      `json:"snapshot_id"`
	MultiAttach        bool         `json:"multiattach"`
	MetaData           MetaData     `json:"metadata"`
	ID                 string       `json:"id"`
	Size               int          `json:"size"`
}

// VolumeResponse is the json response for the createVolume, and updateVolume endpoint.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#createVolume
// http://developer.openstack.org/api-ref-blockstorage-v2.html#updateVolume
type VolumeResponse struct {
	Volume Volume `json:"volume"`
}

// ListVolume is the contains volume information for the listVolume endpoint.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#listVolumes
type ListVolume struct {
	ID    string `json:"id"`
	Links []Link `json:"links"`
	Name  string `json:"name"`
}

// ListVolumes is the json response for the listVolume endpoint.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#listVolumes
type ListVolumes struct {
	Volumes []ListVolume `json:"volumes"`
}

// VolumeDetail contains volume information for the listVolumeDetails endpoint.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#listVolumesDetail
type VolumeDetail struct {
	MigrationStatus          *string      `json:"migration_status"`
	Attachments              []Attachment `json:"attachments"`
	Links                    []Link       `json:"links"`
	AvailabilityZone         *string      `json:"availability_zone"`
	OSVolHostAttr            string       `json:"os-vol-host-attr:host"`
	Encrypted                bool         `json:"encrypted"`
	UpdatedAt                *time.Time   `json:"updated_at"`
	OSVolReplicationExStatus string       `json:"os-volume-replication:extended_status,omitempty"`
	ReplicationStatus        string       `json:"replication_status"`
	SnapshotID               *string      `json:"snapshot_id"`
	ID                       string       `json:"id"`
	Size                     int          `json:"size"`
	UserID                   string       `json:"user_id"`
	OSVolTenantAttr          string       `json:"os-vol-tenant-attr:tenant_id"`
	OSVolMigStatusAttrStatus *string      `json:"os-vol-mig-status-attr:migstat"`
	MetaData                 MetaData     `json:"metadata"`
	Status                   VolumeStatus `json:"status"`
	Description              *string      `json:"description"`
	MultiAttach              bool         `json:"multiattach"`
	OSVolReplicationDriver   string       `json:"os-volume-replication:driver_data,omitempty"`
	SourceVolID              *string      `json:"source_volid"`
	ConsistencyGroupID       *string      `json:"consistencygroup_id"`
	OSVolMigStatusAttrNameID *string      `json:"os-vol-mig-status-attr:name_id"`
	Name                     *string      `json:"name"`
	Bootable                 string       `json:"bootable"`
	CreatedAt                *time.Time   `json:"created_at"`
	VolumeType               *string      `json:"volume_type"`
}

// ListVolumesDetail is the json response for the listVolumeDetails endpoint.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#listVolumesDetail
type ListVolumesDetail struct {
	Volumes []VolumeDetail `json:"volumes"`
}

// ShowVolumeDetails is the json response for the showVolumeDetail endpoint.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#showVolume
type ShowVolumeDetails struct {
	Volume VolumeDetail `json:"volume"`
}

// These errors can be returned by the Service interface
var (
	ErrQuota                = errors.New("Tenant over quota")
	ErrTenantNotFound       = errors.New("Tenant not found")
	ErrVolumeNotFound       = errors.New("Volume not found")
	ErrInstanceNotFound     = errors.New("Instance not found")
	ErrVolumeNotAvailable   = errors.New("Volume not available")
	ErrVolumeOwner          = errors.New("You are not volume owner")
	ErrInstanceOwner        = errors.New("You are not instance owner")
	ErrInstanceNotAvailable = errors.New("Instance not available")
	ErrVolumeNotAttached    = errors.New("Volume not attached")
)

// errorResponse maps service error responses to http responses.
// this helper function can help functions avoid having to switch
// on return values all the time.
func errorResponse(err error) APIResponse {
	switch err {
	case ErrQuota:
		return APIResponse{http.StatusForbidden, nil}
	case ErrTenantNotFound:
		return APIResponse{http.StatusNotFound, nil}
	case ErrVolumeNotFound:
		return APIResponse{http.StatusNotFound, nil}
	case ErrInstanceNotFound:
		return APIResponse{http.StatusNotFound, nil}
	case ErrVolumeNotAvailable,
		ErrVolumeNotAvailable,
		ErrVolumeOwner,
		ErrInstanceOwner,
		ErrInstanceNotAvailable,
		ErrVolumeNotAttached:
		return APIResponse{http.StatusForbidden, nil}
	default:
		return APIResponse{http.StatusInternalServerError, nil}
	}
}

// APIConfig contains information needed to start the block api service.
type APIConfig struct {
	Port       int     // the https port of the block api service
	VolService Service // the service interface
}

// Service contains the required interface to the block service.
// The caller who is starting the api service needs to provide this
// interface.
type Service interface {
	GetAbsoluteLimits(tenant string) (AbsoluteLimits, error)
	CreateVolume(tenant string, req RequestedVolume) (Volume, error)
	DeleteVolume(tenant string, volume string) error
	AttachVolume(tenant string, volume string, instance string, mountpoint string) error
	DetachVolume(tenant string, volume string, attachment string) error
	ListVolumes(tenant string) ([]ListVolume, error)
	ListVolumesDetail(tenant string) ([]VolumeDetail, error)
	ShowVolumeDetails(tenant string, volume string) (VolumeDetail, error)
}

// Context contains data and interfaces that the block api will need.
// TBD: do we really need this, or is just a service interface sufficient?
type Context struct {
	port int
	Service
}

// APIResponse is returned from the API handlers.
type APIResponse struct {
	status   int
	response interface{}
}

// APIHandler is a custom handler for the block APIs.
// This custom handler allows us to more cleanly return an error and response,
// and pass some package level context into the handler.
type APIHandler struct {
	*Context
	Handler func(*Context, http.ResponseWriter, *http.Request) (APIResponse, error)
}

// ServeHTTP satisfies the http Handler interface.
// It wraps our api response in json as well.
func (h APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Handler(h.Context, w, r)
	if err != nil {
		http.Error(w, http.StatusText(resp.status), resp.status)
	}

	b, err := json.Marshal(resp.response)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.status)
	w.Write(b)
}

// not completed
func listAPIVersions(context *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	host := r.Host
	var href string
	if host == "" {
		var err error
		host, err = os.Hostname()
		if err != nil {
			return APIResponse{http.StatusInternalServerError, nil}, err
		}
		href = fmt.Sprintf("https://%s:%d/v2/", host, context.port)
	} else {
		href = fmt.Sprintf("https://%s/v2/", host)
	}

	// TBD clean up this code
	var resp Versions

	// need to create Links
	docLink := Link{
		Href: "http://docs.openstack.org/",
		Type: "text/html",
		Rel:  "describedby",
	}

	selfLink := Link{
		Href: href,
		Rel:  "self",
	}

	jsonType := MediaType{
		Base: "application/json",
		Type: "application/vnd.openstack.volume+json;version=1",
	}

	// I'm not sure how much of this struct is important
	v := Version{
		Status:     Supported,
		ID:         "v2.0",
		Links:      []Link{docLink, selfLink},
		MediaTypes: []MediaType{jsonType},
		Updated:    "2014-06-28T12:20:21Z",
	}

	resp.Versions = append(resp.Versions, v)

	return APIResponse{http.StatusOK, resp}, nil
}

// not completed
func showAPIv2Details(context *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	host := r.Host
	var href string
	if host == "" {
		var err error
		host, err = os.Hostname()
		if err != nil {
			return APIResponse{http.StatusInternalServerError, nil}, err
		}
		href = fmt.Sprintf("https://%s:%d/v2/v2.json", host, context.port)
	} else {
		href = fmt.Sprintf("https://%s/v2/v2.json", host)
	}

	// we only support json
	mt := MediaType{
		Base: "application/json",
		Type: "application/vnd.openstack.volume+json;version=1",
	}

	selfLink := Link{
		Href: href,
		Rel:  "self",
	}

	choice := Choice{
		Status:     Current,
		ID:         "v2.0",
		MediaTypes: []MediaType{mt},
		Links:      []Link{selfLink},
	}

	resp := Choices{Choices: []Choice{choice}}

	return APIResponse{http.StatusOK, resp}, nil
}

func showAbsoluteLimits(bc *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]

	abs, err := bc.GetAbsoluteLimits(tenant)
	if err != nil {
		return errorResponse(err), err
	}

	resp := Limits{Limit{make([]string, 0), abs}}

	return APIResponse{http.StatusOK, resp}, nil
}

func createVolume(bc *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return APIResponse{http.StatusBadRequest, nil}, err
	}

	var req VolumeCreateRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		return APIResponse{http.StatusInternalServerError, nil}, err
	}

	var resp VolumeResponse

	vol, err := bc.CreateVolume(tenant, req.Volume)
	if err != nil {
		return errorResponse(err), err
	}

	resp.Volume = vol

	return APIResponse{http.StatusAccepted, resp}, nil
}

func listVolumes(bc *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]

	// TBD: support sorting and paging

	vols, err := bc.ListVolumes(tenant)
	if err != nil {
		return errorResponse(err), err
	}

	resp := ListVolumes{Volumes: vols}

	return APIResponse{http.StatusOK, resp}, nil
}

func listVolumesDetail(bc *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]

	// TBD: support sorting and paging

	vols, err := bc.ListVolumesDetail(tenant)
	if err != nil {
		return errorResponse(err), err
	}

	resp := ListVolumesDetail{Volumes: vols}

	return APIResponse{http.StatusOK, resp}, nil
}

func showVolumeDetails(bc *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	volume := vars["volume_id"]

	vol, err := bc.ShowVolumeDetails(tenant, volume)
	if err != nil {
		return errorResponse(err), err
	}

	resp := ShowVolumeDetails{Volume: vol}

	return APIResponse{http.StatusOK, resp}, nil
}

func deleteVolume(bc *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	volume := vars["volume_id"]

	// TBD - satisfy preconditions here, or in interface?
	err := bc.DeleteVolume(tenant, volume)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusAccepted, nil}, nil
}

func volumeActionAttach(bc *Context, m map[string]interface{}, tenant string, volume string) (APIResponse, error) {
	val := m["os-attach"]

	m = val.(map[string]interface{})

	val, ok := m["instance_uuid"]
	if !ok {
		// we have to have the instance uuid
		return APIResponse{http.StatusBadRequest, nil}, nil
	}
	instance := val.(string)

	val, ok = m["mountpoint"]
	if !ok {
		// we have to have the mountpoint ?
		return APIResponse{http.StatusBadRequest, nil}, nil
	}
	mountPoint := val.(string)

	err := bc.AttachVolume(tenant, volume, instance, mountPoint)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusAccepted, nil}, nil
}

func volumeActionDetach(bc *Context, m map[string]interface{}, tenant string, volume string) (APIResponse, error) {
	val := m["os-detach"]

	m = val.(map[string]interface{})

	// attachment-id is optional
	var attachment string
	val = m["attachment-id"]
	if val != nil {
		attachment = val.(string)
	}

	err := bc.DetachVolume(tenant, volume, attachment)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusAccepted, nil}, nil
}

func volumeAction(bc *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	volume := vars["volume_id"]

	var req interface{}

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return APIResponse{http.StatusBadRequest, nil}, err
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		return APIResponse{http.StatusInternalServerError, nil}, err
	}

	m := req.(map[string]interface{})

	// for now, we will support only attach and detach

	if m["os-attach"] != nil {
		return volumeActionAttach(bc, m, tenant, volume)
	}

	if m["os-detach"] != nil {
		return volumeActionDetach(bc, m, tenant, volume)
	}

	return APIResponse{http.StatusBadRequest, nil}, err
}

// Routes provides gorilla mux routes for the supported endpoints.
func Routes(config APIConfig) *mux.Router {
	// make new Context
	context := &Context{config.Port, config.VolService}

	r := mux.NewRouter()

	// API versions
	r.Handle("/", APIHandler{context, listAPIVersions}).Methods("GET")
	r.Handle("/v2", APIHandler{context, showAPIv2Details}).Methods("GET")

	// Limits
	r.Handle("/v2/{tenant}/limits",
		APIHandler{context, showAbsoluteLimits}).Methods("GET")

	// Volumes
	r.Handle("/v2/{tenant}/volumes",
		APIHandler{context, createVolume}).Methods("POST")
	r.Handle("/v2/{tenant}/volumes",
		APIHandler{context, listVolumes}).Methods("GET")
	r.Handle("/v2/{tenant}/volumes/detail",
		APIHandler{context, listVolumesDetail}).Methods("GET")
	r.Handle("/v2/{tenant}/volumes/{volume_id}",
		APIHandler{context, showVolumeDetails}).Methods("GET")
	r.Handle("/v2/{tenant}/volumes/{volume_id}",
		APIHandler{context, deleteVolume}).Methods("DELETE")

	// Volume actions
	r.Handle("/v2/{tenant}/volumes/{volume_id}/action",
		APIHandler{context, volumeAction}).Methods("POST")

	return r
}
