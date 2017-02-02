/*
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
*/

package payloads_test

import (
	"testing"

	. "github.com/01org/ciao/payloads"
	"github.com/01org/ciao/testutil"
	"gopkg.in/yaml.v2"
)

func TestNodeConnectedUnmarshal(t *testing.T) {
	var nodeConnected NodeConnected

	err := yaml.Unmarshal([]byte(testutil.NodeConnectedYaml), &nodeConnected)
	if err != nil {
		t.Error(err)
	}

	if nodeConnected.Connected.NodeUUID != testutil.AgentUUID {
		t.Errorf("Wrong node UUID field [%s]", nodeConnected.Connected.NodeUUID)
	}

	if nodeConnected.Connected.NodeType != NetworkNode {
		t.Errorf("Wrong node Type field [%s]", nodeConnected.Connected.NodeType)
	}
}

func TestNodeConnectedMarshal(t *testing.T) {
	var nodeConnected NodeConnected

	nodeConnected.Connected.NodeUUID = testutil.AgentUUID
	nodeConnected.Connected.NodeType = NetworkNode

	y, err := yaml.Marshal(&nodeConnected)
	if err != nil {
		t.Error(err)
	}

	if string(y) != testutil.NodeConnectedYaml {
		t.Errorf("NodeConnected marshalling failed\n[%s]\n vs\n[%s]", string(y), testutil.NodeConnectedYaml)
	}
}
