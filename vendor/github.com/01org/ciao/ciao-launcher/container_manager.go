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
	"io"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/network"
	"golang.org/x/net/context"
)

type containerManager interface {
	ImageList(context.Context, types.ImageListOptions) ([]types.Image, error)
	ImagePull(context.Context, types.ImagePullOptions, client.RequestPrivilegeFunc) (io.ReadCloser, error)
	ContainerCreate(context.Context, *container.Config, *container.HostConfig,
		*network.NetworkingConfig, string) (types.ContainerCreateResponse, error)
	ContainerRemove(context.Context, types.ContainerRemoveOptions) error
	ContainerStart(context.Context, string) error
	ContainerInspect(context.Context, string) (types.ContainerJSON, error)
	ContainerInspectWithRaw(context.Context, string, bool) (types.ContainerJSON, []byte, error)
	ContainerStats(context.Context, string, bool) (io.ReadCloser, error)
	ContainerKill(context.Context, string, string) error
	ContainerWait(context.Context, string) (int, error)
}
