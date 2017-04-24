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
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func fcntl(fd int, cmd int, arg int) (val int, err error) {
	r0, _, errno := syscall.Syscall(
		syscall.SYS_FCNTL, uintptr(fd), uintptr(cmd), uintptr(arg))
	val = int(r0)
	if errno != 0 {
		err = errno
	}
	return
}

// FdLeakDetector detects fd leaks by letting its user take snapshots of the
// list of opened fds at key points in the profiled program and compare two
// snapshots.
type FdLeakDetector struct {
}

type FdInfo struct {
	Fd          int
	Flags       int
	CloseOnExec bool
	Text        string
}

func (info *FdInfo) dump(w io.Writer) {
	flags := make([]string, 5)

	if info.CloseOnExec {
		flags = append(flags, "cloexec")
	}

	// file status
	if info.Flags&syscall.O_APPEND != 0 {
		flags = append(flags, "append")
	}
	if info.Flags&syscall.O_NONBLOCK != 0 {
		flags = append(flags, "nonblock")
	}

	// acc mode
	if info.Flags&syscall.O_RDONLY != 0 {
		flags = append(flags, "read-only")
	}
	if info.Flags&syscall.O_RDWR != 0 {
		flags = append(flags, "read-write")
	}
	if info.Flags&syscall.O_WRONLY != 0 {
		flags = append(flags, "write-only")
	}
	if info.Flags&syscall.O_DSYNC != 0 {
		flags = append(flags, "dsync")
	}
	if info.Flags&syscall.O_RSYNC != 0 {
		flags = append(flags, "rsync")
	}
	if info.Flags&syscall.O_SYNC != 0 {
		flags = append(flags, "sync")
	}

	fmt.Fprintf(w, "  %d: %s (%s)\n", info.Fd, info.Text,
		strings.Join(flags, ""))
}

func (info *FdInfo) equal(other *FdInfo) bool {
	return reflect.DeepEqual(info, other)
}

type FdSnapshot struct {
	Fds []FdInfo
}

// ByFd implements sort.Interface for []FdInfo based on the Fd field.
type ByFd []FdInfo

func (a ByFd) Len() int           { return len(a) }
func (a ByFd) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByFd) Less(i, j int) bool { return a[i].Fd < a[j].Fd }

func newFdSnapshot() *FdSnapshot {
	return &FdSnapshot{}
}

func (snap *FdSnapshot) FdInfo(i int) *FdInfo {
	return &snap.Fds[i]
}

func (snap *FdSnapshot) dump(w io.Writer) {
	for _, info := range snap.Fds {
		info.dump(w)
	}
}

// NewFdLeadDetector creates a new FdLeakDetector
func NewFdLeadDetector() *FdLeakDetector {
	return &FdLeakDetector{}
}

const selfFdPath = "/proc/self/fd"

// Snapshot captures the list of opened file descriptors of the current
// process
func (d *FdLeakDetector) Snapshot() (snap *FdSnapshot, err error) {
	root, err := os.Open(selfFdPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	infos, err := root.Readdir(-1)
	if err != nil {
		return nil, err
	}

	snap = newFdSnapshot()
	for _, info := range infos {
		fd, err := strconv.Atoi(info.Name())
		if err != nil {
			return nil, err
		}

		// Don't capture the fd used to list /proc/self/fd
		if fd == int(root.Fd()) {
			continue
		}

		flags, err := fcntl(fd, syscall.F_GETFL, 0)
		if err != nil {
			return nil, err
		}

		fdFlags, err := fcntl(fd, syscall.F_GETFD, 0)
		if err != nil {
			return nil, err
		}

		// readlink on /proc/self/fd gives nice textual information
		// about the fd
		rl, err := os.Readlink(fmt.Sprintf("%s/%d", selfFdPath, fd))
		if err != nil {
			return nil, err
		}

		snap.Fds = append(snap.Fds, FdInfo{
			Fd:          fd,
			Flags:       flags,
			CloseOnExec: fdFlags&syscall.FD_CLOEXEC != 0,
			Text:        rl,
		})
	}

	sort.Sort(ByFd(snap.Fds))

	return
}

// Compare writes to w the differences between the a and b snaphots.
// true will be returned if the two snapshots are equal, false otherwise.
func (d *FdLeakDetector) Compare(w io.Writer, a, b *FdSnapshot) bool {
	if a == nil || b == nil {
		return a == b
	}

	equal := true

	i := 0
	j := 0

	for i < len(a.Fds) && j < len(b.Fds) {
		aInfo := a.FdInfo(i)
		bInfo := b.FdInfo(j)

		if aInfo.Fd == bInfo.Fd {
			// File descriptor found in both snapshots
			equal = equal && aInfo.equal(bInfo)
			i++
			j++

			if !equal {
				fmt.Fprintf(w, "- fd %d\n", aInfo.Fd)
				aInfo.dump(w)
				fmt.Fprintf(w, "+ fd %d\n", bInfo.Fd)
				bInfo.dump(w)
			}
		} else if aInfo.Fd < bInfo.Fd {
			// File descriptor present in a but not in b
			equal = false
			i++

			fmt.Fprintf(w, "- fd %d\n", aInfo.Fd)
			aInfo.dump(w)
		} else {
			// File descriptor present in b but not in a
			equal = false
			j++

			fmt.Fprintf(w, "+ fd %d\n", bInfo.Fd)
			bInfo.dump(w)
		}
	}

	for ; i < len(a.Fds); i++ {
		// File descriptor present in a but not in b
		equal = false
		aInfo := a.FdInfo(i)

		fmt.Fprintf(w, "- fd %d\n", aInfo.Fd)
		aInfo.dump(w)
	}

	for ; j < len(b.Fds); j++ {
		// File descriptor present in b but not in a
		equal = false
		bInfo := b.FdInfo(j)

		fmt.Fprintf(w, "+ fd %d\n", bInfo.Fd)
		bInfo.dump(w)
	}

	return equal
}

func TestFdDetectorNoLeak(t *testing.T) {
	detector := NewFdLeadDetector()

	old, err := detector.Snapshot()
	if err != nil {
		t.Error(err)
	}

	new, err := detector.Snapshot()
	if err != nil {
		t.Error(err)
	}

	buffer := bytes.NewBuffer(nil)
	equal := detector.Compare(buffer, old, new)
	if buffer.Len() != 0 {
		fmt.Print(buffer.String())
		t.Fatal()
	}
	if !equal {
		fmt.Print(buffer.String())
		t.Fatal()
	}
}

func TestFdDetectorLeak(t *testing.T) {
	detector := NewFdLeadDetector()

	old, err := detector.Snapshot()
	if err != nil {
		t.Error(err)
	}

	_, err = os.Open("/dev/null")
	if err != nil {
		t.Error(err)
	}

	new, err := detector.Snapshot()
	if err != nil {
		t.Error(err)
	}

	buffer := bytes.NewBuffer(nil)
	equal := detector.Compare(buffer, old, new)
	if equal {
		fmt.Print(buffer.String())
		t.Fatal()
	}
}

func TestFdDetectorCompare(t *testing.T) {
	tests := []struct {
		old, new *FdSnapshot
		equal    bool
	}{
		// handle nil arguments
		{old: nil, new: nil, equal: true},
		{old: &FdSnapshot{}, new: nil, equal: false},
		{old: nil, new: &FdSnapshot{}, equal: false},
		// Same fds
		{
			old: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDWR,
					},
					{
						Fd:    1,
						Flags: syscall.O_RDWR,
						Text:  "/foo",
					},
				},
			},
			new: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDWR,
					},
					{
						Fd:    1,
						Flags: syscall.O_RDWR,
						Text:  "/foo",
					},
				},
			},
			equal: true,
		},

		// Same fd number, different flags
		{
			old: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDWR,
					},
					{
						Fd:    1,
						Flags: syscall.O_RDWR,
						Text:  "/foo",
					},
				},
			},
			new: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDWR,
					},
					{
						Fd:    1,
						Flags: syscall.O_RDONLY,
						Text:  "/foo",
					},
				},
			},
			equal: false,
		},

		// Same fd number, different close on exec status
		{
			old: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDWR,
					},
					{
						Fd:          1,
						Flags:       syscall.O_RDWR,
						CloseOnExec: true,
						Text:        "/foo",
					},
				},
			},
			new: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDWR,
					},
					{
						Fd:    1,
						Flags: syscall.O_RDWR,
						Text:  "/foo",
					},
				},
			},
			equal: false,
		},

		// Same fd number, different readlink
		{
			old: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDWR,
					},
					{
						Fd:    1,
						Flags: syscall.O_RDWR,
						Text:  "/foo",
					},
				},
			},
			new: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDWR,
					},
					{
						Fd:    1,
						Flags: syscall.O_RDWR,
						Text:  "/bar",
					},
				},
			},
			equal: false,
		},

		// old has more fds
		{
			old: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDWR,
					},
					{
						Fd:    1,
						Flags: syscall.O_RDWR,
						Text:  "/foo",
					},
				},
			},
			new: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDWR,
					},
				},
			},
			equal: false,
		},

		// new has more fds
		{
			old: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDWR,
					},
				},
			},
			new: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDWR,
					},
					{
						Fd:    1,
						Flags: syscall.O_RDWR,
						Text:  "/foo",
					},
				},
			},
			equal: false,
		},

		// Same number of fds in the two snapshots, different fds at
		// index 0 but same fds at index 1. We used to have a bug
		// saying the two snapshots were identical.
		{
			old: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDONLY,
					},
					{
						Fd:    1,
						Flags: syscall.O_RDWR,
						Text:  "/foo",
					},
				},
			},
			new: &FdSnapshot{
				Fds: []FdInfo{
					{
						Fd:    0,
						Flags: syscall.O_RDWR,
					},
					{
						Fd:    1,
						Flags: syscall.O_RDWR,
						Text:  "/foo",
					},
				},
			},
			equal: false,
		},
	}

	detector := NewFdLeadDetector()

	for i, test := range tests {
		buffer := bytes.NewBuffer(nil)

		equal := detector.Compare(buffer, test.old, test.new)
		if equal != test.equal {
			test.old.dump(os.Stderr)
			test.new.dump(os.Stderr)
			fmt.Print(buffer.String())
			t.Fatal(fmt.Sprintf("Failed test #%d", i))
		}
	}
}
