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
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"text/template"

	"github.com/01org/ciao/ciao-controller/api"
	"github.com/01org/ciao/ciao-controller/types"
	"github.com/golang/glog"
)

// Item serves to represent a group of related commands
type command struct {
	SubCommands map[string]subCommand
}

// This is not used but needed to comply with subCommand interface
func (c *command) parseArgs(args []string) []string {
	return args
}

func (c *command) run(args []string) error {
	cmdName := args[0]
	subCmdName := args[1]
	subCmd := c.SubCommands[subCmdName]
	if subCmd == nil {
		c.usage(cmdName)
	}
	args = subCmd.parseArgs(args[2:])
	prepareForCommand()
	return subCmd.run(args)
}

// usage prints the available commands in an item
func (c *command) usage(name ...string) {
	fmt.Fprintf(os.Stderr, `ciao-cli: Command-line interface for the Cloud Integrated Advanced Orchestrator (CIAO)

Usage:

	ciao-cli [options] `+name[0]+` sub-command [flags]
`)

	var t = template.Must(template.New("commandTemplate").Parse(commandTemplate))
	t.Execute(os.Stderr, c)

	fmt.Fprintf(os.Stderr, `
Use "ciao-cli `+name[0]+` sub-command -help" for more information about that item.
`)
	os.Exit(2)
}

// subCommand is the interface that all cli commands should implement
type subCommand interface {
	usage(...string)
	parseArgs([]string) []string
	run([]string) error
}

var commands = map[string]subCommand{
	"instance":    instanceCommand,
	"workload":    workloadCommand,
	"tenant":      tenantCommand,
	"event":       eventCommand,
	"node":        nodeCommand,
	"trace":       traceCommand,
	"image":       imageCommand,
	"volume":      volumeCommand,
	"pool":        poolCommand,
	"external-ip": externalIPCommand,
}

var scopedToken string

const openstackComputePort = 8774
const openstackComputeVersion = "v2.1"

type action uint8

const (
	computeActionStart action = iota
	computeActionStop
)

func debugf(format string, args ...interface{}) {
	if glog.V(2) {
		glog.InfoDepth(1, fmt.Sprintf("ciao-cli DEBUG: "+format, args...))
	}
}

func infof(format string, args ...interface{}) {
	if glog.V(1) {
		glog.InfoDepth(1, fmt.Sprintf("ciao-cli INFO: "+format, args...))
	}
}

func warningf(format string, args ...interface{}) {
	glog.WarningDepth(1, fmt.Sprintf("ciao-cli WARNING: "+format, args...))
}

func errorf(format string, args ...interface{}) {
	glog.ErrorDepth(1, fmt.Sprintf("ciao-cli ERROR: "+format, args...))
}

func fatalf(format string, args ...interface{}) {
	glog.FatalDepth(1, fmt.Sprintf("ciao-cli FATAL: "+format, args...))
}

var (
	identityURL      = flag.String("identity", "", "Keystone URL")
	identityUser     = flag.String("username", "", "Openstack Service Username")
	identityPassword = flag.String("password", "", "Openstack Service Password")
	controllerURL    = flag.String("controller", "", "Controller URL")
	tenantID         = flag.String("tenant-id", "", "Tenant UUID")
	tenantName       = flag.String("tenant-name", "", "Tenant name")
	computePort      = flag.Int("computeport", openstackComputePort, "Openstack Compute API port")
	ciaoPort         = flag.Int("ciaoport", api.Port, "ciao API port")
	caCertFile       = flag.String("ca-file", "", "CA Certificate")
)

const (
	ciaoIdentityEnv    = "CIAO_IDENTITY"
	ciaoControllerEnv  = "CIAO_CONTROLLER"
	ciaoUsernameEnv    = "CIAO_USERNAME"
	ciaoPasswordEnv    = "CIAO_PASSWORD"
	ciaoComputePortEnv = "CIAO_COMPUTEPORT"
	ciaoTenantNameEnv  = "CIAO_TENANT_NAME"
	ciaoCACertFileEnv  = "CIAO_CA_CERT_FILE"
)

var caCertPool *x509.CertPool

type queryValue struct {
	name, value string
}

func dumpJSON(body interface{}) {
	switch b := body.(type) {
	case []byte:
		var dump bytes.Buffer

		json.Indent(&dump, b, "", "\t")
		dump.WriteTo(os.Stdout)
	case map[string]interface{}:
		new, err := json.MarshalIndent(b, "", "\t")
		if err == nil {
			os.Stdout.Write(new)
		}
	}

	fmt.Printf("\n")
}

func buildComputeURL(format string, args ...interface{}) string {
	prefix := fmt.Sprintf("https://%s:%d/%s/", *controllerURL, *computePort, openstackComputeVersion)
	return fmt.Sprintf(prefix+format, args...)
}

func buildCiaoURL(format string, args ...interface{}) string {
	prefix := fmt.Sprintf("https://%s:%d/", *controllerURL, *ciaoPort)
	return fmt.Sprintf(prefix+format, args...)
}

func sendHTTPRequestToken(method string, url string, values []queryValue, token string, body io.Reader, content *string) (*http.Response, error) {
	req, err := http.NewRequest(method, os.ExpandEnv(url), body)
	if err != nil {
		return nil, err
	}

	infof("Sending %s %s\n", method, url)

	if values != nil {
		v := req.URL.Query()

		for _, value := range values {
			infof("Adding URL query %s=%s\n", value.name, value.value)
			v.Add(value.name, value.value)
		}

		req.URL.RawQuery = v.Encode()
	}

	if token != "" {
		req.Header.Add("X-Auth-Token", token)
	}

	if content != nil {
		contentType := fmt.Sprintf("application/%s", *content)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Accept", contentType)
	} else if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
	}

	tlsConfig := &tls.Config{}

	if caCertPool != nil {
		tlsConfig.RootCAs = caCertPool
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		errorf("Could not send HTTP request %s\n", err)
		return nil, err
	}

	infof("Got HTTP response (status %s)\n", resp.Status)

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, errBody := ioutil.ReadAll(resp.Body)
		if errBody != nil {
			errorf("Could not read the HTTP response %s\n", errBody)
			dumpJSON(respBody)
			return resp, errBody
		}

		return resp, fmt.Errorf("HTTP Error [%d] for [%s %s]: %s", resp.StatusCode, method, url, respBody)
	}

	return resp, err
}

func sendHTTPRequest(method string, url string, values []queryValue, body io.Reader) (*http.Response, error) {
	return sendHTTPRequestToken(method, url, values, scopedToken, body, nil)
}

func unmarshalHTTPResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errorf("Could not read the HTTP response %s\n", err)
		return err
	}

	err = json.Unmarshal(body, v)
	if err != nil {
		errorf("Could not unmarshal the HTTP response %s\n", err)
		return err
	}

	if glog.V(2) {
		dumpJSON(body)
	}

	return nil
}

func sendCiaoRequest(method string, url string, values []queryValue, body io.Reader, content *string) (*http.Response, error) {
	return sendHTTPRequestToken(method, url, values, scopedToken, body, content)
}

func getRef(rel string, links []types.Link) string {
	for _, link := range links {
		if link.Rel == rel {
			return link.Href
		}
	}
	return ""
}

func getCiaoResource(name string, minVersion string) (string, error) {
	var resources []types.APILink
	var url string

	if checkPrivilege() {
		url = buildCiaoURL("")
	} else {
		url = buildCiaoURL(fmt.Sprintf("%s", *tenantID))
	}

	resp, err := sendCiaoRequest("GET", url, nil, nil, nil)
	if err != nil {
		return "", err
	}

	err = unmarshalHTTPResponse(resp, &resources)
	if err != nil {
		return "", err
	}

	for _, l := range resources {
		if l.Rel == name && l.MinVersion == minVersion {
			return l.Href, nil
		}
	}

	return "", errors.New("Supported version of resource not found")
}

func checkPrivilege() bool {
	if *tenantName == "admin" {
		return true
	}

	return false
}

func limitToString(limit int) string {
	if limit == -1 {
		return "Unlimited"
	}

	return fmt.Sprintf("%d", limit)
}

func getCiaoEnvVariables() {
	identity := os.Getenv(ciaoIdentityEnv)
	controller := os.Getenv(ciaoControllerEnv)
	username := os.Getenv(ciaoUsernameEnv)
	password := os.Getenv(ciaoPasswordEnv)
	port := os.Getenv(ciaoComputePortEnv)
	tenant := os.Getenv(ciaoTenantNameEnv)
	ca := os.Getenv(ciaoCACertFileEnv)

	infof("Ciao environment variables:\n")
	infof("\t%s:%s\n", ciaoIdentityEnv, identity)
	infof("\t%s:%s\n", ciaoControllerEnv, controller)
	infof("\t%s:%s\n", ciaoUsernameEnv, username)
	infof("\t%s:%s\n", ciaoPasswordEnv, password)
	infof("\t%s:%s\n", ciaoComputePortEnv, port)
	infof("\t%s:%s\n", ciaoTenantNameEnv, tenantName)
	infof("\t%s:%s\n", ciaoCACertFileEnv, ca)

	if identity != "" && *identityURL == "" {
		*identityURL = identity
	}

	if controller != "" && *controllerURL == "" {
		*controllerURL = controller
	}

	if username != "" && *identityUser == "" {
		*identityUser = username
	}

	if password != "" && *identityPassword == "" {
		*identityPassword = password
	}

	if port != "" && *computePort == openstackComputePort {
		*computePort, _ = strconv.Atoi(port)
	}

	if tenant != "" && *tenantName == "" {
		*tenantName = tenant
	}

	if ca != "" && *caCertFile == "" {
		*caCertFile = ca
	}
}

func checkCompulsoryOptions() {
	fatal := ""

	if *identityURL == "" {
		fatal += "Missing required identity URL\n"
	}

	if *identityUser == "" {
		fatal += "Missing required username\n"
	}

	if *identityPassword == "" {
		fatal += "Missing required password\n"
	}

	if *controllerURL == "" {
		fatal += "Missing required Ciao controller URL\n"
	}

	if fatal != "" {
		fatalf(fatal)
	}
}

func prepareForCommand() {
	var err error
	/* Load CA file if necessary */
	if *caCertFile != "" {
		caCert, err := ioutil.ReadFile(*caCertFile)
		if err != nil {
			fatalf("Unable to load requested CA certificate: %s\n", err)
		}
		caCertPool, err = x509.SystemCertPool()
		if err != nil {
			fatalf("Unable to create system certificate pool: %s\n", err)
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}

	/* If we're missing the tenant name let's try to fetch one */
	if *tenantName == "" {
		*tenantName, *tenantID, err = getTenant(*identityUser, *identityPassword, *tenantID)
		if err != nil {
			fatalf(err.Error())
		}
		warningf("Unspecified scope, using (%s, %s)", *tenantName, *tenantID)
	}

	scopedToken, *tenantID, _, err = getScopedToken(*identityUser, *identityPassword, *tenantName)
	if err != nil {
		fatalf(err.Error())
	}
}

func main() {
	var err error

	flag.Usage = usage
	flag.Parse()

	getCiaoEnvVariables()
	checkCompulsoryOptions()

	// Print usage if no arguments are given
	args := flag.Args()
	if len(args) < 1 {
		usage()
	}

	// Find command in cmdline args
	cmdName := args[0]
	cmd := commands[cmdName]
	if cmd == nil {
		usage()
	}
	if len(args) < 2 {
		cmd.usage(cmdName)
	}

	// Execute the command
	err = cmd.run(args)
	if err != nil {
		fatalf(err.Error())
	}
}

const usageTemplate1 = `ciao-cli: Command-line interface for the Cloud Integrated Advanced Orchestrator (CIAO)

Usage:

	ciao-cli [options] command sub-command [flags]

The options are:

`

const usageTemplate2 = `

The commands are:
{{range $command, $subCommand := .}}
	{{$command}}{{end}}

Use "ciao-cli command -help" for more information about that command.
`

const commandTemplate = `
The sub-commands are:
{{range $name, $cmd := .SubCommands}}
	{{$name}}{{end}}
`

func usage() {
	var t = template.Must(template.New("usageTemplate1").Parse(usageTemplate1))
	t.Execute(os.Stderr, nil)
	flag.PrintDefaults()
	t = template.Must(template.New("usageTemplate2").Parse(usageTemplate2))
	t.Execute(os.Stderr, commands)
	os.Exit(2)
}
