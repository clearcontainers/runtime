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

package clogger

// CiaoLog is a logging interface to be used by other packages to log various
// interesting pieces of information.  Rather than introduce a dependency
// on a given logging package, ciao-logger presents this interface that allows
// clients to provide their own logging type.
type CiaoLog interface {
	// V returns true if the given argument is less than or equal
	// to the implementation's defined verbosity level.
	V(int32) bool

	// Infof writes informational output to the log.  A newline will be
	// added to the output if one is not provided.
	Infof(string, ...interface{})

	// Warningf writes warning output to the log.  A newline will be
	// added to the output if one is not provided.
	Warningf(string, ...interface{})

	// Errorf writes error output to the log.  A newline will be
	// added to the output if one is not provided.
	Errorf(string, ...interface{})
}

// CiaoNullLogger is a do nothing implementation of CiaoLog
type CiaoNullLogger struct{}

// V no message is verbose
func (l CiaoNullLogger) V(level int32) bool {
	return false
}

// Infof no logging done
func (l CiaoNullLogger) Infof(format string, v ...interface{}) {
}

// Warningf no logging done
func (l CiaoNullLogger) Warningf(format string, v ...interface{}) {
}

// Errorf no logging done
func (l CiaoNullLogger) Errorf(format string, v ...interface{}) {
}
