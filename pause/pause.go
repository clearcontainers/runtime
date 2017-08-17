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
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func sigHandler(sig os.Signal) {
	var ret int

	switch sig {
	case syscall.SIGCHLD:
		fallthrough
	case syscall.SIGURG:
		fallthrough
	case syscall.SIGWINCH:
		// Ignore these signals
		// (for parity with standard C programs).
		return

	case syscall.SIGINT:
		fallthrough
	case syscall.SIGTERM:
		// The signals we expect result in a successful exit.
		ret = 0

	default:
		// Something bad happened.
		ret = 42
	}

	fmt.Fprintf(os.Stderr, "shutting down, got signal %v\n", sig)
	os.Exit(ret)
}

func main() {
	ch := make(chan os.Signal, 1)

	// Pass all signals to the handler
	signal.Notify(ch)

	// Keep on handling signals until either an expected one arrives
	// (successful exit), or a signal that is not being ignored
	// arrives (non-zero exit).
	for {
		sigHandler(<-ch)
	}
}
