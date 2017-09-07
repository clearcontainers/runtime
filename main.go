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

const (
	// name holds the name of this program
	name    = "cc-runtime"
	project = "Intel® Clear Containers"
)

// specConfig is the name of the file holding the containers configuration
const specConfig = "config.json"

const usage = project + ` runtime

cc-runtime is a command line program for running applications packaged
according to the Open Container Initiative (OCI).`

const notes = `
NOTES:

- Commands starting "cc-" and options starting "--cc-" are ` + project + ` extensions.

`

var ccLog = logrus.New()

// concrete virtcontainer implementation
var virtcontainersImpl = &vc.VCImpl{}

// vci is used to access a particular virtcontainers implementation.
// Normally, it refers to the official package, but is re-assigned in
// the tests to allow virtcontainers to be mocked.
var vci vc.VC = virtcontainersImpl

// defaultOutputFile is the default output file to write the gathered
// information to.
var defaultOutputFile = os.Stdout

// defaultErrorFile is the default output file to write error
// messages to.
var defaultErrorFile = os.Stderr

func beforeSubcommands(context *cli.Context) error {
	if userWantsUsage(context) || (context.NArg() == 1 && (context.Args()[0] == "cc-check")) {
		// No setup required if the user just
		// wants to see the usage statement or are
		// running a command that does not manipulate
		// containers.
		return nil
	}

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
	vci.SetLogger(ccLog)

	ignoreLogging := false
	if context.NArg() == 1 && context.Args()[0] == "cc-env" {
		// "cc-env" should simply report the logging setup
		ignoreLogging = true
	}

	configFile, logfilePath, runtimeConfig, err := loadConfiguration(context.GlobalString("cc-config"), ignoreLogging)
	if err != nil {
		fatal(err)
	}

	ccLog.Infof("%v (version %v, commit %v) called as: %v", name, version, commit, context.Args())
	ccLog.Infof("Using configuration file %q", configFile)

	// make the data accessible to the sub-commands.
	context.App.Metadata = map[string]interface{}{
		"runtimeConfig": runtimeConfig,
		"configFile":    configFile,
		"logfilePath":   logfilePath,
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = name
	app.Usage = usage

	cli.AppHelpTemplate = fmt.Sprintf(`%s%s`, cli.AppHelpTemplate, notes)

	v := make([]string, 0, 3)
	if version != "" {
		v = append(v, name+"  : "+version)
	}
	if commit != "" {
		v = append(v, "   commit   : "+commit)
	}
	v = append(v, "   OCI specs: "+specs.Version)
	app.Version = strings.Join(v, "\n")

	// Override the default function to display version details to
	// ensure the "--version" option and "version" command are identical.
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(c.App.Version)
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "cc-config",
			Usage: project + " config file path",
		},
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
			Value: defaultRootDirectory,
			Usage: "root directory for storage of container state (this should be located in tmpfs)",
		},
	}

	app.Commands = []cli.Command{
		checkCLICommand,
		envCLICommand,
		createCLICommand,
		deleteCLICommand,
		execCLICommand,
		killCLICommand,
		listCLICommand,
		runCLICommand,
		pauseCLICommand,
		resumeCLICommand,
		startCLICommand,
		stateCLICommand,
		versionCLICommand,
	}

	app.Before = beforeSubcommands
	// If the command returns an error, cli takes upon itself to print
	// the error on cli.ErrWriter and exit.
	// Use our own writer here to ensure the log gets sent to the right
	// location.
	cli.ErrWriter = &fatalWriter{cli.ErrWriter}

	if err := app.Run(os.Args); err != nil {
		fatal(err)
	}
}

// userWantsUsage determines if the user only wishes to see the usage
// statement.
func userWantsUsage(context *cli.Context) bool {
	if context.NArg() == 0 {
		return true
	}

	if context.NArg() == 1 && (context.Args()[0] == "help" || context.Args()[0] == "version") {
		return true
	}

	if context.NArg() >= 2 && (context.Args()[1] == "-h" || context.Args()[1] == "--help") {
		return true
	}

	return false
}

// fatal prints the error's details exits the program.
func fatal(err error) {
	ccLog.Error(err)
	fmt.Fprintln(defaultErrorFile, err)
	exit(1)
}

type fatalWriter struct {
	cliErrWriter io.Writer
}

func (f *fatalWriter) Write(p []byte) (n int, err error) {
	// Ensure error is logged before displaying to the user
	ccLog.Error(string(p))
	return f.cliErrWriter.Write(p)
}
