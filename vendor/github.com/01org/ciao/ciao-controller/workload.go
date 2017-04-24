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

package main

import (
	"github.com/golang/glog"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp/uuid"
)

func validateVMWorkload(req types.Workload) error {
	// FWType must be either EFI or legacy.
	if req.FWType != string(payloads.EFI) && req.FWType != payloads.Legacy {
		return types.ErrBadRequest
	}

	// Must have storage for VMs
	if len(req.Storage) == 0 {
		return types.ErrBadRequest
	}

	return nil
}

func validateContainerWorkload(req types.Workload) error {
	// we should reject anything with ImageID set, but
	// we'll just ignore it.
	if req.ImageName == "" {
		return types.ErrBadRequest
	}

	return nil
}

func validateWorkloadStorage(req types.Workload) error {
	bootableCount := 0
	for i := range req.Storage {
		// check that a workload type is specified
		if req.Storage[i].SourceType == "" {
			return types.ErrBadRequest
		}

		// you may not request a sized volume unless it's empty.
		if req.Storage[i].Size > 0 && req.Storage[i].SourceType != types.Empty {
			return types.ErrBadRequest
		}

		// you may not request a bootable empty volume.
		if req.Storage[i].Bootable && req.Storage[i].SourceType == types.Empty {
			return types.ErrBadRequest
		}

		if req.Storage[i].ID != "" {
			// validate that the id is at least valid
			// uuid4.
			_, err := uuid.Parse(req.Storage[i].ID)
			if err != nil {
				return types.ErrBadRequest
			}

			// If we have an ID we must have a type to get it from
			if req.Storage[i].SourceType != types.Empty {
				return types.ErrBadRequest
			}
		}

		if req.Storage[i].SourceID == "" {
			// you may only use no source id with empty type
			if req.Storage[i].SourceType != types.Empty {
				return types.ErrBadRequest
			}
		}

		if req.Storage[i].Bootable {
			bootableCount++
		}
	}

	// must be at least one bootable volume
	if req.VMType == payloads.QEMU && bootableCount == 0 {
		return types.ErrBadRequest
	}

	return nil
}

// this is probably an insufficient amount of checking.
func validateWorkloadRequest(req types.Workload) error {
	// ID must be blank.
	if req.ID != "" {
		glog.V(2).Info("Invalid workload request: ID is not blank")
		return types.ErrBadRequest
	}

	// we don't validate the TenantID right now - it is passed
	// in via the ciao api, and it has passed the regex input
	// validation already. there's also a conflict with ssntp's uuid.Parse()
	// function where they assume you are using a uuid4 with '-' as
	// separator, and keystone doesn't use the '-' separator for
	// uuids.

	if req.VMType == payloads.QEMU {
		err := validateVMWorkload(req)
		if err != nil {
			glog.V(2).Info("Invalid workload request: invalid VM workload")
			return err
		}
	} else {
		err := validateContainerWorkload(req)
		if err != nil {
			glog.V(2).Info("Invalid workload request: invalid container workload")
			return err
		}
	}

	if req.ImageID != "" {
		// validate that the image id is at least valid
		// uuid4.
		_, err := uuid.Parse(req.ImageID)
		if err != nil {
			glog.V(2).Info("Invalid workload request: ImageID is not uuid4")
			return types.ErrBadRequest
		}
	}

	if req.Config == "" {
		glog.V(2).Info("Invalid workload request: config is blank")
		return types.ErrBadRequest
	}

	if len(req.Storage) > 0 {
		err := validateWorkloadStorage(req)
		if err != nil {
			glog.V(2).Info("Invalid workload request: invalid storage")
			return err
		}
	}

	return nil
}

func (c *controller) CreateWorkload(req types.Workload) (types.Workload, error) {
	err := validateWorkloadRequest(req)
	if err != nil {
		return req, err
	}

	// check to see if this is a new tenant. If so, we need to add
	// them to the datastore. We do not however want to launch a
	// CNCI yet since this might be a request to upload a CNCI
	// workload. Instead, we'll add the new tenant directly to the
	// datastore. On first launch request, if the tenant doesn't yet
	// have a cnci, it will get launched for them then.
	tenant, err := c.ds.GetTenant(req.TenantID)
	if err != nil {
		return req, err
	}

	if tenant == nil {
		_, err := c.ds.AddTenant(req.TenantID)
		if err != nil {
			return req, err
		}
	}

	// create a workload storage resource for this new workload.
	if req.ImageID != "" {
		// validate that the image id is at least valid
		// uuid4.
		_, err = uuid.Parse(req.ImageID)
		if err != nil {
			return req, err
		}

		storage := types.StorageResource{
			Bootable:   true,
			Ephemeral:  true,
			SourceType: types.ImageService,
			SourceID:   req.ImageID,
		}

		req.ImageID = ""
		req.Storage = append(req.Storage, storage)
	}

	req.ID = uuid.Generate().String()

	err = c.ds.AddWorkload(req)
	return req, err
}
