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

package bat

import "context"

// QuotaDetails holds quota information returned by ListQuotas()
type QuotaDetails struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Usage string `json:"usage"`
}

// ListQuotas returns the quotas by calling "ciao-cli quotas list". If a
// forTenantID is specified then it is run as an admin and the quotas for the
// the specified tenant will be returned. Otherwise it will provide the quotas
// for the current tenant.
func ListQuotas(ctx context.Context, tenantID string, forTenantID string) ([]QuotaDetails, error) {
	var qds []QuotaDetails
	var err error
	if forTenantID == "" {
		args := []string{"quotas", "list", "-f", "{{tojson .}}"}
		err = RunCIAOCLIJS(ctx, tenantID, args, &qds)
	} else {
		args := []string{"quotas", "list", "-for-tenant", forTenantID, "-f", "{{tojson .}}"}
		err = RunCIAOCLIAsAdminJS(ctx, tenantID, args, &qds)
	}

	if err != nil {
		return nil, err
	}

	return qds, nil
}

// UpdateQuota updates the provided named quota for the provided tenant (using
// forTenantID) to the desired value.
func UpdateQuota(ctx context.Context, tenantID string, forTenantID string, name string, value string) error {
	args := []string{"quotas", "update", "-for-tenant", forTenantID, "-name", name, "-value", value}

	_, err := RunCIAOCLIAsAdmin(ctx, tenantID, args)

	return err
}
