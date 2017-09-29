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

package ssntp

import (
	"fmt"
	"time"

	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/ssntp/uuid"
)

// TraceConfig is the SSNTP tracing configuration to be used
// when calling into the client SendTraced* APIs.
type TraceConfig struct {
	// Label places a a label in the SSNTP frame sent
	// using this config.
	Label []byte

	// Start is defined by the API caller to specify when
	// operations related to that frames actually started.
	// Together with SetEndStamp, this allows for an
	// end-to-end timestamping.
	Start time.Time

	// PathTrace turns frame timestamping on or off.
	PathTrace bool
}

// Node represent an SSNTP networking node.
type Node struct {
	UUID        []byte
	Role        Role
	TxTimestamp time.Time
	RxTimestamp time.Time
}

// FrameTrace gathers all SSNTP frame tracing information,
// including frame labelling, per Node timestamping and both
// start and end timestamps as provided by the frame API callers.
type FrameTrace struct {
	Label          []byte
	StartTimestamp time.Time
	EndTimestamp   time.Time
	PathLength     uint8
	Path           []Node
}

// Frame represents an SSNTP frame structure.
type Frame struct {
	Major   uint8
	Minor   uint8
	Type    Type
	Operand uint8

	// Origin is the frame first sender and creator UUID.
	// When a SSNTP frame is forwarded by a server, the client
	// then only sees a new frame coming but it can not tell
	// who the frame creator and first sender is. This method
	// allows to fetch such information from a frame.
	Origin        uuid.UUID
	PayloadLength uint32
	Trace         *FrameTrace
	Payload       []byte
}

// ConnectFrame is the SSNTP connection frame structure.
type ConnectFrame struct {
	Major       uint8
	Minor       uint8
	Type        Type
	Operand     uint8
	Role        Role
	Source      []byte
	Destination []byte
}

// ConnectedFrame is the SSNTP connected frame structure.
type ConnectedFrame struct {
	Major         uint8
	Minor         uint8
	Type          Type
	Operand       uint8
	Role          Role
	Source        []byte
	Destination   []byte
	PayloadLength uint32
	Payload       []byte
}

const majorMask = 0x7f
const pathTraceEnabled = 1 << 7

// PathTrace tells if an SSNTP frames contains tracing information or not.
func (f Frame) PathTrace() bool {
	if f.Trace == nil {
		return false
	}

	return (f.Major & pathTraceEnabled) == pathTraceEnabled
}

func (f *Frame) setTrace(trace *TraceConfig) {
	if trace == nil || (len(trace.Label) == 0 && trace.PathTrace == false) {
		f.Major = f.Major &^ pathTraceEnabled
		return
	}

	f.Trace = &FrameTrace{Label: trace.Label}

	if trace.PathTrace == true {
		f.Major |= pathTraceEnabled
		f.Trace.StartTimestamp = trace.Start
	}
}

// GetMajor returns the SSNTP major number for the frame.
func (f Frame) GetMajor() uint8 {
	return f.Major & majorMask
}

func (f Frame) String() string {
	var node uuid.UUID
	var op string
	t := f.Type

	switch t {
	case COMMAND:
		op = (Command)(f.Operand).String()
	case STATUS:
		op = (Status)(f.Operand).String()
	case EVENT:
		op = (Event)(f.Operand).String()
	case ERROR:
		op = fmt.Sprintf("%d", f.Operand)
	}

	if f.PathTrace() == true {
		path := ""
		for i, n := range f.Trace.Path {
			ts := ""
			copy(node[:], n.UUID[:16])

			if n.RxTimestamp.IsZero() == false {
				ts = ts + fmt.Sprintf("\t\tRx %q\n", n.RxTimestamp.Format(time.StampNano))
			}

			if n.TxTimestamp.IsZero() == false {
				ts = ts + fmt.Sprintf("\t\tTx %q\n", n.TxTimestamp.Format(time.StampNano))
			}

			path = path + fmt.Sprintf("\n\t\tNode #%d\n\t\tUUID %s\n", i, node) + ts
		}

		return fmt.Sprintf("\n\tMajor %d\n\tMinor %d\n\tType %s\n\tOp %s\n\tOrigin %s\n\tPayload len %d\n\tPath %s\n",
			f.GetMajor(), f.Minor, t, op, f.Origin, f.PayloadLength, path)
	}

	return fmt.Sprintf("\n\tMajor %d\n\tMinor %d\n\tType %s\n\tOp %s\n\tOrigin %s\n\tPayload len %d\n",
		f.GetMajor(), f.Minor, t, op, f.Origin, f.PayloadLength)
}

func (f ConnectFrame) String() string {
	var dest, src uuid.UUID
	var op string
	t := f.Type

	switch t {
	case COMMAND:
		op = (Command)(f.Operand).String()
	case STATUS:
		op = (Status)(f.Operand).String()
	case ERROR:
		op = fmt.Sprintf("%d", f.Operand)
	}

	copy(src[:], f.Source[:16])
	copy(dest[:], f.Destination[:16])

	return fmt.Sprintf("\tMajor %d\n\tMinor %d\n\tType %s\n\tOp %s\n\tRole %s\n\tSource %s\n\tDestination %s\n",
		f.Major, f.Minor, (Type)(f.Type), op, &f.Role, src, dest)
}

func (f ConnectedFrame) String() string {
	var dest, src uuid.UUID
	var op string
	t := f.Type

	switch t {
	case COMMAND:
		op = (Command)(f.Operand).String()
	case STATUS:
		op = (Status)(f.Operand).String()
	case ERROR:
		op = fmt.Sprintf("%d", f.Operand)
	}

	copy(src[:], f.Source[:16])
	copy(dest[:], f.Destination[:16])

	return fmt.Sprintf("\tMajor %d\n\tMinor %d\n\tType %s\n\tOp %s\n\tRole %s\n\tSource %s\n\tDestination %s\n",
		f.Major, f.Minor, (Type)(f.Type), op, &f.Role, src, dest)
}

func (f *Frame) addPathNode(session *session) {
	if f.PathTrace() == false {
		return
	}

	node := Node{
		UUID: session.src[:],
		Role: session.srcRole,
	}

	f.Trace.Path = append(f.Trace.Path, node)
	f.Trace.PathLength++
}

// Duration returns the time spent between the first frame transmission
// and its last reception.
func (f Frame) Duration() (time.Duration, error) {
	if f.PathTrace() != true {
		return 0, fmt.Errorf("Timestamps not available")
	}

	return f.Trace.Path[f.Trace.PathLength-1].RxTimestamp.Sub(f.Trace.Path[0].TxTimestamp), nil
}

// SetEndStamp adds the final timestamp to an SSNTP frame.
// This is called by the SSNTP node that believes it's the
// last frame receiver. It provides information to build the
// complete duration of the operation related to an SSNTP frame.
func (f *Frame) SetEndStamp() {
	if f.PathTrace() != true {
		return
	}

	f.Trace.EndTimestamp = time.Now()
}

// DumpTrace builds SSNTP frame tracing data into a FrameTrace
// payload. Callers typically marshall this structure into a
// TraceReport YAML payload.
func (f Frame) DumpTrace() (*payloads.FrameTrace, error) {
	var s payloads.FrameTrace
	var node uuid.UUID

	if f.PathTrace() != true {
		return nil, fmt.Errorf("Traces not available")
	}

	s.Label = string(f.Trace.Label)
	s.StartTimestamp = f.Trace.StartTimestamp.Format(time.RFC3339Nano)
	s.EndTimestamp = f.Trace.EndTimestamp.Format(time.RFC3339Nano)
	s.Type = f.Type.String()

	switch f.Type {
	case COMMAND:
		s.Operand = (Command)(f.Operand).String()
	case STATUS:
		s.Operand = (Status)(f.Operand).String()
	case EVENT:
		s.Operand = (Event)(f.Operand).String()
	case ERROR:
		s.Operand = fmt.Sprintf("%d", f.Operand)
	}

	for _, n := range f.Trace.Path {
		copy(node[:], n.UUID[:16])
		sNode := payloads.SSNTPNode{
			SSNTPUUID: node.String(),
			SSNTPRole: n.Role.String(),
		}

		if n.TxTimestamp.IsZero() == false {
			sNode.TxTimestamp = n.TxTimestamp.Format(time.RFC3339Nano)
		}

		if n.RxTimestamp.IsZero() == false {
			sNode.RxTimestamp = n.RxTimestamp.Format(time.RFC3339Nano)
		}

		s.Nodes = append(s.Nodes, sNode)
	}

	return &s, nil
}
