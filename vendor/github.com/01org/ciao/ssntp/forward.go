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
	"sync"
)

// ForwardDecision tells SSNTP how it should forward a frame.
// Callers set that value as part of the ForwardDestination
// structure.
type ForwardDecision uint8

const (
	// Forward the frame. The recipients are defined by the ForwardDecision
	// UUIDs field.
	Forward ForwardDecision = iota

	// Discard the frame. The frame will be discarded by SSNTP.
	Discard

	// Queue the frame. SSNTP will queue the frame and the caller will have to call
	// into the SSNTP Server queueing API to fetch it back.
	Queue
)

// ForwardDestination is returned by the forwading interfaces
// and allows the interface implementer to let SSNTP know what
// to do next with a received frame.
// The interface implementer needs to specify if the frame
// should be forwarded, discarded or queued (Decision).
// If the implementer decision is to forward the frame, it
// should also provide a list of recipients to forward it to (UUIDs)
type ForwardDestination struct {
	decision       ForwardDecision
	recipientUUIDs []string
}

// Decision is a simple accessor for the ForwardDecision.decision field
func (d *ForwardDestination) Decision() ForwardDecision {
	return d.decision
}

// Recipients is a simple accessor for the ForwardDecision.recipientUUIDs field
func (d *ForwardDestination) Recipients() []string {
	return d.recipientUUIDs
}

// AddRecipient adds a recipient to a ForwardDestination structure.
// AddRecipient implicitly sets the forwarding decision to Forward
// since adding a recipient means the frame must be forwarded.
func (d *ForwardDestination) AddRecipient(uuid string) {
	d.decision = Forward
	d.recipientUUIDs = append(d.recipientUUIDs, uuid)
}

// SetDecision is a helper for setting the ForwardDestination Decision field.
func (d *ForwardDestination) SetDecision(decision ForwardDecision) {
	d.decision = decision
}

// CommandForwarder is the SSNTP Command forwarding interface.
// The uuid argument is the sender's UUID.
type CommandForwarder interface {
	CommandForward(uuid string, command Command, frame *Frame) ForwardDestination
}

// StatusForwarder is the SSNTP Status forwarding interface.
// The uuid argument is the sender's UUID.
type StatusForwarder interface {
	StatusForward(uuid string, status Status, frame *Frame) ForwardDestination
}

// ErrorForwarder is the SSNTP Error forwarding interface.
// The uuid argument is the sender's UUID.
type ErrorForwarder interface {
	ErrorForward(uuid string, error Error, frame *Frame) ForwardDestination
}

// EventForwarder is the SSNTP Event forwarding interface.
// The uuid argument is the sender's UUID.
type EventForwarder interface {
	EventForward(uuid string, event Event, frame *Frame) ForwardDestination
}

// FrameForwardRule defines a forwarding rule for a SSNTP frame.
// The rule creator can either choose to forward this frame to
// all clients playing a specified SSNTP role (Dest), or can return
// a forwarding decision back to SSNTP depending on the frame payload (*Forwarder).
// If a frame forwarder interface implementation is provided, the
// Dest field will be ignored.
type FrameForwardRule struct {
	// Operand is the SSNTP frame operand to which this rule applies.
	Operand interface{}

	// A frame which operand is Operand will be forwarded to all
	// SSNTP clients playing the Dest SSNTP role.
	// This field is ignored if a forwarding interface is provided.
	Dest Role

	// The SSNTP Command forwarding interface implementation for this SSNTP frame.
	CommandForward CommandForwarder

	// The SSNTP Status forwarding interface implementation for this SSNTP frame.
	StatusForward StatusForwarder

	// The SSNTP Error forwarding interface implementation for this SSNTP frame.
	ErrorForward ErrorForwarder

	// The SSNTP Event forwarding interface implementation for this SSNTP frame.
	EventForward EventForwarder
}

type frameForward struct {
	forwardRules       []FrameForwardRule
	forwardMutex       sync.RWMutex
	forwardCommandDest map[Command][]*session
	forwardErrorDest   map[Error][]*session
	forwardEventDest   map[Event][]*session
	forwardStatusDest  map[Status][]*session

	forwardCommandFunc map[Command]CommandForwarder
	forwardStatusFunc  map[Status]StatusForwarder
	forwardErrorFunc   map[Error]ErrorForwarder
	forwardEventFunc   map[Event]EventForwarder
}

func (f *frameForward) init(rules []FrameForwardRule) {
	/* TODO Validate rules, e.g. look for duplicates */
	f.forwardCommandDest = make(map[Command][]*session)
	f.forwardErrorDest = make(map[Error][]*session)
	f.forwardEventDest = make(map[Event][]*session)
	f.forwardStatusDest = make(map[Status][]*session)
	f.forwardCommandFunc = make(map[Command]CommandForwarder)
	f.forwardStatusFunc = make(map[Status]StatusForwarder)
	f.forwardErrorFunc = make(map[Error]ErrorForwarder)
	f.forwardEventFunc = make(map[Event]EventForwarder)

	f.forwardMutex.Lock()

	for _, r := range rules {
		switch op := r.Operand.(type) {
		case Command:
			if r.CommandForward != nil {
				f.forwardCommandFunc[op] = r.CommandForward
			}
		case Status:
			if r.StatusForward != nil {
				f.forwardStatusFunc[op] = r.StatusForward
			}
		case Error:
			if r.ErrorForward != nil {
				f.forwardErrorFunc[op] = r.ErrorForward
			}
		case Event:
			if r.EventForward != nil {
				f.forwardEventFunc[op] = r.EventForward
			}
		}
	}

	f.forwardMutex.Unlock()
}

func (f *frameForward) addForwardDestination(session *session) {
	f.forwardMutex.Lock()

	for _, r := range f.forwardRules {
		if r.Dest == UNKNOWN {
			continue
		}

		switch op := r.Operand.(type) {
		case Command:
			if session.destRole.HasRole(r.Dest) {
				f.forwardCommandDest[op] = append(f.forwardCommandDest[op], session)
			}
		case Status:
			if session.destRole.HasRole(r.Dest) {
				f.forwardStatusDest[op] = append(f.forwardStatusDest[op], session)
			}
		case Error:
			if session.destRole.HasRole(r.Dest) {
				f.forwardErrorDest[op] = append(f.forwardErrorDest[op], session)
			}
		case Event:
			if session.destRole.HasRole(r.Dest) {
				f.forwardEventDest[op] = append(f.forwardEventDest[op], session)
			}
		}
	}

	f.forwardMutex.Unlock()
}

func (f *frameForward) deleteForwardDestination(dest *session) {
	var sessions []*session

	f.forwardMutex.Lock()

	for _, r := range f.forwardRules {
		switch op := r.Operand.(type) {
		case Command:
			sessions = f.forwardCommandDest[op]
			for i, s := range sessions {
				if s != dest {
					continue
				}

				f.forwardCommandDest[op] = append(sessions[:i], sessions[i+1:]...)
				break
			}
		case Status:
			sessions = f.forwardStatusDest[op]
			for i, s := range sessions {
				if s != dest {
					continue
				}

				f.forwardStatusDest[op] = append(sessions[:i], sessions[i+1:]...)
				break
			}
		case Error:
			sessions = f.forwardErrorDest[op]
			for i, s := range sessions {
				if s != dest {
					continue
				}

				f.forwardErrorDest[op] = append(sessions[:i], sessions[i+1:]...)
				break
			}
		case Event:
			sessions = f.forwardEventDest[op]
			for i, s := range sessions {
				if s != dest {
					continue
				}

				f.forwardEventDest[op] = append(sessions[:i], sessions[i+1:]...)
				break
			}
		}
	}

	f.forwardMutex.Unlock()
}

func forwardDestination(destination ForwardDestination, server *Server, frame *Frame) {
	/* TODO Handle queueing */
	if destination.decision == Discard || destination.recipientUUIDs == nil {
		return
	}

	server.sessionMutex.RLock()
	for _, uuid := range destination.recipientUUIDs {
		session := server.sessions[uuid]
		if session == nil {
			continue
		}

		session.Write(frame)
	}
	server.sessionMutex.RUnlock()
}

func commandForward(uuid string, f CommandForwarder, cmd Command, server *Server, frame *Frame) {
	dest := f.CommandForward(uuid, cmd, frame)

	forwardDestination(dest, server, frame)
}

func statusForward(uuid string, f StatusForwarder, status Status, server *Server, frame *Frame) {
	dest := f.StatusForward(uuid, status, frame)

	forwardDestination(dest, server, frame)
}

func errorForward(uuid string, f ErrorForwarder, error Error, server *Server, frame *Frame) {
	dest := f.ErrorForward(uuid, error, frame)

	forwardDestination(dest, server, frame)
}

func eventForward(uuid string, f EventForwarder, event Event, server *Server, frame *Frame) {
	dest := f.EventForward(uuid, event, frame)

	forwardDestination(dest, server, frame)
}

func (f *frameForward) forwardFrame(server *Server, source *session, operand interface{}, frame *Frame) {
	var sessions []*session
	src := source.dest.String()

	f.forwardMutex.RLock()
	defer f.forwardMutex.RUnlock()

	switch op := operand.(type) {
	case Command:
		forwarder := f.forwardCommandFunc[op]
		if forwarder != nil {
			go commandForward(src, forwarder, op, server, frame)
			return
		}

		sessions = f.forwardCommandDest[op]
	case Status:
		forwarder := f.forwardStatusFunc[op]
		if forwarder != nil {
			go statusForward(src, forwarder, op, server, frame)
			return
		}

		sessions = f.forwardStatusDest[op]
	case Error:
		forwarder := f.forwardErrorFunc[op]
		if forwarder != nil {
			go errorForward(src, forwarder, op, server, frame)
			return
		}

		sessions = f.forwardErrorDest[op]
	case Event:
		forwarder := f.forwardEventFunc[op]
		if forwarder != nil {
			go eventForward(src, forwarder, op, server, frame)
			return
		}

		sessions = f.forwardEventDest[op]
	default:
		sessions = nil
	}

	if sessions == nil {
		return
	}

	for _, s := range sessions {
		if s == source {
			continue
		}
		s.Write(frame)
	}
}
