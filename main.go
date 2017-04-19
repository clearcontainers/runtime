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
	"io"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	vc "github.com/containers/virtcontainers"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli"
)

// name holds the name of this program
const name = "cc-runtime"

// version is the runtime version. It is be specified at compilation time (see
// Makefile).
var version = ""

// commit is the git commit the runtime is compiled from. It is specified at
// compilation time (see Makefile)
var commit = ""

// specConfig is the name of the file holding the containers configuration
const specConfig = "config.json"

const usage = `Clear Containers runtime

cc-runtime is a command line program for running applications packaged
according to the Open Container Initiative (OCI).`

var ccLog = logrus.New()

func main() {
	app := cli.NewApp()
	app.Name = name
	app.Usage = usage

	v := make([]string, 0, 3)
	if version != "" {
		v = append(v, "runtime  : "+version)
	}
	if commit != "" {
		v = append(v, "   commit   : "+commit)
	}
	v = append(v, "   OCI specs: "+specs.Version)
	app.Version = strings.Join(v, "\n")

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output for logging",
		},
		cli.StringFlag{
			Name:  "log",
			Value: "/dev/null",
			Usage: "set the log file path where internal debug information is written",
		},
		cli.StringFlag{
			Name:  "log-format",
			Value: "text",
			Usage: "set the format used by logs ('text' (default), or 'json')",
		},
		cli.StringFlag{
			Name:  "root",
			Value: "/run/clearcontainers",
			Usage: "root directory for storage of container state (this should be located in tmpfs)",
		},
	}

	app.Commands = []cli.Command{
		createCommand,
		deleteCommand,
		execCommand,
		killCommand,
		startCommand,
		stateCommand,
	}

	app.Before = func(context *cli.Context) error {
		if context.GlobalBool("debug") {
			ccLog.Level = logrus.DebugLevel
		}
		if path := context.GlobalString("log"); path != "" {
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_SYNC, 0640)
			if err != nil {
				return err
			}
			ccLog.Out = f
		}
		switch context.GlobalString("log-format") {
		case "text":
			// retain logrus's default.
		case "json":
			ccLog.Formatter = new(logrus.JSONFormatter)
		default:
			return fmt.Errorf("unknown log-format %q", context.GlobalString("log-format"))
		}

		// Set virtcontainers logger.
		vc.SetLog(ccLog)

		return nil
	}

	// If the command returns an error, cli takes upon itself to print
	// the error on cli.ErrWriter and exit.
	// Use our own writer here to ensure the log gets sent to the right
	// location.
	cli.ErrWriter = &fatalWriter{cli.ErrWriter}

	if err := app.Run(os.Args); err != nil {
		fatal(err)
	}
}

// fatal prints the error's details exits the program.
func fatal(err error) {
	ccLog.Error(err)
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

type fatalWriter struct {
	cliErrWriter io.Writer
}

func (f *fatalWriter) Write(p []byte) (n int, err error) {
	ccLog.Error(string(p))
	return f.cliErrWriter.Write(p)
}
