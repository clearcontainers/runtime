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
	"crypto/rand"
	"encoding/base64"
)

// Token represents the communication between the process inside the VM and the
// host.
// In the case of clear containers, the shim process will use that token to
// identify itself against the proxy.
type Token string

const nilToken = Token("")

// generateRandomBytes returns securely generated random bytes.
//
// It will return an error if the system's secure random number generator
// fails to function correctly, in which case the caller should not continue.
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateToken returns a URL-safe, base64 encoded securely generated random
// string.
//
// It will return an error if the system's secure random number generator
// fails to function correctly, in which case the caller should not continue.
func GenerateToken(s int) (Token, error) {
	b, err := generateRandomBytes(s)
	return Token(base64.URLEncoding.EncodeToString(b)), err
}
