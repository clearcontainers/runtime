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
	"fmt"
	"os"
	"syscall"
	"testing"
	"unsafe"

	. "github.com/containers/virtcontainers/pkg/mock"
)

var testShimPath = "/usr/bin/virtcontainers/bin/test/shim"
var testProxyURL = "foo:///foo/clear-containers/proxy.sock"
var testWrongConsolePath = "/foo/wrong-console"
var testConsolePath = "tty-console"

func getMockCCShimBinPath() string {
	if DefaultMockCCShimBinPath == "" {
		return testShimPath
	}

	return DefaultMockCCShimBinPath
}

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
				Path: getMockCCShimBinPath(),
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
				Path: getMockCCShimBinPath(),
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
				Path: getMockCCShimBinPath(),
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
				Path: getMockCCShimBinPath(),
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

func ioctl(fd uintptr, flag, data uintptr) error {
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, flag, data); err != 0 {
		return err
	}

	return nil
}

// unlockpt unlocks the slave pseudoterminal device corresponding to the master pseudoterminal referred to by f.
func unlockpt(f *os.File) error {
	var u int32

	return ioctl(f.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&u)))
}

// ptsname retrieves the name of the first available pts for the given master.
func ptsname(f *os.File) (string, error) {
	var n int32

	if err := ioctl(f.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n))); err != nil {
		return "", err
	}

	return fmt.Sprintf("/dev/pts/%d", n), nil
}

func newConsole() (*os.File, string, error) {
	master, err := os.OpenFile("/dev/ptmx", syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, "", err
	}

	console, err := ptsname(master)
	if err != nil {
		return nil, "", err
	}

	if err := unlockpt(master); err != nil {
		return nil, "", err
	}

	if err := os.Chmod(console, 0600); err != nil {
		return nil, "", err
	}

	return master, console, nil
}

func TestCCShimStartWithConsoleSuccessful(t *testing.T) {
	cleanUp()

	master, console, err := newConsole()
	t.Logf("Console created for tests:%s\n", console)

	if err != nil {
		t.Fatal(err)
	}

	pod := Pod{
		config: &PodConfig{
			ShimType: CCShimType,
			ShimConfig: CCShimConfig{
				Path: getMockCCShimBinPath(),
			},
		},
	}

	params := ShimParams{
		Token:   "testToken",
		URL:     testProxyURL,
		Console: console,
	}

	testCCShimStart(t, pod, params, false)
	master.Close()
}
