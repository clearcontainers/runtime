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

package testutil_test

import (
	"testing"

	"github.com/01org/ciao/ssntp"
	. "github.com/01org/ciao/testutil"
)

func TestRoleToTestCert(t *testing.T) {
	var roleToCertTests = []struct {
		role ssntp.Role
		cert string
	}{
		{ssntp.SCHEDULER, TestCertScheduler},
		{ssntp.SERVER, TestCertServer},
		{ssntp.AGENT, TestCertAgent},
		{ssntp.Controller, TestCertController},
		{ssntp.CNCIAGENT, TestCertCNCIAgent},
		{ssntp.NETAGENT, TestCertNetAgent},
		{ssntp.AGENT | ssntp.NETAGENT, TestCertAgentNetAgent},
		{ssntp.UNKNOWN, TestCertUnknown},
	}

	for _, test := range roleToCertTests {
		cert := RoleToTestCert(test.role)
		if cert != test.cert {
			t.Errorf("expected:\n%s\ngot:\n%s\n", test.cert, cert)
		}
	}
}

func TestMakeTestCerts(t *testing.T) {
	err := MakeTestCerts()
	if err != nil {
		t.Errorf("Failed to create test certificates: %v", err)
	}
	RemoveTestCerts()
}
