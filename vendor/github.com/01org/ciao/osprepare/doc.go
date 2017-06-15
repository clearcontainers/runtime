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

/*

Package osprepare is the Operating System Preparation facility of Ciao,
enabling very simple automated configuration of dependencies between
various Linux distributions in a sane fashion.


Expressing Dependencies

InstallDeps is used to ensure the presence of mandatory dependencies during
the early initialization of Ciao components, and install them via the native
package deployment system when they are absent.
InstallDeps should be invoked with a PackageRequirements map, which is simply
a mapping of operating system IDs to path/package pairs.
The function may not yet support all systems, so it is expected that it should
perform its function without any outside interference, and should never cause
failures or require checking of returns.

The PackageRequirements map (deps.go)

Below is an example of a valid PackageRequirements structure.

	var deps = osprepare.PackageRequirements{
		"ubuntu": {
			{"/usr/bin/docker", "docker"},
		},
		"clearlinux": {
			{"/usr/bin/docker", "containers-basic"},
		},
	}

As we can see, full paths to the expected locations on the filesystem are given
for each OS, and the name of the native package or bundle to install should those
files be missing. That is to say, if /usr/bin/docker is missing, on a system that
has been identified as Ubuntu, then the 'docker' package would be installed.

The dependencies for each component should be placed into a `deps.go` file, which
then facilitates easier maintenance and discoverability of the dependencies for
every component.

Verified lack of dependencies

Note that it is also valid to state you have *no* dependencies at all, and that
you have verified that each distro requires no additional dependencies to be
installed during startup. Just pass an empty pair to the distro ID:

	var schedDeps = osprepare.PackageRequirements{
		"clearlinux": {
			{"", ""},
		},
		"fedora": {
			{"", ""},
		},
		"ubuntu": {
			{"", ""},
		},
	}

Operating System Identification

Operating Systems are identified via their `os-release` file, a standardized
mechanism for identifying Linux distributions. Currently, detection is performed
by comparison the lower-case value of the `ID` field to our known implementations,
which are:

 * clearlinux
 * ubuntu
 * fedora

*/
package osprepare
