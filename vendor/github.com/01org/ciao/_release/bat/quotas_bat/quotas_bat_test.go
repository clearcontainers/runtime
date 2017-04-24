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

// Package quotasbat is a placeholder package for the basic BAT tests.
package quotasbat

import (
	"context"
	"testing"
	"time"

	"github.com/01org/ciao/bat"
)

const standardTimeout = time.Second * 300

func findQuota(qds []bat.QuotaDetails, name string) *bat.QuotaDetails {
	for i := range qds {
		if qds[i].Name == name {
			return &qds[i]
		}
	}
	return nil
}

func restoreQuotas(ctx context.Context, tenantID string, origQuotas []bat.QuotaDetails, currentQuotas []bat.QuotaDetails) error {
	for i := range currentQuotas {
		qd := findQuota(origQuotas, currentQuotas[i].Name)
		if qd != nil && qd.Value != currentQuotas[i].Value {
			err := bat.UpdateQuota(ctx, "", tenantID, qd.Name, qd.Value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Test getting and setting quotas
//
// Tests retrieving and setting quotas.
//
// Gets the current quotas, sets several, gets them again checks they've
// changed, restores the original and checks the restoration.
func TestQuotas(t *testing.T) {
	qn := "tenant-vcpu-quota"
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	tenants, err := bat.GetUserTenants(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(tenants) < 1 {
		t.Fatal("Expected user to have access to at least one tenant")
	}

	tenantID := tenants[0].ID
	origQuotas, err := bat.ListQuotas(ctx, tenantID, "")
	if err != nil {
		t.Fatal(err)
	}

	err = bat.UpdateQuota(ctx, "", tenantID, qn, "10")
	if err != nil {
		t.Fatal(err)
	}

	updatedQuotas, err := bat.ListQuotas(ctx, tenantID, "")
	if err != nil {
		t.Error(err)
	}

	qd := findQuota(updatedQuotas, qn)
	if qd.Value != "10" {
		t.Error("Quota not expected value")
	}

	err = restoreQuotas(ctx, tenantID, origQuotas, updatedQuotas)
	if err != nil {
		t.Fatal(err)
	}
}
