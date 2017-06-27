// Copyright (c) 2014,2015,2016,2017 Docker, Inc.
// Copyright (c) 2017 Intel Corporation
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
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli"

	vc "github.com/containers/virtcontainers"
	oci "github.com/containers/virtcontainers/pkg/oci"
)

const formatOptions = `table or json`

// containerState represents the platform agnostic pieces relating to a
// running container's status and state
type containerState struct {
	// Version is the OCI version for the container
	Version string `json:"ociVersion"`
	// ID is the container ID
	ID string `json:"id"`
	// InitProcessPid is the init process id in the parent namespace
	InitProcessPid int `json:"pid"`
	// Status is the current status of the container, running, paused, ...
	Status string `json:"status"`
	// Bundle is the path on the filesystem to the bundle
	Bundle string `json:"bundle"`
	// Rootfs is a path to a directory containing the container's root filesystem.
	Rootfs string `json:"rootfs"`
	// Created is the unix timestamp for the creation time of the container in UTC
	Created time.Time `json:"created"`
	// Annotations is the user defined annotations added to the config.
	Annotations map[string]string `json:"annotations,omitempty"`
	// The owner of the state directory (the owner of the container).
	Owner string `json:"owner"`
}

// hypervisorDetails stores details of the hypervisor used to host
// the container
type hypervisorDetails struct {
	HypervisorPath string `json:"hypervisorPath"`
	ImagePath      string `json:"imagePath"`
	KernelPath     string `json:"kernelPath"`
}

// fullContainerState specifies the core state plus the hypervisor
// details
type fullContainerState struct {
	containerState
	hypervisorDetails `json:"hypervisor"`
}

type formatState interface {
	Write(state []fullContainerState, showAll bool, file *os.File) error
}

type formatJSON struct{}
type formatIDList struct{}
type formatTabular struct{}

var listCommand = cli.Command{
	Name:  "list",
	Usage: "lists containers started by " + name + " with the given root",
	ArgsUsage: `

Where the given root is specified via the global option "--root"
(default: "` + defaultRootDirectory + `").

EXAMPLE 1:
To list containers created via the default "--root":
       # ` + name + ` list

EXAMPLE 2:
To list containers created using a non-default value for "--root":
       # ` + name + ` --root value list`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "format, f",
			Value: "table",
			Usage: `select one of: ` + formatOptions,
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "display only container IDs",
		},
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "display all available information",
		},
	},
	Action: func(context *cli.Context) error {
		s, err := getContainers(context)
		if err != nil {
			return err
		}

		file := os.Stdout
		showAll := context.Bool("all")

		if context.Bool("quiet") {
			return (&formatIDList{}).Write(s, showAll, file)
		}

		switch context.String("format") {
		case "table":
			return (&formatTabular{}).Write(s, showAll, file)

		case "json":
			return (&formatJSON{}).Write(s, showAll, file)

		default:
			return fmt.Errorf("invalid format option")
		}
	},
}

func (f *formatIDList) Write(state []fullContainerState, showAll bool, file *os.File) error {
	for _, item := range state {
		_, err := fmt.Fprintln(file, item.ID)
		if err != nil {
			return err
		}
	}

	return nil
}
func (f *formatTabular) Write(state []fullContainerState, showAll bool, file *os.File) error {
	// values used by runc
	flags := uint(0)
	minWidth := 12
	tabWidth := 1
	padding := 3

	w := tabwriter.NewWriter(file, minWidth, tabWidth, padding, ' ', flags)

	fmt.Fprint(w, "ID\tPID\tSTATUS\tBUNDLE\tCREATED\tOWNER")

	if showAll {
		fmt.Fprint(w, "\tHYPERVISOR\tKERNEL\tIMAGE\n")
	} else {
		fmt.Fprintf(w, "\n")
	}

	for _, item := range state {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s",
			item.ID,
			item.InitProcessPid,
			item.Status,
			item.Bundle,
			item.Created.Format(time.RFC3339Nano),
			item.Owner)

		if showAll {
			fmt.Fprintf(w, "\t%s\t%s\t%s\n",
				item.HypervisorPath,
				item.KernelPath,
				item.ImagePath)
		} else {
			fmt.Fprintf(w, "\n")
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}

func (f *formatJSON) Write(state []fullContainerState, showAll bool, file *os.File) error {
	return json.NewEncoder(file).Encode(state)
}

func getContainers(context *cli.Context) ([]fullContainerState, error) {
	runtimeConfig, ok := context.App.Metadata["runtimeConfig"].(oci.RuntimeConfig)
	if !ok {
		return nil, errors.New("invalid runtime config")
	}

	hypervisorDetails, err := getHypervisorDetails(runtimeConfig)
	if err != nil {
		return nil, err
	}

	podList, err := vc.ListPod()
	if err != nil {
		return nil, err
	}

	var s []fullContainerState

	for _, pod := range podList {
		if len(pod.ContainersStatus) == 0 {
			// ignore empty pods
			continue
		}

		for _, container := range pod.ContainersStatus {
			ociState, err := oci.StatusToOCIState(container)
			if err != nil {
				return nil, err
			}

			s = append(s, fullContainerState{
				containerState: containerState{
					Version:        ociState.Version,
					ID:             ociState.ID,
					InitProcessPid: ociState.Pid,
					Status:         ociState.Status,
					Bundle:         ociState.Bundle,
					Rootfs:         container.RootFs,
					Created:        container.StartTime,
					Annotations:    ociState.Annotations,

					// FIXME: Owner,
				},
				hypervisorDetails: hypervisorDetails,
			})
		}
	}

	return s, nil
}

// getHypervisorDetails returns details of the hypervisor used to host
// the container.
//
// It ensures all paths are fully expanded.
func getHypervisorDetails(runtimeConfig oci.RuntimeConfig) (hypervisorDetails, error) {
	hypervisorPath, err := filepath.EvalSymlinks(runtimeConfig.HypervisorConfig.HypervisorPath)
	if err != nil {
		return hypervisorDetails{}, err
	}

	kernelPath, err := filepath.EvalSymlinks(runtimeConfig.HypervisorConfig.KernelPath)
	if err != nil {
		return hypervisorDetails{}, err
	}

	imagePath, err := filepath.EvalSymlinks(runtimeConfig.HypervisorConfig.ImagePath)
	if err != nil {
		return hypervisorDetails{}, err
	}

	return hypervisorDetails{
		HypervisorPath: hypervisorPath,
		KernelPath:     kernelPath,
		ImagePath:      imagePath,
	}, nil
}
