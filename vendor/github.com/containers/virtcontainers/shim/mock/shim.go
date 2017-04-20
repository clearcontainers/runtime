//
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
//

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
)

const logFileName = "mock_shim.log"
const numArgsExpected = 2

func main() {
	logDirPath, err := ioutil.TempDir("", "cc-shim-")
	if err != nil {
		fmt.Printf("ERROR: Could not generate temporary log directory path: %s\n", err)
		os.Exit(1)
	}

	logFilePath := filepath.Join(logDirPath, logFileName)

	f, err := os.Create(logFilePath)
	if err != nil {
		fmt.Printf("ERROR: Could not create temporary log file %q: %s\n", logFilePath, err)
		os.Exit(1)
	}
	defer f.Close()

	tokenFlag := flag.String("t", "", "Proxy token")
	urlFlag := flag.String("u", "", "Proxy URL")

	flag.Parse()

	fmt.Fprintf(f, "INFO: Token = %s\n", *tokenFlag)
	fmt.Fprintf(f, "INFO: URL = %s\n", *urlFlag)

	if *tokenFlag == "" {
		fmt.Fprintf(f, "ERROR: Token should not be empty\n")
		os.Exit(1)
	}

	if *urlFlag == "" {
		fmt.Fprintf(f, "ERROR: URL should not be empty\n")
		os.Exit(1)
	}

	if _, err := url.Parse(*urlFlag); err != nil {
		fmt.Fprintf(f, "ERROR: Could not parse the URL %q: %s\n", *urlFlag, err)
		os.Exit(1)
	}

	fmt.Fprintf(f, "INFO: Shim exited properly\n")
}
