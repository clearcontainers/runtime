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
	"io/ioutil"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type testData struct {
	network     string
	raddr       string
	expectError bool
}

func init() {
	// Ensure all log levels are logged
	ccLog.Logger.Level = logrus.DebugLevel

	// Discard log output
	ccLog.Logger.Out = ioutil.Discard
}

func TestHandleSystemLog(t *testing.T) {
	assert := assert.New(t)

	data := []testData{
		{"invalid-net-type", "999.999.999.999", true},
		{"invalid net-type", "a a ", true},
		{"invalid-net-type", ".", true},
		{"moo", "999.999.999.999", true},
		{"moo", "999.999.999.999:99999999999999999", true},
		{"qwerty", "uiop:ftw!", true},
		{"", "", false},
	}

	for _, d := range data {
		err := handleSystemLog(d.network, d.raddr)
		if d.expectError {
			assert.Error(err, fmt.Sprintf("%+v", d))
		} else {
			assert.NoError(err, fmt.Sprintf("%+v", d))
		}
	}
}
