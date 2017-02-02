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

package payloads

// ServiceType is used to define OpenStack service types, like e.g. the image
// or identity ones.
type ServiceType string

// StorageType is used to define the configuration backend storage type.
type StorageType string

const (
	// Glance is used to define the imaging service.
	Glance ServiceType = "glance"

	// Keystone is used to define the identity service.
	Keystone ServiceType = "keystone"
)

const (
	// Filesystem defines the local filesystem backend storage type for the
	// configuration data.
	Filesystem StorageType = "file"
)

func (s ServiceType) String() string {
	switch s {
	case Glance:
		return "glance"
	case Keystone:
		return "keystone"
	}

	return ""
}

func (s StorageType) String() string {
	switch s {
	case Filesystem:
		return "file"
	}

	return ""
}

// ConfigureScheduler contains the unmarshalled configurations for the
// scheduler service.
type ConfigureScheduler struct {
	ConfigStorageURI string `yaml:"storage_uri"`
}

// ConfigureController contains the unmarshalled configurations for the
// controller service.
type ConfigureController struct {
	VolumePort       int    `yaml:"volume_port"`
	ComputePort      int    `yaml:"compute_port"`
	CiaoPort         int    `yaml:"ciao_port"`
	HTTPSCACert      string `yaml:"compute_ca"`
	HTTPSKey         string `yaml:"compute_cert"`
	IdentityUser     string `yaml:"identity_user"`
	IdentityPassword string `yaml:"identity_password"`
}

// ConfigureLauncher contains the unmarshalled configurations for the
// launcher service.
type ConfigureLauncher struct {
	ComputeNetwork    []string `yaml:"compute_net"`
	ManagementNetwork []string `yaml:"mgmt_net"`
	DiskLimit         bool     `yaml:"disk_limit"`
	MemoryLimit       bool     `yaml:"mem_limit"`
}

// ConfigureStorage contains the unmarshalled configurations for the
// Ceph storage driver.
type ConfigureStorage struct {
	CephID string `yaml:"ceph_id"`
}

// ConfigureService contains the unmarshalled configurations for the resources
// of the configurations.
type ConfigureService struct {
	Type ServiceType `yaml:"type"`
	URL  string      `yaml:"url"`
}

// ConfigurePayload is a wrapper to read and unmarshall all posible
// configurations for the following services: scheduler, controller, launcher,
//  imaging and identity.
type ConfigurePayload struct {
	Scheduler       ConfigureScheduler  `yaml:"scheduler"`
	Storage         ConfigureStorage    `yaml:"storage"`
	Controller      ConfigureController `yaml:"controller"`
	Launcher        ConfigureLauncher   `yaml:"launcher"`
	ImageService    ConfigureService    `yaml:"image_service"`
	IdentityService ConfigureService    `yaml:"identity_service"`
}

// Configure represents the SSNTP CONFIGURE command payload.
type Configure struct {
	Configure ConfigurePayload `yaml:"configure"`
}

// InitDefaults initializes default vaulues for Configure structure.
func (conf *Configure) InitDefaults() {
	conf.Configure.Controller.VolumePort = 8776
	conf.Configure.Controller.ComputePort = 8774
	conf.Configure.Controller.CiaoPort = 8889
	conf.Configure.ImageService.Type = Glance
	conf.Configure.IdentityService.Type = Keystone
	conf.Configure.Launcher.DiskLimit = true
	conf.Configure.Launcher.MemoryLimit = true
}
