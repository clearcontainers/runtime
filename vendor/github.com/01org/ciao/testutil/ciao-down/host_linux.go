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
// +build linux

package main

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
)

// TODO: Copied from launcher

func getOnlineCPUs() int {
	cpuStatsRegexp := regexp.MustCompile(`^cpu[0-9]+.*$`)
	file, err := os.Open("/proc/stat")
	if err != nil {
		return -1
	}
	defer func() {
		_ = file.Close()
	}()

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

func getTotalMemory() int {
	memTotalRegexp := regexp.MustCompile(`MemTotal:\s+(\d+)`)
	total := -1
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return total
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return total
	}

	matches := memTotalRegexp.FindStringSubmatch(scanner.Text())
	if matches == nil {
		return total
	}

	parsedNum, err := strconv.Atoi(matches[1])
	if err != nil {
		return total
	}

	if parsedNum >= 0 {
		total = parsedNum / (1024 * 1024)
	}

	if total == 0 {
		return -1
	}

	return total
}

func getMemAndCpus() (mem int, cpus int) {
	cpus = getOnlineCPUs() / 2
	if cpus < 0 {
		cpus = 1
	}

	mem = getTotalMemory() / 2
	if mem < 0 {
		mem = 1
	}

	return mem, cpus
}
