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

	"github.com/urfave/cli"
)

func percent(active, total int64) float64 {
	if total == 0 {
		total = 1
	}
	return 100 * float64(active) / float64(total)
}

func report(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 1 {
		return fmt.Errorf("expecting one argument, got %d", len(args))
	}

	profiles, err := ParseProfiles(args[0])
	if err != nil {
		return err
	}

	var active, total int64
	for _, profile := range profiles {
		blocks := profile.Blocks
		for i := range blocks {
			statements := int64(blocks[i].NumStmt)
			if blocks[i].Count > 0 {
				active += statements
			}
			total += statements
		}
	}

	fmt.Printf("coverage: %.1f%% of statements\n", percent(active, total))

	return nil
}

var reportCommand = cli.Command{

	Name:      "report",
	Usage:     "report coverage information",
	ArgsUsage: "profile",
	Action:    report,
}
