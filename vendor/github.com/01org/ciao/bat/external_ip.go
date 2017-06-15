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

// ExternalIP contains information about a single external ip.
type ExternalIP struct {
	ExternalIP string `json:"external_ip"`
	InternalIP string `json:"internal_ip"`
	InstanceID string `json:"instance_id"`
	TenantID   string `json:"tenant_id"`
	PoolName   string `json:"pool_name"`
}

// CreateExternalIPPool creates a new pool for external ips.  The pool is created
// using the ciao-cli pool create command.  An error will be returned if the
// following environment are not set;  CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func CreateExternalIPPool(ctx context.Context, tenant, name string) error {
	args := []string{"pool", "create", "-name", name}
	_, err := RunCIAOCLIAsAdmin(ctx, tenant, args)
	return err
}

// AddExternalIPToPool adds an external ips to an existing pool.  The address
// is added using the ciao-cli pool add command.  An error will be returned if the
// following environment are not set;  CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func AddExternalIPToPool(ctx context.Context, tenant, name, ip string) error {
	args := []string{"pool", "add", "-name", name, ip}
	_, err := RunCIAOCLIAsAdmin(ctx, tenant, args)
	return err
}

// MapExternalIP maps an external ip from a given pool to an instance.  The
// address is mapped using the ciao-cli external-ip map command.  An error
// will be returned if the following environment are not set;  CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func MapExternalIP(ctx context.Context, tenant, pool, instance string) error {
	args := []string{"external-ip", "map", "-instance", instance, "-pool", pool}
	_, err := RunCIAOCLI(ctx, tenant, args)
	return err
}

// UnmapExternalIP unmaps an external ip from an instance.  The address is unmapped
// using the ciao-cli external-ip unmap command.  An error will be returned if the
// following environment are not set;  CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_USERNAME,
// CIAO_PASSWORD.
func UnmapExternalIP(ctx context.Context, tenant, address string) error {
	args := []string{"external-ip", "unmap", "-address", address}
	_, err := RunCIAOCLI(ctx, tenant, args)
	return err
}

// DeleteExternalIPPool deletes an external-ip pool.  The pool is deleted using
// the ciao-cli pool delete command.  An error will be returned if the
// following environment are not set;  CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func DeleteExternalIPPool(ctx context.Context, tenant, name string) error {
	args := []string{"pool", "delete", "-name", name}
	_, err := RunCIAOCLIAsAdmin(ctx, tenant, args)
	return err
}

// ListExternalIPs returns detailed information about all the external ips
// defined for the given tenant.  The information is retrieved using the
// ciao-cli external-ip list command.  An error will be returned if the
// following environment are not set;  CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func ListExternalIPs(ctx context.Context, tenant string) ([]*ExternalIP, error) {
	var externalIPs []*ExternalIP
	args := []string{"external-ip", "list", "-f", "{{tojson .}}"}
	err := RunCIAOCLIAsAdminJS(ctx, tenant, args, &externalIPs)
	if err != nil {
		return nil, err
	}

	return externalIPs, nil
}
