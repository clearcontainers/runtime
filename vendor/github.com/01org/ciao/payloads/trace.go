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

package payloads

// SSNTPNode contains information about a node through which a SSNTP frame
// has passed.
type SSNTPNode struct {
	SSNTPUUID   string `yaml:"ssntp_node_uuid"`
	SSNTPRole   string `yaml:"ssntp_role"`
	TxTimestamp string `yaml:"tx_timestamp"`
	RxTimestamp string `yaml:"rx_timestamp"`
}

// FrameTrace captures trace information for an SSNTP frame
// as it makes its way through a SSNTP cluster.
type FrameTrace struct {
	Label          string `yaml:"label"`
	Type           string `yaml:"type"`
	Operand        string `yaml:"operand"`
	StartTimestamp string `yaml:"start_timestamp"`
	EndTimestamp   string `yaml:"end_timestamp"`
	Nodes          []SSNTPNode
}

// Trace represents the unmarshalled version of the contents of an SSNTP
// ssntp.TraceReport event.  The structure contains tracing information
// for an SSNTP frame.
type Trace struct {
	Frames []FrameTrace
}
