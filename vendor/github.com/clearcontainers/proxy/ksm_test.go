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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const ksmString = "ksmrules"
const anonPagesMemory = 16777216 // Typically 4096 pages
const run = "1"
const interval = "10"
const scan = "1000"

func ksmTestPrepare() error {
	newKSMRoot, err := ioutil.TempDir("", "cc-ksm-test")
	if err != nil {
		return err
	}

	defaultKSMRoot = newKSMRoot

	memInfoFile, err := ioutil.TempFile("", "cc-ksm-meminfo")
	if err != nil {
		return err
	}

	memInfo = memInfoFile.Name()

	_, err = memInfoFile.WriteString(fmt.Sprintf("AnonPages: %v kB", anonPagesMemory))
	if err != nil {
		return err
	}

	ksmTestRun, err := os.Create(filepath.Join(defaultKSMRoot, ksmRunFile))
	if err != nil {
		return err
	}

	ksmTestPagesToScan, err := os.Create(filepath.Join(defaultKSMRoot, ksmPagesToScan))
	if err != nil {
		return err
	}

	ksmTestSleepMillisec, err := os.Create(filepath.Join(defaultKSMRoot, ksmSleepMillisec))
	if err != nil {
		return err
	}

	defer ksmTestRun.Close()
	defer ksmTestPagesToScan.Close()
	defer ksmTestSleepMillisec.Close()

	return nil
}

func ksmTestCleanup() {
	os.RemoveAll(defaultKSMRoot)
	os.RemoveAll(memInfo)
}

func TestKSMSysfsAttributeOpen(t *testing.T) {
	pagesToScanSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmPagesToScan),
	}

	err := pagesToScanSysFs.open()
	defer pagesToScanSysFs.close()

	assert.Nil(t, err)
}

func TestKSMSysfsAttributeOpenNonExistent(t *testing.T) {
	pagesToScanSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, "foo"),
	}

	err := pagesToScanSysFs.open()
	defer pagesToScanSysFs.close()

	assert.NotNil(t, err)
}

func TestKSMSysfsAttributeReadWrite(t *testing.T) {
	pagesToScanSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmPagesToScan),
	}

	err := pagesToScanSysFs.open()
	defer pagesToScanSysFs.close()

	assert.Nil(t, err)

	err = pagesToScanSysFs.write(ksmString)
	assert.Nil(t, err)

	s, err := pagesToScanSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, s, ksmString, "Wrong sysfs read: %s", s)
}

func initKSM(root string, t *testing.T) *ksm {
	_, err := newKSM("")
	assert.NotNil(t, err)

	k, err := newKSM(root)
	assert.Nil(t, err)

	return k
}

func TestKSMAvailabilityDummy(t *testing.T) {
	_, err := newKSM("foo")
	assert.NotNil(t, err)
}

func TestKSMAvailability(t *testing.T) {
	k := initKSM(defaultKSMRoot, t)

	err := k.isAvailable()
	assert.Nil(t, err)
}

func TestKSMAnonPages(t *testing.T) {
	pageSize := (int64)(os.Getpagesize())
	expectedAnonPages := (anonPagesMemory * 1024) / pageSize

	anon, err := anonPages()
	assert.Nil(t, err)
	assert.Equal(t, expectedAnonPages, anon, "Anonymous pages mismatch")
}

func TestKSMPagesToScan(t *testing.T) {
	setting, valid := ksmSettings[ksmAggressive]
	assert.True(t, valid)

	anonPages, err := anonPages()
	assert.Nil(t, err)
	expectedPagesToScan := fmt.Sprintf("%v", anonPages/setting.pagesPerScanFactor)

	pagesToScan, err := setting.pagesToScan()
	assert.Nil(t, err)
	assert.Equal(t, pagesToScan, expectedPagesToScan, "")
}

func TestKSMPagesToScanInvalidSetting(t *testing.T) {
	setting := ksmSetting{
		pagesPerScanFactor: 0,
	}

	_, err := setting.pagesToScan()
	assert.NotNil(t, err)
}

func TestKSMInit(t *testing.T) {
	runSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmRunFile),
	}

	err := runSysFs.open()
	defer runSysFs.close()
	assert.Nil(t, err)

	err = runSysFs.write(run)
	assert.Nil(t, err)

	pagesToScanSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmPagesToScan),
	}

	err = pagesToScanSysFs.open()
	defer pagesToScanSysFs.close()
	assert.Nil(t, err)

	err = pagesToScanSysFs.write(scan)
	assert.Nil(t, err)

	sleepIntervalSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmSleepMillisec),
	}

	err = sleepIntervalSysFs.open()
	defer sleepIntervalSysFs.close()
	assert.Nil(t, err)

	err = sleepIntervalSysFs.write(interval)
	assert.Nil(t, err)

	k := initKSM(defaultKSMRoot, t)

	assert.Equal(t, k.initialPagesToScan, scan)
	assert.Equal(t, k.initialSleepInterval, interval)
	assert.Equal(t, k.initialKSMRun, run)
}

func TestKSMRestore(t *testing.T) {
	runSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmRunFile),
	}

	err := runSysFs.open()
	defer runSysFs.close()
	assert.Nil(t, err)

	err = runSysFs.write(run)
	assert.Nil(t, err)

	pagesToScanSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmPagesToScan),
	}

	err = pagesToScanSysFs.open()
	defer pagesToScanSysFs.close()
	assert.Nil(t, err)

	err = pagesToScanSysFs.write(scan)
	assert.Nil(t, err)

	sleepIntervalSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmSleepMillisec),
	}

	err = sleepIntervalSysFs.open()
	defer sleepIntervalSysFs.close()
	assert.Nil(t, err)

	err = sleepIntervalSysFs.write(interval)
	assert.Nil(t, err)

	k := initKSM(defaultKSMRoot, t)

	// Write dummy values and read them back
	var newInterval = "foo"
	var newRun = "bar"
	var newScan = "foobar"

	err = sleepIntervalSysFs.write(newInterval)
	assert.Nil(t, err)

	s, err := sleepIntervalSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, s, newInterval)

	err = runSysFs.write(newRun)
	assert.Nil(t, err)
	s, err = runSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, s, newRun)

	err = pagesToScanSysFs.write(newScan)
	assert.Nil(t, err)
	s, err = pagesToScanSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, s, newScan)

	// Now restore and verify that we read the initial values back
	k.restore()

	s, err = pagesToScanSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, s, scan)

	s, err = runSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, s, run)

	s, err = sleepIntervalSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, s, interval)
}

func TestKSMKick(t *testing.T) {
	k := initKSM(defaultKSMRoot, t)

	timer := time.NewTimer(time.Second)
	k.throttling = true
	go k.kick()

	select {
	case <-k.kickChannel:
		return

	case <-timer.C:
		t.Fatalf("KSM kick timeout")
	}
}

func TestKSMTune(t *testing.T) {
	var err error
	var s string

	sleepIntervalSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmSleepMillisec),
	}

	runSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmRunFile),
	}

	err = sleepIntervalSysFs.open()
	defer sleepIntervalSysFs.close()
	assert.Nil(t, err)

	err = runSysFs.open()
	defer runSysFs.close()
	assert.Nil(t, err)

	k := initKSM(defaultKSMRoot, t)

	for _, v := range ksmSettings {
		err = k.tune(v)
		assert.Nil(t, err)

		s, err = runSysFs.read()

		assert.Nil(t, err)
		assert.NotNil(t, s)
		if v.run {
			assert.Equal(t, s, "1", "Wrong run value")
		} else {
			assert.Equal(t, s, "0", "Wrong run value")
		}

		if !v.run {
			continue
		}

		s, err = sleepIntervalSysFs.read()

		assert.Nil(t, err)
		assert.NotNil(t, s)
		assert.Equal(t, s, fmt.Sprintf("%v", v.scanIntervalMS), "Wrong sleep interval")
	}
}

func TestKSMModeSet(t *testing.T) {
	var k ksmMode
	var err error

	err = k.Set(string(ksmOff))
	assert.Nil(t, err)
	assert.Equal(t, k, ksmOff)

	err = k.Set(string(ksmAuto))
	assert.Nil(t, err)
	assert.Equal(t, k, ksmAuto)

	err = k.Set(string(ksmInitial))
	assert.Nil(t, err)
	assert.Equal(t, k, ksmInitial)

	err = k.Set(string(ksmStandard))
	assert.NotNil(t, err)

	err = k.Set(fmt.Sprintf("%s,%s", ksmAuto, ksmOff))
	assert.Nil(t, err)
	assert.Equal(t, k, ksmAuto)
}

func TestKSMModeString(t *testing.T) {
	var k ksmMode
	var s string

	k = ksmOff
	s = string(k)
	assert.Equal(t, string(k), s)

	k = ksmInitial
	s = string(k)
	assert.Equal(t, string(k), s)

	k = ksmAuto
	s = string(k)
	assert.Equal(t, string(k), s)
}

func boolString(b bool) string {
	if b {
		return "1"
	}

	return "0"
}

func testThrottle(k *ksm, t *testing.T) {
	var err error
	var s string

	assert.NotNil(t, k)

	sleepIntervalSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmSleepMillisec),
	}

	runSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmRunFile),
	}

	err = sleepIntervalSysFs.open()
	defer sleepIntervalSysFs.close()
	assert.Nil(t, err)

	err = runSysFs.open()
	defer runSysFs.close()
	assert.Nil(t, err)

	// We should first be in aggressive mode
	time.Sleep(100 * time.Millisecond)
	k.Lock()
	assert.Equal(t, k.currentKnob, ksmAggressive)
	k.Unlock()

	// Let's check sysfs values are the aggressive ones
	s, err = runSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, boolString(ksmSettings[ksmAggressive].run), s, "Wrong sysfs read: %s", s)

	s, err = sleepIntervalSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, fmt.Sprintf("%d", ksmSettings[ksmAggressive].scanIntervalMS), s, "Wrong sysfs read: %s", s)

	nextKnob := ksmThrottleIntervals[ksmAggressive].nextKnob
	time.Sleep(ksmAggressiveInterval)

	// Let's check for the next knob
	time.Sleep(100 * time.Millisecond)
	k.Lock()
	assert.Equal(t, k.currentKnob, nextKnob)
	expectedRun := boolString(ksmSettings[k.currentKnob].run)
	expectedScan := ksmSettings[k.currentKnob].scanIntervalMS
	k.Unlock()

	// Let's check sysfs values are properly set
	s, err = runSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, expectedRun, s, "Wrong sysfs read: %s", s)

	s, err = sleepIntervalSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, fmt.Sprintf("%d", expectedScan), s, "Wrong sysfs read: %s", s)
}

func TestKSMThrottle(t *testing.T) {
	k := initKSM(defaultKSMRoot, t)

	// Let's make the throttling down faster, for quicker tests purpose.
	ksmAggressiveInterval = 500 * time.Millisecond

	k.throttle()
	k.kick()

	testThrottle(k, t)
}

func TestKSMStartInitialMode(t *testing.T) {
	var err error

	sleepIntervalSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmSleepMillisec),
	}

	runSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmRunFile),
	}

	err = sleepIntervalSysFs.open()
	defer sleepIntervalSysFs.close()
	assert.Nil(t, err)

	err = runSysFs.open()
	defer runSysFs.close()
	assert.Nil(t, err)

	initialRun, err := runSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, initialRun)

	initialSleep, err := sleepIntervalSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, initialSleep)

	k, err := startKSM(defaultKSMRoot, ksmInitial)
	assert.Nil(t, err)
	assert.NotNil(t, k)

	newRun, err := runSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, newRun)
	assert.Equal(t, newRun, initialRun, "Run sysfs attribute modified")

	newSleep, err := sleepIntervalSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, newSleep)
	assert.Equal(t, newSleep, initialSleep, "Sleep sysfs attribute modified")
}

func TestKSMStartAutoMode(t *testing.T) {
	// Let's make the throttling down faster, for quicker tests purpose.
	ksmAggressiveInterval = 500 * time.Millisecond

	k, err := startKSM(defaultKSMRoot, ksmAuto)
	assert.Nil(t, err)
	assert.NotNil(t, k)

	k.kick()

	testThrottle(k, t)
}

func TestKSMStartOffMode(t *testing.T) {
	var err error

	runSysFs := sysfsAttribute{
		path: filepath.Join(defaultKSMRoot, ksmRunFile),
	}

	err = runSysFs.open()
	defer runSysFs.close()
	assert.Nil(t, err)

	k, err := startKSM(defaultKSMRoot, ksmOff)
	assert.Nil(t, err)
	assert.NotNil(t, k)

	s, err := runSysFs.read()
	assert.Nil(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, s, ksmStop, "KSM not stopped")
}

func TestKSMStartInvalidMode(t *testing.T) {
	k, err := startKSM(defaultKSMRoot, "foo")
	assert.NotNil(t, k)
	assert.NotNil(t, err)
}
