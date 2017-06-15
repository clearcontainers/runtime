//
// Copyright © 2016 Intel Corporation
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

package osprepare

import (
	"context"
	"strings"

	"github.com/01org/ciao/clogger"
)

// Minimal versions supported by ciao
const (
	MinDockerVersion = "1.11.0"
	MinQemuVersion   = "2.5.0"
)

// PackageRequirement contains the BinaryName expected to
// exist on the filesystem once PackageName is installed
// (e.g: { '/usr/bin/qemu-system-x86_64', 'qemu'})
type PackageRequirement struct {
	BinaryName  string
	PackageName string
}

// PackageRequirements type allows to create complex
// mapping to group a set of PackageRequirement to a single
// key.
// (e.g:
//
//	"ubuntu": {
//		{"/usr/bin/docker", "docker"},
//	},
//	"clearlinux": {
//		{"/usr/bin/docker", "containers-basic"},
//	},
// )
type PackageRequirements map[string][]PackageRequirement

// NewPackageRequirements is a PackageRequirements constructor
func NewPackageRequirements() PackageRequirements {
	return make(PackageRequirements)
}

//TODO: once minimum required package version is a part of type
//      PackageRequirement, then keep the larger of two listed
//      version numbers.  So rather than collectPackages()
//      having a check for dup's and "continue", you'd end there
//      with a call to:
//
//Deduplicate prunes duplicates in a PackageRequirements list,
//keeping the one instance amount duplicate BinaryName/PackageName
//with the highest version number.
//func (cur *PackageRequirements) Deduplicate(reqs PackageRequirements)

//Append a list of PackageRequirements to a PackageRequirements list
func (cur *PackageRequirements) Append(newReqs PackageRequirements) {
	curReqs := *cur

	for distro, reqList := range newReqs {
		// assume a deduplication happens once later on the
		// instance of PackageRequirements and simply append
		// here for efficiency
		curReqs[distro] = append(curReqs[distro], reqList...)
	}
}

// BootstrapRequirements lists required dependencies for absolutely core
// functionality across all Ciao components
var BootstrapRequirements = PackageRequirements{
	"ubuntu": {
		{"/usr/bin/cephfs", "ceph-fs-common"},
		{"/usr/bin/ceph", "ceph-common"},
	},
	"fedora": {
		{"/usr/bin/cephfs", "ceph"},
		{"/usr/bin/ceph", "ceph-common"},
	},
	"clearlinux": {
		{"/usr/bin/ceph", "storage-cluster"},
	},
}

// CollectPackages returns a list of non-installed packages from
// the PackageRequirements received
func collectPackages(dist distro, reqs PackageRequirements) []string {
	// For now just support keys like "ubuntu" vs "ubuntu:16.04"
	var pkgsMissing []string
	if reqs == nil {
		return nil
	}

	id := dist.getID()
	if pkgs, success := reqs[id]; success {
		for _, pkg := range pkgs {
			// skip empties
			if pkg.BinaryName == "" || pkg.PackageName == "" {
				continue
			}

			// Have the path existing, skip.
			if pathExists(pkg.BinaryName) {
				continue
			}
			// Mark the package for installation
			pkgsMissing = append(pkgsMissing, pkg.PackageName)
		}
		return pkgsMissing
	}
	return nil
}

// InstallDeps installs all the dependencies defined in a component
// specific PackageRequirements in order to enable running the component
func InstallDeps(ctx context.Context, reqs PackageRequirements, logger clogger.CiaoLog) {
	if logger == nil {
		logger = clogger.CiaoNullLogger{}
	}

	distro := getDistro()

	if distro == nil {
		logger.Errorf("Running on an unsupported distro")
		if rel := getOSRelease(); rel != nil {
			logger.Errorf("Unsupported distro: %s %s", rel.Name, rel.Version)
		} else {
			logger.Errorf("No os-release found on this host")
		}
		return
	}
	logger.Infof("OS Detected: %s", distro.getID())

	// Nothing requested to install
	if reqs == nil {
		return
	}
	if reqPkgs := collectPackages(distro, reqs); reqPkgs != nil {
		logger.Infof("Missing packages detected: %v", reqPkgs)
		if distro.InstallPackages(ctx, reqPkgs, logger) == false {
			logger.Errorf("Failed to install: %s", strings.Join(reqPkgs, ", "))
			return
		}
		logger.Infof("Missing packages installed.")
	}
}

// Bootstrap installs all the core dependencies required to bootstrap the core
// configuration of all Ciao components
func Bootstrap(ctx context.Context, logger clogger.CiaoLog) {
	InstallDeps(ctx, BootstrapRequirements, logger)
}
