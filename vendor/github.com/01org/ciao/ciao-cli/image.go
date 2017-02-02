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
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/imageservice/v2/images"
	"github.com/rackspace/gophercloud/pagination"
)

const imageTemplateDesc = `struct {
	Name             string   // Image name
	SizeBytes        int      // Size of image in bytes
	ID               string   // Image UUID
	Status           string   // Image status.  Can be queued or active
	CreatedDate      string   // Image creation date
	LastUpdate       string   // Timestamp of last update
	File             string   // Image path
	Schema           string   // Path to json schema
}`

var imageCommand = &command{
	SubCommands: map[string]subCommand{
		"add":    new(imageAddCommand),
		"show":   new(imageShowCommand),
		"list":   new(imageListCommand),
		"delete": new(imageDeleteCommand),
	},
}

type imageAddCommand struct {
	Flag     flag.FlagSet
	name     string
	id       string
	file     string
	template string
}

func (cmd *imageAddCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image add [flags]

Creates a new image

The add flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a 

%s
`, imageTemplateDesc)
	fmt.Fprintln(os.Stderr, templateFunctionHelp)
	os.Exit(2)
}

func (cmd *imageAddCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Image Name")
	cmd.Flag.StringVar(&cmd.id, "id", "", "Image UUID")
	cmd.Flag.StringVar(&cmd.file, "file", "", "Image file to upload")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageAddCommand) run(args []string) error {
	if cmd.name == "" {
		return errors.New("Missing required -name parameter")
	}

	if cmd.file == "" {
		return errors.New("Missing required -file parameter")
	}

	_, err := os.Stat(cmd.file)
	if err != nil {
		fatalf("Could not open %s [%s]\n", cmd.file, err)
	}

	client, err := imageServiceClient(*identityUser, *identityPassword, *tenantID)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	opts := images.CreateOpts{
		Name: cmd.name,
		ID:   cmd.id,
	}

	image, err := images.Create(client, opts).Extract()
	if err != nil {
		fatalf("Could not create image [%s]\n", err)
	}

	uploadTenantImage(*identityUser, *identityPassword, *tenantID, image.ID, cmd.file)
	image, err = images.Get(client, image.ID).Extract()
	if err != nil {
		fatalf("Could not retrieve new created image [%s]\n", err)
	}

	if cmd.template != "" {
		return outputToTemplate("image-add", cmd.template, image)
	}

	fmt.Printf("Created image:\n")
	dumpImage(image)
	return nil
}

type imageShowCommand struct {
	Flag     flag.FlagSet
	image    string
	template string
}

func (cmd *imageShowCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image show

Show images
`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a 

%s
`, imageTemplateDesc)
	fmt.Fprintln(os.Stderr, templateFunctionHelp)
	os.Exit(2)
}

func (cmd *imageShowCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.StringVar(&cmd.image, "image", "", "Image UUID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageShowCommand) run(args []string) error {
	if cmd.image == "" {
		return errors.New("Missing required -image parameter")
	}

	client, err := imageServiceClient(*identityUser, *identityPassword, *tenantID)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	i, err := images.Get(client, cmd.image).Extract()
	if err != nil {
		fatalf("Could not retrieve image %s [%s]\n", cmd.image, err)
	}

	if cmd.template != "" {
		return outputToTemplate("image-show", cmd.template, i)
	}

	dumpImage(i)

	return nil
}

type imageListCommand struct {
	Flag     flag.FlagSet
	template string
}

func (cmd *imageListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image list

List images
`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a 

[]%s

As images are retrieved in pages, the template may be applied multiple
times.  You can not therefore rely on the length of the slice passed
to the template to determine the total number of images.
`, imageTemplateDesc)
	fmt.Fprintln(os.Stderr, templateFunctionHelp)
	os.Exit(2)
}

func (cmd *imageListCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageListCommand) run(args []string) error {
	client, err := imageServiceClient(*identityUser, *identityPassword, *tenantID)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	var t *template.Template
	if cmd.template != "" {
		t = createTemplate("image-list", cmd.template)
	}

	pager := images.List(client, images.ListOpts{})

	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		imageList, err := images.ExtractImages(page)
		if err != nil {
			errorf("Could not extract image [%s]\n", err)
		}

		if t != nil {
			if err = t.Execute(os.Stdout, &imageList); err != nil {
				fatalf(err.Error())
			}
			return false, nil
		}

		for k, i := range imageList {
			fmt.Printf("Image #%d\n", k+1)
			dumpImage(&i)
			fmt.Printf("\n")
		}
		return false, nil
	})
	return err
}

type imageDownloadCommand struct {
	Flag  flag.FlagSet
	image string
	file  string
}

func (cmd *imageDownloadCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image download [flags]

Fetch an image

The download flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *imageDownloadCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.image, "image", "", "Image UUID")
	cmd.Flag.StringVar(&cmd.file, "file", "", "Filename to save the image (default will print to stdout)")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageDownloadCommand) run(args []string) (err error) {
	client, err := imageServiceClient(*identityUser, *identityPassword, *tenantID)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	r, err := images.Download(client, cmd.image).Extract()
	if err != nil {
		fatalf("Could not download image [%s]\n", err)
	}

	dest := os.Stdout
	if cmd.file != "" {
		dest, err = os.Create(cmd.file)
		defer func() {
			closeErr := dest.Close()
			if err == nil {
				err = closeErr
			}
		}()
		if err != nil {
			fatalf("Could not create destination file: %s: %v", cmd.file, err)
		}
	}

	_, err = io.Copy(dest, r)
	if err != nil {
		fatalf("Error copying to destination: %v", err)
	}

	return nil
}

type imageDeleteCommand struct {
	Flag  flag.FlagSet
	image string
}

func (cmd *imageDeleteCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image delete [flags]

Deletes an image

The delete flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *imageDeleteCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.image, "image", "", "Image UUID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageDeleteCommand) run(args []string) error {
	client, err := imageServiceClient(*identityUser, *identityPassword, *tenantID)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	res := images.Delete(client, cmd.image)
	if res.Err != nil {
		fatalf("Could not delete Image [%s]\n", res.Err)
	}
	fmt.Printf("Deleted image %s\n", cmd.image)
	return res.Err
}

func uploadTenantImage(username, password, tenant, image, filename string) error {
	client, err := imageServiceClient(username, password, tenant)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	file, err := os.Open(filename)
	if err != nil {
		fatalf("Could not open %s [%s]", filename, err)
	}
	defer file.Close()

	res := images.Upload(client, image, file)
	if res.Err != nil {
		fatalf("Could not upload %s [%s]", filename, res.Err)
	}
	return res.Err
}

type imageModifyCommand struct {
	Flag  flag.FlagSet
	name  string
	image string
}

func (cmd *imageModifyCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image modify [flags]

Modify an image

The modify flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *imageModifyCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Image Name")
	cmd.Flag.StringVar(&cmd.image, "image", "", "Image UUID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageModifyCommand) run(args []string) error {
	if cmd.image == "" {
		return errors.New("Missing required -image parameter")
	}

	client, err := imageServiceClient(*identityUser, *identityPassword, *tenantID)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	var opts images.UpdateOpts
	if cmd.name != "" {
		n := images.ReplaceImageName{
			NewName: cmd.name,
		}
		opts = append(opts, n)
	}

	image, err := images.Update(client, cmd.image, opts).Extract()
	if err != nil {
		fatalf("Could not update image's properties [%s]\n", err)
	}

	fmt.Printf("Updated image:\n")
	dumpImage(image)
	return nil
}

func dumpImage(i *images.Image) {
	fmt.Printf("\tName             [%s]\n", i.Name)
	fmt.Printf("\tSize             [%d bytes]\n", i.SizeBytes)
	fmt.Printf("\tUUID             [%s]\n", i.ID)
	fmt.Printf("\tStatus           [%s]\n", i.Status)
	fmt.Printf("\tCreatedDate      [%s]\n", i.CreatedDate)
}

func imageServiceClient(username, password, tenant string) (*gophercloud.ServiceClient, error) {
	opt := gophercloud.AuthOptions{
		IdentityEndpoint: *identityURL + "/v3/",
		Username:         username,
		Password:         password,
		DomainID:         "default",
		TenantID:         tenant,
		AllowReauth:      true,
	}

	provider, err := newAuthenticatedClient(opt)
	if err != nil {
		errorf("Could not get AuthenticatedClient %s\n", err)
		return nil, err
	}

	return openstack.NewImageServiceV2(provider, gophercloud.EndpointOpts{
		Name:   "glance",
		Region: "RegionOne",
	})
}
