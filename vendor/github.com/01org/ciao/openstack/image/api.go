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

package image

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/01org/ciao/ssntp/uuid"
	"github.com/gorilla/mux"
)

// APIPort is the standard OpenStack Image port
const APIPort = 9292

// TBD - are these thing shared enough between OpenStack services
// to be pulled out to a common area?
// ---------

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

// Version is used by the API to create the version json strings
type Version struct {
	Status VersionStatus `json:"status"`
	ID     string        `json:"id"`
	Links  []Link        `json:"links"`
}

// Versions creates multiple version json strings
type Versions struct {
	Versions []Version `json:"versions"`
}

// --------
// end possible common json

// Status defines the possible states for an image
type Status string

const (
	// Queued means that the image service reserved an image ID
	// for the image but did not yet upload any image data.
	Queued Status = "queued"

	// Saving means that the image service is currently uploading
	// the raw data for the image.
	Saving Status = "saving"

	// Active means that the image is active and fully available
	// in the image service.
	Active Status = "active"

	// Killed means that an image data upload error occurred.
	Killed Status = "killed"

	// Deleted means that the image service retains information
	// about the image but the image is no longer available for use.
	Deleted Status = "deleted"

	// PendingDelete is similar to the deleted status.
	// An image in this state is not recoverable.
	PendingDelete Status = "pending_delete"
)

// Visibility defines whether an image is per tenant or public.
type Visibility string

const (
	// Public indicates that the image can be used by anyone.
	Public Visibility = "public"

	// Private indicates that the image is only available to a tenant.
	Private Visibility = "private"
)

// ContainerFormat defines the acceptable container format strings.
type ContainerFormat string

const (
	// Bare is the only format we support right now.
	Bare ContainerFormat = "bare"
)

// DiskFormat defines the valid values for the disk_format string
type DiskFormat string

// we support the following disk formats
const (
	// Raw
	Raw DiskFormat = "raw"

	// QCow
	QCow DiskFormat = "qcow2"

	// ISO
	ISO DiskFormat = "iso"
)

// ErrorImage defines all possible image handling errors
type ErrorImage error

var (
	// ErrNoImage is returned when an image is not found.
	ErrNoImage = errors.New("Image not found")

	// ErrImageSaving is returned when an image is being uploaded.
	ErrImageSaving = errors.New("Image being uploaded")

	// ErrBadUUID is returned when an invalid UUID is specified
	ErrBadUUID = errors.New("Bad UUID")

	// ErrAlreadyExists is returned when an attempt is made to add
	// an image with a UUID that already exists.
	ErrAlreadyExists = errors.New("Already Exists")

	// ErrDecodeImage is returned when there was an error on image decoding
	ErrDecodeImage = errors.New("Error on Image decode")
)

// CreateImageRequest contains information for a create image request.
// http://developer.openstack.org/api-ref/image/v2/index.html#create-an-image
type CreateImageRequest struct {
	Name            string          `json:"name,omitempty"`
	ID              string          `json:"id,omitempty"`
	Visibility      Visibility      `json:"visibility,omitempty"`
	Tags            []string        `json:"tags,omitempty"`
	ContainerFormat ContainerFormat `json:"container_format,omitempty"`
	DiskFormat      DiskFormat      `json:"disk_format,omitempty"`
	MinDisk         int             `json:"min_disk,omitempty"`
	MinRAM          int             `json:"min_ram,omitempty"`
	Protected       bool            `json:"protected,omitempty"`
	Properties      interface{}     `json:"properties,omitempty"`
}

// DefaultResponse contains information about an image
// http://developer.openstack.org/api-ref/image/v2/index.html#create-an-image
type DefaultResponse struct {
	Status          Status           `json:"status"`
	ContainerFormat *ContainerFormat `json:"container_format"`
	MinRAM          *int             `json:"min_ram"`
	UpdatedAt       *time.Time       `json:"updated_at"`
	Owner           *string          `json:"owner"`
	MinDisk         *int             `json:"min_disk"`
	Tags            []string         `json:"tags"`
	Locations       []string         `json:"locations"`
	Visibility      Visibility       `json:"visibility"`
	ID              string           `json:"id"`
	Size            *int             `json:"size"`
	VirtualSize     *int             `json:"virtual_size"`
	Name            *string          `json:"name"`
	CheckSum        *string          `json:"checksum"`
	CreatedAt       time.Time        `json:"created_at"`
	DiskFormat      DiskFormat       `json:"disk_format"`
	Properties      interface{}      `json:"properties"`
	Protected       bool             `json:"protected"`
	Self            string           `json:"self"`
	File            string           `json:"file"`
	Schema          string           `json:"schema"`
}

// ListImagesResponse contains the list of all images that have been created.
// http://developer.openstack.org/api-ref/image/v2/index.html#show-images
type ListImagesResponse struct {
	Images []DefaultResponse `json:"images"`
	Schema string            `json:"schema"`
	First  string            `json:"first"`
}

// NoContentImageResponse contains the UUID of the image which content
// got uploaded or deleted
// http://developer.openstack.org/api-ref/image/v2/index.html#upload-binary-image-data
type NoContentImageResponse struct {
	ImageID string `json:"image_id"`
}

// TBD - can we pull these structs out into some sort of common
// api service file?
// ----------

// APIConfig contains information needed to start the block api service.
type APIConfig struct {
	Port         int     // the https port of the block api service
	ImageService Service // the service interface
}

// Service is the interface that the api requires in order to get
// information needed to implement the image endpoints.
type Service interface {
	CreateImage(CreateImageRequest) (DefaultResponse, error)
	UploadImage(string, io.Reader) (NoContentImageResponse, error)
	ListImages() ([]DefaultResponse, error)
	GetImage(string) (DefaultResponse, error)
	DeleteImage(string) (NoContentImageResponse, error)
}

// Context contains data and interfaces that the image api will need.
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

// APIHandler is a custom handler for the image APIs.
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

// ---------------
// end possible common service api stuff

// errorResponse maps service error responses to http responses.
// this helper function can help functions avoid having to switch
// on return values all the time.
func errorResponse(err error) APIResponse {
	switch err {
	case ErrNoImage:
		return APIResponse{http.StatusNotFound, nil}
	case ErrBadUUID:
		return APIResponse{http.StatusBadRequest, nil}
	case ErrAlreadyExists:
		return APIResponse{http.StatusConflict, nil}
	default:
		return APIResponse{http.StatusInternalServerError, nil}
	}
}

// endpoints
func listAPIVersions(context *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	host, err := os.Hostname()
	if err != nil {
		return APIResponse{http.StatusInternalServerError, nil}, err
	}

	// maybe we should just put href in context
	href := fmt.Sprintf("https://%s:%d/v2/", host, context.port)

	// TBD clean up this code
	var resp Versions

	selfLink := Link{
		Href: href,
		Rel:  "self",
	}

	v := Version{
		Status: Current,
		ID:     "v2.3",
		Links:  []Link{selfLink},
	}

	resp.Versions = append(resp.Versions, v)

	return APIResponse{http.StatusOK, resp}, nil
}

// createImage creates information about an image, but doesn't contain
// any actual image.
//
// TBD: this endpoint has no tenant var - how do you know who owns the
// image if the tenant id isn't passed in? Default visibility is supposed
// to be private.
func createImage(context *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return APIResponse{http.StatusBadRequest, nil}, err
	}

	var req CreateImageRequest

	err = json.Unmarshal(body, &req)
	if err != nil {
		return APIResponse{http.StatusInternalServerError, nil}, err
	}

	resp, err := context.CreateImage(req)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusCreated, resp}, nil
}

// listImages returns a list of all created images.
//
// TBD: support query & sort parameters
func listImages(context *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	images, err := context.ListImages()
	if err != nil {
		return errorResponse(err), err
	}

	resp := ListImagesResponse{
		Images: images,
		Schema: "/v2/schemas/images",
		First:  "/v2/images",
	}

	return APIResponse{http.StatusOK, resp}, nil
}

// getImage get information about an image by image_id field
//
func getImage(context *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	imageID := vars["image_id"]

	resp, err := context.GetImage(imageID)
	if err != nil {
		return errorResponse(err), err
	}
	return APIResponse{http.StatusOK, resp}, nil
}

func uploadImage(context *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	imageID := vars["image_id"]

	_, err := context.UploadImage(imageID, r.Body)
	if err != nil {
		return errorResponse(err), err
	}
	return APIResponse{http.StatusNoContent, nil}, nil
}

func deleteImage(context *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	imageID := vars["image_id"]

	_, err := context.DeleteImage(imageID)
	if err != nil {
		return errorResponse(err), err
	}
	return APIResponse{http.StatusNoContent, nil}, nil
}

// Routes provides gorilla mux routes for the supported endpoints.
func Routes(config APIConfig) *mux.Router {
	// make new Context
	context := &Context{config.Port, config.ImageService}

	r := mux.NewRouter()

	// API versions
	r.Handle("/", APIHandler{context, listAPIVersions}).Methods("GET")
	r.Handle("/v2/images", APIHandler{context, createImage}).Methods("POST")
	r.Handle("/v2/images/{image_id:"+uuid.UUIDRegex+"}/file", APIHandler{context, uploadImage}).Methods("PUT")
	r.Handle("/v2/images", APIHandler{context, listImages}).Methods("GET")
	r.Handle("/v2/images/{image_id:"+uuid.UUIDRegex+"}", APIHandler{context, getImage}).Methods("GET")
	r.Handle("/v2/images/{image_id:"+uuid.UUIDRegex+"}", APIHandler{context, deleteImage}).Methods("DELETE")

	return r
}
