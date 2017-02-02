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
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"time"

	"context"

	"github.com/docker/engine-api/types"
	"github.com/golang/glog"
)

func dockerKillInstance(instanceDir string) {
	idPath := path.Join(instanceDir, "docker-id")
	data, err := ioutil.ReadFile(idPath)
	if err != nil {
		glog.Errorf("Unable to read docker container ID %v", err)
		return
	}

	cli, err := getDockerClient()
	if err != nil {
		return
	}

	dockerID := string(data)
	err = cli.ContainerRemove(context.Background(),
		types.ContainerRemoveOptions{
			ContainerID: dockerID,
			Force:       true})
	if err != nil {
		glog.Warningf("Unable to delete docker instance %s err %v", dockerID, err)
	}
}

func qemuKillInstance(instanceDir string) {
	var conn net.Conn

	qmpSocket := path.Join(instanceDir, "socket")
	conn, err := net.DialTimeout("unix", qmpSocket, time.Second*30)
	if err != nil {
		return
	}

	defer func() { _ = conn.Close() }()

	_, err = fmt.Fprintln(conn, "{ \"execute\": \"qmp_capabilities\" }")
	if err != nil {
		glog.Errorf("Unable to send qmp_capabilities to instance %s: %v", instanceDir, err)
		return
	}

	glog.Infof("Powering Down %s", instanceDir)

	_, err = fmt.Fprintln(conn, "{ \"execute\": \"quit\" }")
	if err != nil {
		glog.Errorf("Unable to send power down command to %s: %v\n", instanceDir, err)
	}

	// Keep reading until the socket fails.  If we close the socket straight away, qemu does not
	// honour our quit command.

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
	}

	return
}

func purgeLauncherState() {

	glog.Info("======= HARD RESET ======")

	glog.Info("Shutting down running instances")

	toRemove := make([]string, 0, 1024)
	dockerNetworking := false

	glog.Info("Init networking")

	if err := initNetworkPhase1(); err != nil {
		glog.Warningf("Failed to init network: %v\n", err)
	} else {
		defer shutdownNetwork()
		if err := initDockerNetworking(context.Background()); err != nil {
			glog.Info("Unable to initialise docker networking")
		} else {
			dockerNetworking = true
		}
	}

	_ = filepath.Walk(instancesDir, func(path string, info os.FileInfo, err error) error {
		if path == instancesDir {
			return nil
		}

		if !info.IsDir() {
			return nil
		}

		cfg, err := loadVMConfig(path)
		if err != nil {
			glog.Warningf("Unable to load config for %s: %v", path, err)
		} else {
			if cfg.Container {
				dockerKillInstance(path)
			} else {
				qemuKillInstance(path)
			}
		}
		toRemove = append(toRemove, path)
		return nil
	})

	for _, p := range toRemove {
		err := os.RemoveAll(p)
		if err != nil {
			glog.Warningf("Unable to remove instance dir for %s: %v", p, err)
		}
	}

	if dockerNetworking {
		glog.Info("Reset docker networking")

		resetDockerNetworking()
	}

	glog.Info("Reset networking")

	err := cnNet.ResetNetwork()
	if err != nil {
		glog.Warningf("Unable to reset network: %v", err)
	}
}
