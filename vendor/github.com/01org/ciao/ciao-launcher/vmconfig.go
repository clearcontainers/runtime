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
	"encoding/gob"
	"os"
	"path"

	"github.com/golang/glog"
)

type volumeConfig struct {
	UUID     string
	Bootable bool
}

type vmConfig struct {
	Cpus        int
	Mem         int
	Disk        int
	Instance    string
	Image       string
	Legacy      bool
	Container   bool
	NetworkNode bool
	VnicMAC     string
	VnicIP      string
	ConcIP      string
	SubnetIP    string
	TenantUUID  string
	ConcUUID    string
	VnicUUID    string
	SSHPort     int
	Volumes     []volumeConfig
}

func loadVMConfig(instanceDir string) (*vmConfig, error) {
	cfgFilePath := path.Join(instanceDir, instanceState)
	cfgFile, err := os.Open(cfgFilePath)
	if err != nil {
		glog.Errorf("Unable to open instance file %s", cfgFilePath)
		return nil, err
	}

	dec := gob.NewDecoder(cfgFile)
	cfg := &vmConfig{}
	err = dec.Decode(cfg)
	_ = cfgFile.Close()

	if err != nil {
		glog.Error("Unable to retrieve state info")
		return nil, err
	}

	return cfg, nil
}

func (cfg *vmConfig) save(instanceDir string) error {
	cfgFilePath := path.Join(instanceDir, instanceState)
	cfgFile, err := os.OpenFile(cfgFilePath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		glog.Errorf("Unable to create state file %v", err)
		return err
	}

	enc := gob.NewEncoder(cfgFile)
	if err = enc.Encode(cfg); err != nil {
		glog.Errorf("Failed to store state information %v", err)
		_ = cfgFile.Close()
		return err
	}

	return cfgFile.Close()
}

func (cfg *vmConfig) findVolume(UUID string) *volumeConfig {
	for i := range cfg.Volumes {
		if cfg.Volumes[i].UUID == UUID {
			return &cfg.Volumes[i]
		}
	}
	return nil
}
func (cfg *vmConfig) haveBootableVolume() bool {
	for _, vol := range cfg.Volumes {
		if vol.Bootable {
			return true
		}
	}
	return false
}

func (cfg *vmConfig) removeVolume(UUID string) {
	for i := range cfg.Volumes {
		if cfg.Volumes[i].UUID == UUID {
			vols := cfg.Volumes
			cfg.Volumes = vols[:i]
			if i+1 < len(vols) {
				cfg.Volumes = append(cfg.Volumes, vols[i+1:]...)
			}
			break
		}
	}
}
