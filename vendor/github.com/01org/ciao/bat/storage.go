//
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
//

package bat

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"
)

// VolumeOptions contains user supplied volume meta data
type VolumeOptions struct {
	Size        int
	Name        string
	Description string
}

// Volume contains information about a single volume
type Volume struct {
	VolumeOptions
	ID        string
	TenantID  string `json:"os-vol-tenant-attr:tenant_id"`
	CreatedAt string
	Status    string
}

// GetVolume returns a Volume structure containing information about a specific
// volume.  The information is retrieved by calling ciao-cli volume show.  An
// error will be returned if the following environment variables are not set;
// CIAO_IDENTITY,  CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func GetVolume(ctx context.Context, tenant, ID string) (*Volume, error) {
	var vol *Volume
	args := []string{"volume", "show", "--volume", ID, "-f", "{{tojson .}}"}
	err := RunCIAOCLIJS(ctx, tenant, args, &vol)
	if err != nil {
		return nil, err
	}

	return vol, nil
}

// DeleteVolume deletes the specified volume from a given tenant.  The volume
// is deleted by calling ciao-cli volume delete.  An error will be returned
// if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func DeleteVolume(ctx context.Context, tenant, ID string) error {
	args := []string{"volume", "delete", "--volume", ID}
	_, err := RunCIAOCLI(ctx, tenant, args)
	return err
}

// AddVolume adds a new volume to a tenant.  The volume is added using
// ciao-cli volume add.  An error will be returned if the following environment
// variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_USERNAME,
// CIAO_PASSWORD.
func AddVolume(ctx context.Context, tenant, source, sourceType string,
	options *VolumeOptions) (string, error) {
	args := []string{"volume", "add"}

	if sourceType != "" {
		if source == "" {
			panic("sourceType supplied but source is empty")
		}
		args = append(args, "--source_type", sourceType)
	}

	if source != "" {
		args = append(args, "--source", source)
	}

	if options.Name != "" {
		args = append(args, "--name", options.Name)
	}

	if options.Description != "" {
		args = append(args, "--description", options.Description)
	}

	if options.Size != 0 {
		args = append(args, "--size", fmt.Sprintf("%d", options.Size))
	}

	data, err := RunCIAOCLI(ctx, tenant, args)
	if err != nil {
		return "", err
	}

	split := bytes.Split(data, []byte{':'})
	if len(split) != 2 {
		return "", fmt.Errorf("unable to determine id of created volume")
	}

	return strings.TrimSpace(string(split[1])), nil
}

// GetAllVolumes returns a map of all the volumes defined in the specified
// tenant.  The map is indexed by volume ID.  The map is retrieved by calling
// ciao-cli volume list.  An error will be returned if the following environment
// variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_USERNAME,
// CIAO_PASSWORD.
func GetAllVolumes(ctx context.Context, tenant string) (map[string]*Volume, error) {
	var volumes map[string]*Volume

	template := `
{
{{- range $i, $val := .}}
  {{- if $i }},{{end}}
  "{{$val.ID | js }}" : {{tojson $val}}
{{- end }}
}
`
	args := []string{"volume", "list", "-f", template}
	err := RunCIAOCLIJS(ctx, tenant, args, &volumes)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}

// WaitForVolumeStatus blocks until the status of the specified volume matches
// the status parameter or the context is cancelled.    An error will be returned
// if the following environment variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
func WaitForVolumeStatus(ctx context.Context, tenant, volume, status string) error {
	for {
		vol, err := GetVolume(ctx, tenant, volume)
		if err != nil {
			return fmt.Errorf("Unable to retrieve meta data for volume %s :%v",
				volume, err)
		}
		if status == vol.Status {
			break
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("Test timed out waiting for volume %s status=%s",
				volume, status)
		case <-time.After(time.Second):
		}
	}

	return nil
}

// AttachVolume attaches a volume to an instance.    An error will be returned
// if the following environment variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
func AttachVolume(ctx context.Context, tenant, instance, volume string) error {
	args := []string{"volume", "attach", "--volume", volume,
		"--instance", instance}
	_, err := RunCIAOCLI(ctx, tenant, args)
	return err
}

// AttachVolumeAndWait attaches a volume to an instance and waits for the status of that
// volume to transition to "in-use".  An error will be returned if the following environment
// variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func AttachVolumeAndWait(ctx context.Context, tenant, instance, volume string) error {
	err := AttachVolume(ctx, tenant, instance, volume)
	if err != nil {
		return err
	}
	return WaitForVolumeStatus(ctx, tenant, volume, "in-use")
}

// DetachVolume detaches a volume from an instance.    An error will be returned
// if the following environment variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
func DetachVolume(ctx context.Context, tenant, volume string) error {
	args := []string{"volume", "detach", "--volume", volume}
	_, err := RunCIAOCLI(ctx, tenant, args)
	return err
}

// DetachVolumeAndWait attaches a volume to an instance and waits for the status of that
// volume to transition to "available".  An error will be returned if the following environment
// variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func DetachVolumeAndWait(ctx context.Context, tenant, volume string) error {
	err := DetachVolume(ctx, tenant, volume)
	if err != nil {
		return err
	}
	return WaitForVolumeStatus(ctx, tenant, volume, "available")
}
