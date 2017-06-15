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

package gloginterface

import (
	"fmt"

	"github.com/golang/glog"
)

// CiaoGlogLogger is a type that makes use of glog for the CiaoLog interface
type CiaoGlogLogger struct{}

// V returns true if the given argument is less than or equal
// to glog's verbosity level.
func (l CiaoGlogLogger) V(level int32) bool {
	return bool(glog.V(glog.Level(level)))
}

// Infof writes informational output to glog.
func (l CiaoGlogLogger) Infof(format string, v ...interface{}) {
	glog.InfoDepth(2, fmt.Sprintf(format, v...))
}

// Warningf writes warning output to glog.
func (l CiaoGlogLogger) Warningf(format string, v ...interface{}) {
	glog.WarningDepth(2, fmt.Sprintf(format, v...))
}

// Errorf writes error output to glog.
func (l CiaoGlogLogger) Errorf(format string, v ...interface{}) {
	glog.ErrorDepth(2, fmt.Sprintf(format, v...))
}
