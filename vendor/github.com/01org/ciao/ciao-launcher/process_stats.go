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

package main

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/golang/glog"
)

func computeProcessMemUsage(pid int) int {
	mapsPath := path.Join("/proc", fmt.Sprintf("%d", pid), "smaps")
	return parseProcSmaps(mapsPath)
}

func parseProcSmaps(smapsPath string) int {
	smaps, err := os.Open(smapsPath)
	if err != nil {
		if glog.V(1) {
			glog.Warning("Unable to open %s: %v", smapsPath, err)
		}
		return -1
	}
	var mem64 int64
	scanner := bufio.NewScanner(smaps)
	for scanner.Scan() {
		matches := pssRegexp.FindStringSubmatch(scanner.Text())
		if matches == nil || len(matches) < 2 {
			continue
		}

		sizeInKb, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			continue
		}
		mem64 += sizeInKb
	}
	mem := int(mem64 / 1024)
	_ = smaps.Close()

	return mem
}

func computeProcessCPUTime(pid int) int64 {
	statPath := path.Join("/proc", fmt.Sprintf("%d", pid), "stat")
	return parseProcStat(statPath)
}

func parseProcStat(statPath string) int64 {
	stat, err := os.Open(statPath)
	if err != nil {
		if glog.V(1) {
			glog.Warning("Unable to open %s: %v", statPath, err)
		}
		return -1
	}
	defer func() { _ = stat.Close() }()

	var userTime int64 = -1
	var sysTime int64 = -1
	scanner := bufio.NewScanner(stat)
	scanner.Split(bufio.ScanWords)
	i := 0
	for ; i < 13 && scanner.Scan(); i++ {

	}

	if scanner.Scan() {
		userTime, _ = strconv.ParseInt(scanner.Text(), 10, 64)
		if scanner.Scan() {
			sysTime, _ = strconv.ParseInt(scanner.Text(), 10, 64)
		}
	}

	if userTime == -1 || sysTime == -1 {
		if glog.V(1) {
			glog.Warningf("Invalid user or systime %d %d",
				userTime, sysTime)
		}
		return -1
	}

	cpuTime := (1000 * 1000 * 1000 * (userTime + sysTime)) /
		clockTicksPerSecond

		//	if glog.V(1) {
		//		glog.Infof("PID %d: cpuTime %d userTime %d sysTime %d",
		//			q.pid, cpuTime, userTime, sysTime)
		//	}

	return cpuTime
}
