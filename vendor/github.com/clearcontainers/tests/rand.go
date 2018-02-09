/// Copyright (c) 2017 Intel Corporation
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

package tests

import (
	"math/rand"
	"time"
)

const letters = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const lettersMask = 63

var randSrc = rand.NewSource(time.Now().UnixNano())

// RandID returns a random string
func RandID(n int) string {
	b := make([]byte, n)
	for i := 0; i < n; {
		if j := int(randSrc.Int63() & lettersMask); j < len(letters) {
			b[i] = letters[j]
			i++
		}
	}

	return string(b)
}
