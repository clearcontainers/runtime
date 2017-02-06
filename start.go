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
	vc "github.com/containers/virtcontainers"
	"github.com/urfave/cli"
)

var startCommand = cli.Command{
	Name:  "start",
	Usage: "executes the user defined process in a created container",
	ArgsUsage: `<container-id> [container-id...]

   <container-id> is your name for the instance of the container that you
   are starting. The name you provide for the container instance must be
   unique bon your host.`,
	Description: `The start command executes the user defined process in a created container .`,
	Action: func(context *cli.Context) error {
		return start(context.String("container-id"))
	},
}

func start(containerID string) error {
	// Checks the MUST and MUST NOT from OCI runtime specification
	if err := validContainer(containerID); err != nil {
		return err
	}

	if _, err := vc.StartPod(containerID); err != nil {
		return err
	}

	return nil
}
