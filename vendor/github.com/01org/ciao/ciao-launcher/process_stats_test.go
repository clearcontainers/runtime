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
	"io/ioutil"
	"os"
	"testing"
)

var smapsTest = []struct {
	data string
	res  int
}{
	{
		`
7fffeb521000-7fffeb523000 r--p 00000000 00:00 0                          [vvar]
Size:                  8 kB
Rss:                   0 kB
Pss:                   8 kB
`,
		0,
	},
	{`
7fffeb521000-7fffeb523000 r--p 00000000 00:00 0                          [vvar]
Size:              10504 kB
Rss:                   0 kB
Pss:               10504 kB
`,
		10,
	},
	{`
Pss:               10504 kB
Pss:               10504 kB
Pss:               10504 kB
Rss:                   0 kB
Pss:                   0 kB
Pss:               10504 kB
`,
		41,
	},
	{`
Pss:               10504 kB
Pss:               asasas kB
`,
		10,
	},
	{
		"",
		00,
	},
	{`
Rss:               10504 kB
`,
		0,
	},
	{`
Psst:               10504 kB
`,
		0,
	},
}

var statsTest = []struct {
	data string
	res  int64
}{
	{
		"9017 (emacs) S 1731 9017 1731 34816 1731 4194304 10138 4271 165 0 38 16 3 0 20 0 4 0 2154773 656859136 18106 18446744073709551615 4194304 6539236 140736937569392 140736937564512 140338947582652 0 0 67112960 1535209215 0 0 0 17 1 0 0 12 0 0 8637344 21454848 44126208 140736937575538 140736937575544 140736937575544 140736937578473 0",
		(38 + 16) * 1000 * 1000 * 1000,
	},
	{
		"9017 (emacs) S 1731 9017 1731 34816 1731 4194304 10138 4271 165 0 38",
		-1,
	},
	{
		"9017 (emacs) S 1731 9017 1731 34816 1731 4194304 10138 4271 165 0",
		-1,
	},
	{
		"9017 (emacs) S 1731 9017 1731 34816 1731 4194304 10138 4271 165 0 38 16",
		(38 + 16) * 1000 * 1000 * 1000,
	},
	{
		"",
		-1,
	},
}

// Verify the smaps parser
//
// This test passes a number of different test files to the parseProcSmaps
// function.  Some of the files are valid and some are invalid.  Finally,
// it calls parseProcSmaps on an invalid path.
//
// parseProcSmaps should correctly parse the valid smaps files and return
// an error when asked to parse the invalid files.  The attempt to call
// parseProcSmaps on the invalid path should fail.
func TestParseProcSmaps(t *testing.T) {
	for _, tst := range smapsTest {
		f, err := ioutil.TempFile("", "process_stats_test")
		if err != nil {
			t.Errorf("Unable to create tempory file : %v", err)
			continue
		}
		_, err = f.WriteString(tst.data)
		err2 := f.Close()
		func() {
			defer func() {
				_ = os.RemoveAll(f.Name())
			}()
			if err != nil {
				t.Errorf("Unable to write to tempory file : %v", err)
				return
			}
			if err2 != nil {
				t.Errorf("Unable to close tempory file : %v", err)
				return
			}

			mem := parseProcSmaps(f.Name())
			if mem != tst.res {
				t.Errorf("Incorrect value from parseProcSmaps.  Expected %d found %d",
					mem, tst.res)
			}
		}()
	}

	if parseProcSmaps("") != -1 {
		t.Errorf("Expected parseProcSmaps to fail when passed invalid path")
	}
}

// Verify the proc parser
//
// This test passes a number of different test files to the parseProcStat
// function.  Some of the files are valid and some are invalid.  Finally,
// it calls parseProcStat on an invalid path.
//
// parseProcStat should correctly parse the valid stat files and return
// an error when asked to parse the invalid files.  The attempt to call
// parseProcStat on the invalid path should fail.
func TestParseProcStat(t *testing.T) {
	for _, tst := range statsTest {
		f, err := ioutil.TempFile("", "process_stats_test")
		if err != nil {
			t.Errorf("Unable to create tempory file : %v", err)
			continue
		}
		_, err = f.WriteString(tst.data)
		err2 := f.Close()
		func() {
			defer func() {
				_ = os.RemoveAll(f.Name())
			}()
			if err != nil {
				t.Errorf("Unable to write to tempory file : %v", err)
				return
			}
			if err2 != nil {
				t.Errorf("Unable to close tempory file : %v", err)
				return
			}

			mem := parseProcStat(f.Name())
			if tst.res != -1 {
				tst.res /= clockTicksPerSecond
			}
			if mem != tst.res {
				t.Errorf("Incorrect value from parseProcStat.  Expected %d found %d",
					mem, tst.res)
			}
		}()
	}

	if parseProcStat("") != -1 {
		t.Errorf("Expected parseProcStat to fail when passed invalid path")
	}
}
