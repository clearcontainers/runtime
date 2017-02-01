//
// Copyright © 2016 Intel Corporation
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

package osprepare

import (
	"testing"
)

func TestGetDistro(t *testing.T) {
	if pathExists("/usr/share/clear/bundles") == false {
		t.Skip("Unsupported test distro")
	}
	d := getDistro()
	if d == nil {
		t.Fatal("Cannot get known distro object")
	}
	if d.getID() == "" {
		t.Fatal("Invalid ID for distro")
	}
}

type ospTestLogger struct{}

func (l ospTestLogger) V(level int32) bool {
	return true
}

var info []string
var warning []string
var error []string

func (l ospTestLogger) Infof(format string, v ...interface{}) {
	info = append(info, format)
}

func (l ospTestLogger) Warningf(format string, v ...interface{}) {
	warning = append(warning, format)
}

func (l ospTestLogger) Errorf(format string, v ...interface{}) {
	error = append(error, format)
}

func TestSudoFormatCommandLogging(t *testing.T) {
	info = []string{}
	if getDistro() == nil {
		t.Skip("Unsupported test distro")
	}
	l := ospTestLogger{}

	sudoFormatCommand("echo -n foo\nbar", []string{}, l)

	if info[0] != "foo" && info[1] != "bar%!(EXTRA string=)" {
		t.Fatal("Incorrect log message received")
	}
}

func TestSudoFormatCommandBadCommandReturn(t *testing.T) {
	error = []string{}
	if getDistro() == nil {
		t.Skip("Unsupported test distro")
	}
	l := ospTestLogger{}

	if sudoFormatCommand("false", []string{}, l) {
		t.Fatal("Error return code not detected")
	}
	if len(error) != 1 && error[0] != "Error running command: %s" {
		t.Fatal("Incorrect log message received")
	}
}
