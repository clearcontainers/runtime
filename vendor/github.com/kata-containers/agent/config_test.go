//
// Copyright (c) 2018 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	testLogLevel := logrus.DebugLevel

	expectedConfig := agentConfig{
		logLevel: testLogLevel,
	}

	config := newConfig(testLogLevel)

	assert.True(t, reflect.DeepEqual(config, expectedConfig),
		"Config structures should be identical: got %+v, expecting %+v",
		config, expectedConfig)
}

func TestParseCmdlineOptionEmptyOption(t *testing.T) {
	a := &agentConfig{}

	err := a.parseCmdlineOption("")
	assert.Nil(t, err, "%v", err)
}

func TestParseCmdlineOptionWrongOptionValue(t *testing.T) {
	a := &agentConfig{}

	wrongOption := logLevelFlag + "=debgu"

	err := a.parseCmdlineOption(wrongOption)
	assert.Error(t, err, "Parsing should fail because wrong option %q", wrongOption)
}

func TestParseCmdlineOptionWrongOptionParam(t *testing.T) {
	a := &agentConfig{}

	wrongOption := "agent.lgo=debug"

	err := a.parseCmdlineOption(wrongOption)
	assert.Error(t, err, "Parsing should fail because wrong option %q", wrongOption)
}

func TestParseCmdlineOptionCorrectOptions(t *testing.T) {
	a := &agentConfig{}

	logFlagList := []string{"debug", "info", "warn", "error", "fatal", "panic"}

	for _, logFlag := range logFlagList {
		option := logLevelFlag + "=" + logFlag

		err := a.parseCmdlineOption(option)
		assert.Nil(t, err, "%v", err)
	}
}

func TestParseCmdlineOptionIncorrectOptions(t *testing.T) {
	a := &agentConfig{}

	logFlagList := []string{"debg", "ifo", "wan", "eror", "ftal", "pnic"}

	for _, logFlag := range logFlagList {
		option := logLevelFlag + "=" + logFlag

		err := a.parseCmdlineOption(option)
		assert.Error(t, err, "Should fail because of incorrect option %q", logFlag)
	}
}

func TestGetConfigEmptyFileName(t *testing.T) {
	a := &agentConfig{}

	err := a.getConfig("")
	assert.Error(t, err, "Should fail because command line path is empty")
}

func TestGetConfigFilePathNotExist(t *testing.T) {
	a := &agentConfig{}

	tmpFile, err := ioutil.TempFile("", "test")
	assert.Nil(t, err, "%v", err)

	fileName := tmpFile.Name()
	tmpFile.Close()
	err = os.Remove(fileName)
	assert.Nil(t, err, "%v", err)

	err = a.getConfig(fileName)
	assert.Error(t, err, "Should fail because command line path does not exist")
}

func TestGetConfig(t *testing.T) {
	a := &agentConfig{}

	tmpFile, err := ioutil.TempFile("", "test")
	assert.Nil(t, err, "%v", err)
	fileName := tmpFile.Name()

	tmpFile.Write([]byte(logLevelFlag + "=info"))
	tmpFile.Close()

	defer os.Remove(fileName)

	err = a.getConfig(fileName)
	assert.Nil(t, err, "%v", err)

	assert.True(t, a.logLevel == logrus.InfoLevel,
		"Log levels should be identical: got %+v, expecting %+v",
		a.logLevel, logrus.InfoLevel)
}
