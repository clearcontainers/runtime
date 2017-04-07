/*
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
*/

// Package deviceinfo contains some utility functions that return
// information about the underlying device on which the package
// runs, e.g., how many CPUs it has, how much RAM it has, etc.
package deviceinfo

import (
	"bytes"
	"os"
	"testing"
)

const memInfoContents = `
MemTotal:        1999368 kB
MemFree:         1289644 kB
MemAvailable:    1885704 kB
Buffers:           38796 kB
Cached:           543892 kB
SwapCached:            0 kB
Active:           456232 kB
Inactive:         175996 kB
Active(anon):      50128 kB
Inactive(anon):     5396 kB
Active(file):     406104 kB
Inactive(file):   170600 kB
Unevictable:           0 kB
Mlocked:               0 kB
SwapTotal:       2045948 kB
SwapFree:        2045948 kB
Dirty:                 0 kB
Writeback:             0 kB
AnonPages:         49580 kB
Mapped:            62960 kB
Shmem:              5988 kB
Slab:              55396 kB
SReclaimable:      40152 kB
SUnreclaim:        15244 kB
KernelStack:        2176 kB
PageTables:         4196 kB
NFS_Unstable:          0 kB
Bounce:                0 kB
WritebackTmp:          0 kB
CommitLimit:     3045632 kB
Committed_AS:     380776 kB
VmallocTotal:   34359738367 kB
VmallocUsed:           0 kB
VmallocChunk:          0 kB
HardwareCorrupted:     0 kB
AnonHugePages:     16384 kB
CmaTotal:              0 kB
CmaFree:               0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
DirectMap4k:       57280 kB
DirectMap2M:     1990656 kB
`

const loadAvgContents = `
1.00 0.01 0.05 1/134 23379
`

const statContents = `cpu  29164 292 87649 17177990 544 0 580 0 0 0
cpu0 29164 292 87649 17177990 544 0 580 0 0 0
intr 28478654 38 10 0 0 0 0 0 0 0 0 0 0 156 0 0 169437 0 0 0 163737 19303499 21210 29 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
ctxt 54009655
btime 1465121906
processes 55793
procs_running 1
procs_blocked 0
softirq 2742553 2 1348123 34687 170653 103600 0 45 0 0 1085443
`

// TestGetMemoryInfo tests the code that parses /proc/meminfo
//
// We call getMemoryInfo to parse a buffer that contains the contents of
// an example meminfo file.
//
// The total and available amounts of memory should be correctly computed
// by the getMemoryInfo function.
func TestGetMemoryInfo(t *testing.T) {
	buf := bytes.NewBufferString(memInfoContents)
	const expectedTotal = 1999368 / 1024
	total, available := getMemoryInfo(buf)
	if total != expectedTotal {
		t.Errorf("Bad total memory size.  Expected %d, found %d",
			expectedTotal, total)
	}

	const expectedAvail = (1289644 + 406104 + 170600) / 1024
	if available != expectedAvail {
		t.Errorf("Bad available memory size.  Expected %d, found %d",
			expectedAvail, available)

	}
}

// TestGetOnlineCPUs tests the code that parses /proc/stat
//
// We call getOnlineCPUs to parse a buffer that contains the contents of
// an example /proc/stat file.
//
// The correct number of CPUs should be returned.
func TestGetOnlineCPUs(t *testing.T) {
	buf := bytes.NewBufferString(statContents)
	const expectedCPUs = 1
	cpus := getOnlineCPUs(buf)
	if cpus != expectedCPUs {
		t.Errorf("Expected %d CPUs, found %d", expectedCPUs, cpus)
	}
}

// TestGetFSInfo tests the code that computes the diskspace of a given drive.
//
// We call GetFSInfo to determine the size of the underlying drive of the
// current working directory and the amount of MBs free on that drive.  If
// we fail to retrieve the current working directory, the test is skipped.
//
// Values > 0 should be returned for total and available.
func TestGetFSInfo(t *testing.T) {
	path, err := os.Getwd()
	if err != nil {
		t.Skip()
	}
	total, available := GetFSInfo(path)
	if total == -1 || available == -1 {
		t.Errorf("GetFSInfo failed total %d available %d", total, available)
	}
}

// TestGetLoadAvg tests the code that parse the /proc/loadavg file.
//
// The test passes a dummy loadavg file and checks the returned load matches
// the expected value.
//
// The returned load should be 1.
func TestGetLoadAvg(t *testing.T) {
	buf := bytes.NewBufferString(loadAvgContents)
	load := getLoadAvg(buf)
	const expectedLoad = 1
	if load != expectedLoad {
		t.Errorf("Expected Load %d , found %d", expectedLoad, load)
	}
}
