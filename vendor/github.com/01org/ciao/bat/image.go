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

// Package bat contains a number of helper functions that can be used to perform
// various operations on a ciao cluster such as creating an instance or retrieving
// a list of all the defined workloads, etc.  All of these helper functions are
// implemented by calling ciao-cli rather than by using ciao's REST APIs.  This
// package is mainly intended for use by BAT tests.  Manipulating the cluster
// via ciao-cli, rather than through the REST APIs, allows us to test a little
// bit more of ciao.
package bat

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

// ImageOptions contains user supplied image meta data
type ImageOptions struct {
	Name             string
	ID               string
	ContainerFormat  string
	DiskFormat       string
	MinDiskGigabytes int
	MinRAMMegabytes  int
	Protected        bool
	Visibility       string
	Tags             []string
}

// Image contains all the meta data for a single image
type Image struct {
	ImageOptions
	SizeBytes   int `json:"size"`
	Status      string
	Owner       string
	Checksum    string
	CreatedDate string
	LastUpdate  string
	File        string
	Schema      string
}

func computeImageAddArgs(options *ImageOptions) []string {
	args := make([]string, 0, 8)

	if options.ContainerFormat != "" {
		args = append(args, "-container-format", options.ContainerFormat)
	}
	if options.DiskFormat != "" {
		args = append(args, "-disk-format", options.DiskFormat)
	}
	if options.ID != "" {
		args = append(args, "-id", options.ID)
	}
	if options.MinDiskGigabytes != 0 {
		args = append(args, "-min-disk-size",
			fmt.Sprintf("%d", options.MinDiskGigabytes))
	}
	if options.MinRAMMegabytes != 0 {
		args = append(args, "-min-ram-size",
			fmt.Sprintf("%d", options.MinRAMMegabytes))
	}
	if options.Name != "" {
		args = append(args, "-name", options.Name)
	}
	if options.Protected {
		args = append(args, "-protected")
	}
	if len(options.Tags) > 0 {
		args = append(args, "-tags", strings.Join(options.Tags, ","))
	}
	if options.Visibility != "" {
		args = append(args, "-visibility", options.Visibility)
	}

	return args
}

// AddImage uploads a new image to the ciao-image service.  The caller can
// supply a number of pieces of meta data about the image via the options
// parameter.  It is implemented by calling ciao-cli image add.
// On success the function returns the entire meta data of the
// newly updated image that includes the caller supplied meta data and the
// meta data added by the image service.  An error will be returned
// if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func AddImage(ctx context.Context, tenant, path string, options *ImageOptions) (*Image, error) {
	var img *Image
	args := []string{"image", "add", "-f", "{{tojson .}}", "-file", path}
	args = append(args, computeImageAddArgs(options)...)
	err := RunCIAOCLIAsAdminJS(ctx, tenant, args, &img)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// AddRandomImage uploads a new image of the desired size using random data.
// The caller can supply a number of pieces of meta data about the image via
// the options parameter.  It is implemented by calling ciao-cli image add.  On
// success the function returns the entire meta data of the newly updated image
// that includes the caller supplied meta data and the meta data added by the
// image service.  An error will be returned if the following environment
// variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_ADMIN_USERNAME,
// CIAO_ADMIN_PASSWORD.
func AddRandomImage(ctx context.Context, tenant string, size int, options *ImageOptions) (*Image, error) {
	path, err := CreateRandomFile(size)
	if err != nil {
		return nil, fmt.Errorf("Unable to create random file : %v", err)
	}
	defer func() { _ = os.Remove(path) }()
	return AddImage(ctx, tenant, path, options)
}

// DeleteImage deletes an image from the image service.  It is implemented
// by calling ciao-cli image delete.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func DeleteImage(ctx context.Context, tenant, ID string) error {
	args := []string{"image", "delete", "-image", ID}
	_, err := RunCIAOCLIAsAdmin(ctx, tenant, args)
	return err
}

// GetImage retrieves the meta data for a given image.  It is implemented by
// calling ciao-cli image show.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func GetImage(ctx context.Context, tenant, ID string) (*Image, error) {
	var img *Image
	args := []string{"image", "show", "-image", ID, "-f", "{{tojson .}}"}

	err := RunCIAOCLIAsAdminJS(ctx, tenant, args, &img)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// GetImages retrieves the meta data for all images.  It is implemented by
// calling ciao-cli image list.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func GetImages(ctx context.Context, tenant string) (map[string]*Image, error) {
	var images map[string]*Image
	template := `
{
{{- range $i, $val := .}}
  {{- if $i }},{{end}}
  "{{$val.ID | js }}" : {{tojson $val}}
{{- end }}
}
`
	args := []string{"image", "list", "-f", template}
	err := RunCIAOCLIAsAdminJS(ctx, tenant, args, &images)
	if err != nil {
		return nil, err
	}

	return images, nil
}

// GetImageCount returns the number of images currently stored in the
// image service.  It is implemented by calling ciao-cli image list.
// An error will be returned if the following environment variables are
// not set; CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_ADMIN_USERNAME,
// CIAO_ADMIN_PASSWORD.
func GetImageCount(ctx context.Context, tenant string) (int, error) {
	args := []string{"image", "list", "-f", "{{len .}}"}

	data, err := RunCIAOCLIAsAdmin(ctx, tenant, args)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(data))
}

// UploadImage overrides the contents of an existing image with a new file.  It is
// implemented by calling ciao-cli image upload.  An error will be returned if the
// following environment variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func UploadImage(ctx context.Context, tenant, ID, path string) error {
	args := []string{"image", "upload", "-image", ID, "-file", path}
	_, err := RunCIAOCLIAsAdmin(ctx, tenant, args)
	return err
}

// CreateRandomFile creates a file of the desired size with random data
// returning the path.
func CreateRandomFile(sizeMiB int) (path string, err error) {
	var f *os.File
	f, err = ioutil.TempFile("/tmp", "ciao-random-")
	if err != nil {
		return
	}
	defer func() {
		err1 := f.Close()
		if err1 != nil && err == nil {
			err = err1
		}
	}()

	b := make([]byte, sizeMiB*1024*1024)
	_, err = rand.Read(b)
	if err != nil {
		return
	}
	_, err = f.Write(b)
	if err == nil {
		path = f.Name()
	}

	return
}
