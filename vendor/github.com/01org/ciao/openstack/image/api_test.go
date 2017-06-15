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
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/01org/ciao/service"
)

// TBD - can some of this stuff be pulled out into a common test area?
type test struct {
	method           string
	pattern          string
	handler          func(*Context, http.ResponseWriter, *http.Request) (APIResponse, error)
	request          string
	expectedStatus   int
	expectedResponse string
}

var tests = []test{
	{
		"GET",
		"/",
		listAPIVersions,
		"",
		http.StatusOK,
		`{"versions":[{"status":"CURRENT","id":"v2.3","links":[{"href":"` + fmt.Sprintf("https://%s:9292/v2/", myHostname()) + `","rel":"self"}]}]}`,
	},
	{
		"POST",
		"/v2/images",
		createImage,
		`{"container_format":"bare","disk_format":"raw","name":"Ubuntu","id":"b2173dd3-7ad6-4362-baa6-a68bce3565cb","visibility":"private"}`,
		http.StatusCreated,
		`{"status":"queued","container_format":"bare","min_ram":0,"updated_at":"2015-11-29T22:21:42Z","owner":"bab7d5c60cd041a0a36f7c4b6e1dd978","min_disk":0,"tags":[],"locations":[],"visibility":"private","id":"b2173dd3-7ad6-4362-baa6-a68bce3565cb","size":null,"virtual_size":null,"name":"Ubuntu","checksum":null,"created_at":"2015-11-29T22:21:42Z","disk_format":"raw","properties":null,"protected":false,"self":"/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb","file":"/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file","schema":"/v2/schemas/image"}`,
	},
	{
		"POST",
		"/v2/images",
		createImage,
		"",
		http.StatusInternalServerError,
		fmt.Sprintf("unexpected end of JSON input\nnull"),
	},
	{
		"GET",
		"/v2/images",
		listImages,
		"",
		http.StatusOK,
		`{"images":[{"status":"queued","container_format":"bare","min_ram":0,"updated_at":"2015-11-29T22:21:42Z","owner":"bab7d5c60cd041a0a36f7c4b6e1dd978","min_disk":0,"tags":[],"locations":[],"visibility":"private","id":"b2173dd3-7ad6-4362-baa6-a68bce3565cb","size":null,"virtual_size":null,"name":"Ubuntu","checksum":null,"created_at":"2015-11-29T22:21:42Z","disk_format":"raw","properties":null,"protected":false,"self":"/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb","file":"/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file","schema":"/v2/schemas/image"}],"schema":"/v2/schemas/images","first":"/v2/images"}`,
	},
	{
		"GET",
		"/v2/images/1bea47ed-f6a9-463b-b423-14b9cca9ad27",
		getImage,
		"",
		http.StatusOK,
		`{"status":"active","container_format":"bare","min_ram":0,"updated_at":"2014-05-05T17:15:11Z","owner":"5ef70662f8b34079a6eddb8da9d75fe8","min_disk":0,"tags":[],"locations":[],"visibility":"public","id":"1bea47ed-f6a9-463b-b423-14b9cca9ad27","size":13167616,"virtual_size":null,"name":"cirros-0.3.2-x86_64-disk","checksum":"64d7c1cd2b6f60c92c14662941cb7913","created_at":"2014-05-05T17:15:10Z","disk_format":"qcow2","properties":null,"protected":false,"self":"/v2/images/1bea47ed-f6a9-463b-b423-14b9cca9ad27","file":"/v2/images/1bea47ed-f6a9-463b-b423-14b9cca9ad27/file","schema":"/v2/schemas/image"}`,
	},
	{
		"DELETE",
		"/v2/images/1bea47ed-f6a9-463b-b423-14b9cca9ad27",
		deleteImage,
		"",
		http.StatusNoContent,
		`null`,
	},
	{
		"PUT",
		"/v2/images/1bea47ed-f6a9-463b-b423-14b9cca9ad27",
		uploadImage,
		"",
		http.StatusNoContent,
		`null`,
	},
}

const testTenantID = "1bea47ed-f6a9-463b-b423-14b9cca9ad27"

func myHostname() string {
	host, _ := os.Hostname()
	return host
}

type testImageService struct{}

func (is testImageService) CreateImage(tenantID string, req CreateImageRequest) (DefaultResponse, error) {
	format := Bare
	name := "Ubuntu"
	createdAt, _ := time.Parse(time.RFC3339, "2015-11-29T22:21:42Z")
	updatedAt, _ := time.Parse(time.RFC3339, "2015-11-29T22:21:42Z")
	minDisk := 0
	minRAM := 0
	owner := "bab7d5c60cd041a0a36f7c4b6e1dd978"

	return DefaultResponse{
		Status:          Queued,
		ContainerFormat: &format,
		CreatedAt:       createdAt,
		Tags:            make([]string, 0),
		DiskFormat:      Raw,
		Visibility:      Private,
		UpdatedAt:       &updatedAt,
		Locations:       make([]string, 0),
		Self:            "/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb",
		MinDisk:         &minDisk,
		Protected:       false,
		ID:              "b2173dd3-7ad6-4362-baa6-a68bce3565cb",
		File:            "/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file",
		Owner:           &owner,
		MinRAM:          &minRAM,
		Schema:          "/v2/schemas/image",
		Name:            &name,
	}, nil
}

func (is testImageService) ListImages(tenantID string) ([]DefaultResponse, error) {
	format := Bare
	name := "Ubuntu"
	createdAt, _ := time.Parse(time.RFC3339, "2015-11-29T22:21:42Z")
	updatedAt, _ := time.Parse(time.RFC3339, "2015-11-29T22:21:42Z")
	minDisk := 0
	minRAM := 0
	owner := "bab7d5c60cd041a0a36f7c4b6e1dd978"

	image := DefaultResponse{
		Status:          Queued,
		ContainerFormat: &format,
		CreatedAt:       createdAt,
		Tags:            make([]string, 0),
		DiskFormat:      Raw,
		Visibility:      Private,
		UpdatedAt:       &updatedAt,
		Locations:       make([]string, 0),
		Self:            "/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb",
		MinDisk:         &minDisk,
		Protected:       false,
		ID:              "b2173dd3-7ad6-4362-baa6-a68bce3565cb",
		File:            "/v2/images/b2173dd3-7ad6-4362-baa6-a68bce3565cb/file",
		Owner:           &owner,
		MinRAM:          &minRAM,
		Schema:          "/v2/schemas/image",
		Name:            &name,
	}

	var images []DefaultResponse

	if tenantID == testTenantID {
		images = append(images, image)
	}

	return images, nil
}

func (is testImageService) GetImage(tenantID, ID string) (DefaultResponse, error) {
	imageID := "1bea47ed-f6a9-463b-b423-14b9cca9ad27"
	format := Bare
	name := "cirros-0.3.2-x86_64-disk"
	createdAt, _ := time.Parse(time.RFC3339, "2014-05-05T17:15:10Z")
	updatedAt, _ := time.Parse(time.RFC3339, "2014-05-05T17:15:11Z")
	minDisk := 0
	minRAM := 0
	owner := "5ef70662f8b34079a6eddb8da9d75fe8"
	checksum := "64d7c1cd2b6f60c92c14662941cb7913"
	size := 13167616

	return DefaultResponse{
		Status:          Active,
		ContainerFormat: &format,
		CreatedAt:       createdAt,
		Tags:            make([]string, 0),
		DiskFormat:      QCow,
		Visibility:      Public,
		UpdatedAt:       &updatedAt,
		Locations:       make([]string, 0),
		Self:            fmt.Sprintf("/v2/images/%s", imageID),
		MinDisk:         &minDisk,
		Protected:       false,
		CheckSum:        &checksum,
		ID:              imageID,

		File:   fmt.Sprintf("/v2/images/%s/file", imageID),
		Owner:  &owner,
		MinRAM: &minRAM,
		Schema: "/v2/schemas/image",
		Name:   &name,
		Size:   &size,
	}, nil
}

func (is testImageService) UploadImage(string, string, io.Reader) (NoContentImageResponse, error) {
	return NoContentImageResponse{}, nil
}

func (is testImageService) DeleteImage(string, string) (NoContentImageResponse, error) {
	return NoContentImageResponse{}, nil
}

func TestRoutes(t *testing.T) {
	var is testImageService
	config := APIConfig{9292, is}

	r := Routes(config, nil)
	if r == nil {
		t.Fatalf("No routes returned")
	}
}

func TestAPIResponse(t *testing.T) {
	var is testImageService

	// TBD: add context to test definition so it can be created per
	// endpoint with either a pass testVolumeService or a failure
	// one.
	context := &Context{9292, is, nil}

	for _, tt := range tests {
		req, err := http.NewRequest(tt.method, tt.pattern, bytes.NewBuffer([]byte(tt.request)))
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		handler := APIHandler{context, tt.handler}

		ctx := service.SetPrivilege(req.Context(), true)
		ctx = service.SetTenantID(ctx, testTenantID)
		if err != nil {
			t.Fatalf("Error on setting tenant [%v]", testTenantID)
		}

		handler.ServeHTTP(rr, req.WithContext(ctx))

		status := rr.Code
		if status != tt.expectedStatus {
			t.Errorf("got %v, expected %v", status, tt.expectedStatus)
		}

		if rr.Body.String() != tt.expectedResponse {
			t.Errorf("%s: failed\ngot: %v\nexp: %v", tt.pattern, rr.Body.String(), tt.expectedResponse)
		}
	}
}
