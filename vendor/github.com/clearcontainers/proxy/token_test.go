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
	"testing"

	"github.com/stretchr/testify/assert"
)

// base64Length computes the length of the base64 encoding for n bytes of data
func base64Length(n int) int {
	return ((4 * n / 3) + 3) & ^3
}

func TestGenerateToken(t *testing.T) {
	// 32 bytes of data, 256 bytes. Each digit in base64 represents 6 bits
	// of input data. base64 add some stuffing so the number of bits to
	// encode is divisible by 6.
	token, err := GenerateToken(32)
	assert.Nil(t, err)
	assert.Equal(t, base64Length(32), len(token))
}
