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
	"context"
	"fmt"

	"github.com/01org/ciao/bat"
)

func getWorkloadUUIDs(ctx context.Context, name string) ([]string, error) {
	wls, err := bat.GetAllWorkloads(ctx, "")
	var ids []string
	if err != nil {
		return ids, fmt.Errorf("Failed to retrieve workload list")
	}

	for _, w := range wls {
		if w.Name == name {
			ids = append(ids, w.ID)
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("workload %s not found", name)
	}

	return ids, nil
}

func getWorkloadUUID(ctx context.Context, name string) (string, error) {
	ids, err := getWorkloadUUIDs(ctx, name)
	if err != nil {
		return "", err
	}
	if len(ids) > 1 {
		return "", fmt.Errorf("Multiple workloads with the same name (%s) found", name)
	}

	return ids[0], nil
}
