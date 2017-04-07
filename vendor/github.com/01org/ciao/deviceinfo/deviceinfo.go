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
	"bufio"
	"io"
	"os"
	"regexp"
	"strconv"
	"syscall"
)

var memTotalRegexp *regexp.Regexp
var memFreeRegexp *regexp.Regexp
var memActiveFileRegexp *regexp.Regexp
var memInactiveFileRegexp *regexp.Regexp
var cpuStatsRegexp *regexp.Regexp

func init() {
	memTotalRegexp = regexp.MustCompile(`MemTotal:\s+(\d+)`)
	memFreeRegexp = regexp.MustCompile(`MemFree:\s+(\d+)`)
	memActiveFileRegexp = regexp.MustCompile(`Active\(file\):\s+(\d+)`)
	memInactiveFileRegexp = regexp.MustCompile(`Inactive\(file\):\s+(\d+)`)
	cpuStatsRegexp = regexp.MustCompile(`^cpu[0-9]+.*$`)
}

func grabInt(re *regexp.Regexp, line string, val *int) bool {
	matches := re.FindStringSubmatch(line)
	if matches != nil {
		parsedNum, err := strconv.Atoi(matches[1])
		if err == nil {
			*val = parsedNum
			return true
		}
	}
	return false
}

func getMemoryInfo(file io.Reader) (total, available int) {
	total = -1
	available = -1
	free := -1
	active := -1
	inactive := -1

	scanner := bufio.NewScanner(file)
	for scanner.Scan() && (total == -1 || free == -1 || active == -1 ||
		inactive == -1) {
		line := scanner.Text()
		for _, i := range []struct {
			v *int
			r *regexp.Regexp
		}{
			{&free, memFreeRegexp},
			{&total, memTotalRegexp},
			{&active, memActiveFileRegexp},
			{&inactive, memInactiveFileRegexp},
		} {
			if *i.v == -1 {
				if grabInt(i.r, line, i.v) {
					break
				}
			}
		}
	}

	if free != -1 && active != -1 && inactive != -1 {
		available = (free + active + inactive) / 1024
	}

	if total != -1 {
		total = total / 1024
	}

	return
}

// GetMemoryInfo returns the both the total amount of memory installed in the
// device and the amount of memory currently available.  A return value of -1
// indicates that an error occurred computing the return value.
func GetMemoryInfo() (total, available int) {

	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}

	total, available = getMemoryInfo(file)

	_ = file.Close()

	return
}

func getOnlineCPUs(file io.Reader) int {
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return -1
	}
	cpusOnline := 0
	for scanner.Scan() && cpuStatsRegexp.MatchString(scanner.Text()) {
		cpusOnline++
	}

	if cpusOnline == 0 {
		return -1
	}

	return cpusOnline
}

// GetOnlineCPUs returns the number of CPUS in the device.  A return value of
// -1 indicates that an error has occurred.
func GetOnlineCPUs() int {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return -1
	}
	cpusOnline := getOnlineCPUs(file)
	_ = file.Close()
	return cpusOnline
}

// GetFSInfo returns the total size and the amount of available space of the
// drive on which the specified path is located.  The sizes returned are in
// MiBs.  A return value of -1 indicates an error.
func GetFSInfo(path string) (total, available int) {
	total = -1
	available = -1
	var buf syscall.Statfs_t

	if syscall.Statfs(path, &buf) != nil {
		return
	}

	if buf.Bsize <= 0 {
		return
	}

	total = int((uint64(buf.Bsize) * buf.Blocks) / (1000 * 1000))
	available = int((uint64(buf.Bsize) * buf.Bavail) / (1000 * 1000))

	return
}

func getLoadAvg(file io.Reader) int {
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	if !scanner.Scan() {
		return -1
	}

	loadFloat, err := strconv.ParseFloat(scanner.Text(), 64)
	if err != nil {
		return -1
	}

	return int(loadFloat)
}

// GetLoadAvg returns the average load of the device.  -1 is returned if
// an error occurred.
func GetLoadAvg() int {
	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return -1
	}

	load := getLoadAvg(file)

	_ = file.Close()

	return load
}
