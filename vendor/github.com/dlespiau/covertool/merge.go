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
	"io"
	"io/ioutil"
	"os"
	"strings"

	"bufio"

	"github.com/urfave/cli"
)

type Mode string

type mergeContext struct {
	// Where to write the combined result file.
	out io.Writer
	// Ensure we keep the same Mode throughout the input files.
	mode string
}

func parseMode(line string) (string, error) {
	const p = "mode: "

	if !strings.HasPrefix(line, p) || line == p {
		return "", fmt.Errorf("merge: bad mode line: %v", line)
	}
	mode := line[len(p):]

	if mode != "set" && mode != "count" && mode != "atomic" {
		return "", fmt.Errorf("merge: bad mode line: %v", line)
	}

	return mode, nil
}

func compatibleModes(a, b string) bool {
	if a == "set" && b != "set" {
		return false
	}
	if a != "set" && b == "set" {
		return false
	}
	return true
}

func (ctx *mergeContext) addFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Parse the coverage mode.
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return err
	}
	line := scanner.Text()
	mode, err := parseMode(line)
	if err != nil {
		return err
	}
	if ctx.mode == "" {
		ctx.mode = mode
		ctx.out.Write([]byte(line))
		ctx.out.Write([]byte("\n"))
	} else if !compatibleModes(ctx.mode, mode) {
		return fmt.Errorf("merge: inconsistency in the profiles modes, got both %v and %v", ctx.mode, mode)
	}

	// Copy the list of block information after skipping the line we just parsed.
	f.Seek(int64(len([]byte(line))+1), 0)
	io.Copy(ctx.out, f)

	return nil
}

// WriteProfile writes a profile out to w.
func WriteProfile(w io.Writer, profiles []*Profile) error {
	if len(profiles) == 0 {
		return nil
	}

	// We've checked that all input profiles have compatible modes, so we just
	// write out the first one.
	fmt.Fprintf(w, "mode: %s\n", profiles[0].Mode)
	for _, profile := range profiles {
		blocks := profile.Blocks
		for i := range profile.Blocks {
			fmt.Fprintf(w, "%s:%d.%d,%d.%d %d %d\n", profile.FileName,
				blocks[i].StartLine, blocks[i].StartCol,
				blocks[i].EndLine, blocks[i].EndCol,
				blocks[i].NumStmt, blocks[i].Count)
		}
	}

	return nil
}

func merge(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) < 2 {
		return fmt.Errorf("expecting at least two arguments, got %d", len(args))
	}
	aggregatedProfile, err := ioutil.TempFile("", "cover-tool-")
	if err != nil {
		return err
	}
	defer func() {
		aggregatedProfile.Close()
		os.Remove(aggregatedProfile.Name())
	}()

	parse := &mergeContext{
		out: aggregatedProfile,
	}

	for _, arg := range args {
		if err := parse.addFile(arg); err != nil {
			return err
		}
	}

	// Loading the aggregated profile will merge counters for the same blocks.
	profiles, err := ParseProfiles(aggregatedProfile.Name())
	if err != nil {
		return err
	}

	// Write out the new profile
	output := ctx.String("output")
	out := io.Writer(os.Stdout)
	if output != "-" {
		f, err := os.Create(output)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}

	if err := WriteProfile(out, profiles); err != nil {
		return err
	}

	return nil
}

var mergeCommand = cli.Command{
	Name:      "merge",
	Usage:     "merge two or more profiles",
	ArgsUsage: "profile profile [profile]*",
	Action:    merge,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "output, o",
			Value: "-",
			Usage: "output file",
		},
	},
}
