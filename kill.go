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
	"github.com/urfave/cli"
)

var killCommand = cli.Command{
	Name:  "kill",
	Usage: "Kill sends signals to the container's init process",
	ArgsUsage: `<container-id> [signal]

   <container-id> is the name for the instance of the container
   [signal] is the signal to be sent to the init process (default: SIGTERM)

EXAMPLE:
   If the container id is "ubuntu01" the following will send a "KILL" signal
   to the init process of the "ubuntu01" container:
	 
       # runc kill ubuntu01 KILL`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "send the specified signal to all processes inside the container",
		},
	},
	Action: func(context *cli.Context) error {
		// TODO
		return nil
	},
}
