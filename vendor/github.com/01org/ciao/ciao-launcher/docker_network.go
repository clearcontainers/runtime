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
	"sync"

	"context"

	"github.com/01org/ciao/networking/libsnnet"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/network"
	"github.com/golang/glog"
)

type dockerNetworkState struct {
	done chan struct{}
	err  error
}

var dockerNetworkLock sync.Mutex

func createDockerVnic(vnicCfg *libsnnet.VnicConfig) (*libsnnet.Vnic, *libsnnet.SsntpEventInfo, *libsnnet.ContainerInfo, error) {
	dockerNetworkLock.Lock()
	defer dockerNetworkLock.Unlock()
	vnic, event, info, err := cnNet.CreateVnic(vnicCfg)
	if err != nil || info.CNContainerEvent != libsnnet.ContainerNetworkAdd {
		return vnic, event, info, err
	}

	// We are going to ignore the error here.  Docker network
	// might fail as it could already have been created.  We
	// would not know this if it was created in a previous invocation
	// of launcher.  Should the network really fail to be created
	// the container will not launch.

	_ = createDockerNetwork(context.Background(), info)
	return vnic, event, info, nil
}

func createDockerNetwork(ctx context.Context, info *libsnnet.ContainerInfo) error {
	cli, err := getDockerClient()
	if err != nil {
		return err
	}

	_, err = cli.NetworkCreate(ctx, types.NetworkCreate{
		Name:   info.SubnetID,
		Driver: "ciao",
		IPAM: network.IPAM{
			Driver: "ciao",
			Config: []network.IPAMConfig{{
				Subnet:  info.Subnet.String(),
				Gateway: info.Gateway.String(),
			}}},
		Options: map[string]string{
			"bridge": info.Bridge,
		}})

	if err != nil {
		glog.Errorf("Unable to create docker network %s: %v", info.SubnetID, err)
	}

	return err
}

func destroyDockerNetwork(ctx context.Context, bridge string) error {
	cli, err := getDockerClient()
	if err != nil {
		return err
	}

	err = cli.NetworkRemove(ctx, bridge)
	if err != nil {
		glog.Errorf("Unable to remove docker network %s: %v", bridge, err)
	}

	return err
}

func resetDockerNetworking() {
	cli, err := getDockerClient()
	if err != nil {
		return
	}

	nets, err := cli.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		glog.Errorf("Unable to retrieve list of docker networks: %v", err)
		return
	}

	for i := range nets {
		if nets[i].Driver == "ciao" {
			glog.Infof("Deleting docker network %s", nets[i].Name)
			err = cli.NetworkRemove(context.Background(), nets[i].ID)
			if err != nil {
				glog.Errorf("Unable to remove docker network %s: %v", nets[i].ID, err)
			}
		}
	}

	return
}
