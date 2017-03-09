// Copyright (c) 2014,2015,2016 Docker, Inc.
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
	"fmt"
	"os"

	vc "github.com/containers/virtcontainers"
	"github.com/containers/virtcontainers/pkg/oci"
	"github.com/urfave/cli"
)

var createCommand = cli.Command{
	Name:  "create",
	Usage: "Create a container",
	ArgsUsage: `<container-id>

   <container-id> is your name for the instance of the container that you
   are starting. The name you provide for the container instance must be unique
   on your host.`,
	Description: `The create command creates an instance of a container for a bundle. The
   bundle is a directory with a specification file named "` + specConfig + `" and a
   root filesystem.
   The specification file includes an args parameter. The args parameter is
   used to specify command(s) that get run when the container is started.
   To change the command(s) that get executed on start, edit the args
   parameter of the spec.`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "bundle, b",
			Value: "",
			Usage: `path to the root of the bundle directory, defaults to the current directory`,
		},
		cli.StringFlag{
			Name:  "console",
			Value: "",
			Usage: "path to a pseudo terminal",
		},
		cli.StringFlag{
			Name:  "pid-file",
			Value: "",
			Usage: "specify the file to write the process id to",
		},
	},
	Action: func(context *cli.Context) error {
		return create(context.Args().First(),
			context.String("bundle"),
			context.String("console"),
			context.String("pid-file"))
	},
}

func create(containerID, bundlePath, console, pidFilePath string) error {
	// Checks the MUST and MUST NOT from OCI runtime specification
	if err := validCreateParams(containerID, bundlePath); err != nil {
		return err
	}

	podConfig, shimConfig, err := getConfigs(bundlePath, containerID, console)
	if err != nil {
		return err
	}

	pod, err := vc.CreatePod(podConfig)
	if err != nil {
		return err
	}

	// Start the shim to retrieve its PID.
	containers := pod.GetContainers()
	if len(containers) != 1 {
		return fmt.Errorf("BUG: Container list from pod is wrong, expecting only one container, found %d containers", len(containers))
	}

	pid, err := startContainerShim(containers[0], shimConfig, pod.URL())
	if err != nil {
		return err
	}

	// Creation of PID file has to be the last thing done in the create
	// because containerd considers the create complete after this file
	// is created.
	if err := createPIDFile(pidFilePath, pid); err != nil {
		return err
	}

	return nil
}

func getConfigs(bundlePath, containerID, console string) (vc.PodConfig, ShimConfig, error) {
	runtimeConfig, shimConfig, err := loadConfiguration("")
	if err != nil {
		return vc.PodConfig{}, ShimConfig{}, err
	}

	podConfig, err := oci.PodConfig(runtimeConfig, bundlePath, containerID, console)
	if err != nil {
		return vc.PodConfig{}, ShimConfig{}, err
	}

	return *podConfig, shimConfig, nil
}

func createPIDFile(pidFilePath string, pid int) error {
	if pidFilePath == "" {
		return fmt.Errorf("Missing PID file path")
	}

	if err := os.RemoveAll(pidFilePath); err != nil {
		return err
	}

	f, err := os.Create(pidFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	pidStr := fmt.Sprintf("%d", pid)

	n, err := f.WriteString(pidStr)
	if err != nil {
		return err
	}

	if n < len(pidStr) {
		return fmt.Errorf("Could not write pid to '%s': only %d bytes written out of %d", pidFilePath, n, len(pidStr))
	}

	return nil
}
