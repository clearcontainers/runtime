//
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
//

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/01org/ciao/ciao-controller/api"
	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/templateutils"
)

var quotasCommand = &command{
	SubCommands: map[string]subCommand{
		"update": new(quotasUpdateCommand),
		"list":   new(quotasListCommand),
	},
}

type quotasUpdateCommand struct {
	Flag     flag.FlagSet
	name     string
	value    string
	tenantID string
}

func getCiaoQuotasResource() (string, error) {
	return getCiaoResource("tenants", api.TenantsV1)
}

func (cmd *quotasUpdateCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] quotas update [flags]

Updates the quota entry for the supplied tenant

The update flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *quotasUpdateCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Name of quota or limit")
	cmd.Flag.StringVar(&cmd.value, "value", "", "Value of quota or limit")
	cmd.Flag.StringVar(&cmd.tenantID, "for-tenant", "", "Tenant to update quota for")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *quotasUpdateCommand) run(args []string) error {
	if !checkPrivilege() {
		fatalf("Updating quotas is only available for privileged users")
	}

	if cmd.name == "" {
		errorf("Missing required -name parameter")
		cmd.usage()
	}

	if cmd.value == "" {
		errorf("Missing required -value parameter")
		cmd.usage()
	}

	if cmd.tenantID == "" {
		errorf("Missing required -for-tenant parameter")
		cmd.usage()
	}
	var v int
	if cmd.value == "unlimited" {
		v = -1
	} else {
		var err error
		v, err = strconv.Atoi(cmd.value)
		if err != nil {
			fatalf(err.Error())
		}
	}

	req := types.QuotaUpdateRequest{
		Quotas: []types.QuotaDetails{{
			Name:  cmd.name,
			Value: v,
		}},
	}

	b, err := json.Marshal(req)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(b)

	url, err := getCiaoQuotasResource()
	if err != nil {
		fatalf(err.Error())
	}

	ver := api.TenantsV1

	url = fmt.Sprintf("%s/%s/quotas", url, cmd.tenantID)
	resp, err := sendCiaoRequest("PUT", url, nil, body, &ver)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusCreated {
		fatalf("Update quotas failed: %s", resp.Status)
	}

	fmt.Printf("Update quotas succeeded\n")

	return nil
}

type quotasListCommand struct {
	Flag     flag.FlagSet
	template string
	tenantID string
}

func (cmd *quotasListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] quotas list [flags]

Show all quotas for current tenant or supplied tenant if admin

The list flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a 

%s`,
		templateutils.GenerateUsageUndecorated([]types.QuotaDetails{}))
	fmt.Fprintln(os.Stderr, templateutils.TemplateFunctionHelp(nil))

	os.Exit(2)
}

func (cmd *quotasListCommand) parseArgs(args []string) []string {
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.StringVar(&cmd.tenantID, "for-tenant", "", "Tenant to get quotas for")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

// change this command to return different output depending
// on the privilege level of user. Check privilege, then
// if not privileged, build non-privileged URL.
func (cmd *quotasListCommand) run(args []string) error {
	url, err := getCiaoQuotasResource()
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.tenantID != "" {
		if !checkPrivilege() {
			fatalf("Listing quotas for other tenants is for privileged users only")
		}

		url = fmt.Sprintf("%s/%s/quotas", url, cmd.tenantID)
	} else {
		if checkPrivilege() {
			fatalf("Admin user must specify the tenant with -for-tenant")
		}

		url = fmt.Sprintf("%s/quotas", url)
	}
	ver := api.TenantsV1

	resp, err := sendCiaoRequest("GET", url, nil, nil, &ver)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		fatalf("Getting quotas failed: %s", resp.Status)
	}

	var results types.QuotaListResponse
	err = unmarshalHTTPResponse(resp, &results)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.template != "" {
		return templateutils.OutputToTemplate(os.Stdout, "quotas-list", cmd.template,
			results.Quotas, nil)
	}

	fmt.Printf("Quotas for tenant: %s\n", cmd.tenantID)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	for _, qd := range results.Quotas {
		fmt.Fprintf(w, "%s:\t", qd.Name)
		if strings.Contains(qd.Name, "quota") {
			fmt.Fprintf(w, "%d of ", qd.Usage)
		}
		if qd.Value == -1 {
			fmt.Fprint(w, "unlimited\n")
		} else {
			fmt.Fprintf(w, "%d\n", qd.Value)
		}
	}
	w.Flush()
	return nil
}
