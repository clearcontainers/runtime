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

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	spec "github.com/opencontainers/specs/specs-go"
)

const image = "busybox"

const tmpDir = "/tmp"

const configTemplate = "src/github.com/clearcontainers/tests/data/config.json"

// Bundle represents the root directory where config.json and rootfs are
type Bundle struct {
	// Config represents the config.json
	Config *spec.Spec

	// Path to the bundle
	Path string
}

// NewBundle creates a new bundle
func NewBundle(workload []string) (*Bundle, error) {
	path, err := ioutil.TempDir(tmpDir, "bundle")
	if err != nil {
		return nil, err
	}

	if err := createRootfs(path); err != nil {
		return nil, err
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return nil, fmt.Errorf("GOPATH is not set")
	}

	configTemplatePath := filepath.Join(gopath, configTemplate)
	content, err := ioutil.ReadFile(configTemplatePath)
	if err != nil {
		return nil, err
	}

	var config spec.Spec
	err = json.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	config.Process.Args = workload

	bundle := &Bundle{
		Path:   path,
		Config: &config,
	}

	err = bundle.Save()
	if err != nil {
		return nil, err
	}

	return bundle, nil
}

// createRootfs creates a rootfs in the specific bundlePath
func createRootfs(bundlePath string) error {
	if bundlePath == "" {
		return fmt.Errorf("bundle path should not be empty")
	}

	rootfsDir := filepath.Join(bundlePath, "rootfs")
	if err := os.MkdirAll(rootfsDir, 0755); err != nil {
		return err
	}

	// create container
	var container bytes.Buffer
	createCmd := exec.Command("docker", "create", image)
	createCmd.Stdout = &container
	if err := createCmd.Run(); err != nil {
		return err
	}
	containerName := strings.TrimRight(container.String(), "\n")

	// export container
	tarFile, err := ioutil.TempFile(tmpDir, "tar")
	if err != nil {
		return err
	}
	defer tarFile.Close()

	exportCmd := exec.Command("docker", "export", "-o", tarFile.Name(), containerName)
	if err := exportCmd.Run(); err != nil {
		return err
	}
	defer os.Remove(tarFile.Name())

	// extract container
	tarCmd := exec.Command("tar", "-C", rootfsDir, "-pxf", tarFile.Name())
	if err := tarCmd.Run(); err != nil {
		return err
	}

	// remove container
	rmCmd := exec.Command("docker", "rm", "-f", containerName)
	if err := rmCmd.Run(); err != nil {
		return err
	}

	return nil
}

// Save to disk the Config
func (b *Bundle) Save() error {
	content, err := json.Marshal(b.Config)
	if err != nil {
		return err
	}

	configFile := filepath.Join(b.Path, "config.json")
	err = ioutil.WriteFile(configFile, content, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Remove the bundle files and directories
func (b *Bundle) Remove() error {
	return os.RemoveAll(b.Path)
}
