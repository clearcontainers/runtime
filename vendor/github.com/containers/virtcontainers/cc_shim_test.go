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

package virtcontainers

import (
	"os"
	"path/filepath"
	"testing"
)

var testShimPath = "/tmp/bin/cc-shim-mock"
var testProxyURL = "foo:///foo/clear-containers/proxy.sock"
var testWrongConsolePath = "/foo/wrong-console"
var testConsolePath = "tty-console"

func testCCShimStart(t *testing.T, pod Pod, params ShimParams, expectFail bool) {
	s := &ccShim{}

	pid, err := s.start(pod, params)
	if expectFail {
		if err == nil || pid != -1 {
			t.Fatalf("This test should fail (pod %+v, params %+v, expectFail %t)",
				pod, params, expectFail)
		}
	} else {
		if err != nil {
			t.Fatalf("This test should pass (pod %+v, params %+v, expectFail %t): %s",
				pod, params, expectFail, err)
		}

		if pid == -1 {
			t.Fatalf("This test should pass (pod %+v, params %+v, expectFail %t)",
				pod, params, expectFail)
		}
	}
}

func TestCCShimStartNilPodConfigFailure(t *testing.T) {
	testCCShimStart(t, Pod{}, ShimParams{}, true)
}

func TestCCShimStartNilShimConfigFailure(t *testing.T) {
	pod := Pod{
		config: &PodConfig{},
	}

	testCCShimStart(t, pod, ShimParams{}, true)
}

func TestCCShimStartShimPathEmptyFailure(t *testing.T) {
	pod := Pod{
		config: &PodConfig{
			ShimType:   CCShimType,
			ShimConfig: CCShimConfig{},
		},
	}

	testCCShimStart(t, pod, ShimParams{}, true)
}

func TestCCShimStartParamsTokenEmptyFailure(t *testing.T) {
	pod := Pod{
		config: &PodConfig{
			ShimType: CCShimType,
			ShimConfig: CCShimConfig{
				Path: testShimPath,
			},
		},
	}

	testCCShimStart(t, pod, ShimParams{}, true)
}

func TestCCShimStartParamsURLEmptyFailure(t *testing.T) {
	pod := Pod{
		config: &PodConfig{
			ShimType: CCShimType,
			ShimConfig: CCShimConfig{
				Path: testShimPath,
			},
		},
	}

	params := ShimParams{
		Token: "testToken",
	}

	testCCShimStart(t, pod, params, true)
}

func TestCCShimStartSuccessful(t *testing.T) {
	pod := Pod{
		config: &PodConfig{
			ShimType: CCShimType,
			ShimConfig: CCShimConfig{
				Path: testShimPath,
			},
		},
	}

	params := ShimParams{
		Token: "testToken",
		URL:   testProxyURL,
	}

	testCCShimStart(t, pod, params, false)
}

func TestCCShimStartWithConsoleNonExistingFailure(t *testing.T) {
	pod := Pod{
		config: &PodConfig{
			ShimType: CCShimType,
			ShimConfig: CCShimConfig{
				Path: testShimPath,
			},
		},
	}

	params := ShimParams{
		Token:   "testToken",
		URL:     testProxyURL,
		Console: testWrongConsolePath,
	}

	testCCShimStart(t, pod, params, true)
}

func TestCCShimStartWithConsoleSuccessful(t *testing.T) {
	cleanUp()

	consolePath := filepath.Join(testDir, testConsolePath)
	f, err := os.Create(consolePath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	pod := Pod{
		config: &PodConfig{
			ShimType: CCShimType,
			ShimConfig: CCShimConfig{
				Path: testShimPath,
			},
		},
	}

	params := ShimParams{
		Token:   "testToken",
		URL:     testProxyURL,
		Console: consolePath,
	}

	testCCShimStart(t, pod, params, false)
}
