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
	"fmt"
	"sync"
	"testing"
)

type inspector struct {
	err error
}

func (i *inspector) imageInfo(imagePath string) (int, error) {
	if i.err != nil {
		return 0, i.err
	}
	return 10, nil
}

// Test GetMinImageSize function
//
// Call getMinImageSize 10 times in parallel with the same path.  Call again
// with the same old path but setting an error.  Call getMinSize 1 more time
// with the error set but with a new path.
//
// The first 10 calls should succeed.  The 11th call with the error set should
// also succeed as the size for the path is already cached.  The final function
// call should fail as we have arranged for imageInfo to return an error and
// we are calling getMinImageSize on a new path.
func TestGetMinImageSize(t *testing.T) {
	in := &inspector{}
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			size, err := getMinImageSize(in, "")
			if err != nil || size != 10 {
				t.Errorf("Unexpected return value from getMinImageSize")
			}
			wg.Done()
		}()
	}
	wg.Wait()

	in.err = fmt.Errorf("Can't read image")
	size, err := getMinImageSize(in, "")
	if err != nil || size != 10 {
		t.Errorf("Unexpected return value from getMinImageSize")
	}

	_, err = getMinImageSize(in, "/new/path")
	if err == nil {
		t.Errorf("Error expected from getMinImageSize")
	}

}
