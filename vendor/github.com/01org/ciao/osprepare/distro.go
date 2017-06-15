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
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/01org/ciao/clogger"
)

// pathExists is a helper function which handles the
// error and simply return true or false if the given
// path exists
func pathExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

type distro interface {
	// InstallPackages should implement the installation
	// of packages using distro specific methods for
	// the given target list of items to install
	InstallPackages(ctx context.Context, packages []string, logger clogger.CiaoLog) bool

	// getID should return a string specifying
	// the distribution ID (e.g: "clearlinux")
	getID() string
}

// getDistro will return a distro based on what
// is read from GetOsRelease
func getDistro() distro {
	osRelease := getOSRelease()

	if osRelease == nil {
		return nil
	}

	if strings.HasPrefix(osRelease.ID, "clear-linux") {
		return &clearLinuxDistro{}
	} else if strings.Contains(osRelease.ID, "ubuntu") {
		// Store the Ubuntu codename, i.e. "xenial'
		return &ubuntuDistro{CodeName: osRelease.GetValue("UBUNTU_CODENAME")}
	} else if strings.Contains(osRelease.ID, "fedora") {
		return &fedoraDistro{}
	}
	return nil
}

// os-release clear-linux*
type clearLinuxDistro struct {
}

func (d *clearLinuxDistro) getID() string {
	return "clearlinux"
}

// Correctly split and format the command, using sudo if appropriate, as a
// common mechanism for the various package install functions.
func sudoFormatCommand(ctx context.Context, command string, packages []string, logger clogger.CiaoLog) bool {
	var executable string
	var args string

	toInstall := strings.Join(packages, " ")
	splits := strings.Split(command, " ")

	if syscall.Geteuid() == 0 {
		executable = splits[0]
		args = fmt.Sprintf(strings.Join(splits[1:], " "), toInstall)
	} else {
		executable = "sudo"
		args = fmt.Sprintf(command, toInstall)
	}

	c := exec.CommandContext(ctx, executable, strings.Split(args, " ")...)
	read, err := c.StdoutPipe()
	if err != nil {
		logger.Warningf("Unable to create command output pipe: %s", err)
	}
	c.Stderr = c.Stdout
	err = c.Start()
	if err != nil {
		logger.Errorf("Unable to run command: %s", err)
		return false
	}
	scanner := bufio.NewScanner(read)
	for scanner.Scan() {
		logger.Infof(scanner.Text())
	}
	err = c.Wait()
	if err != nil {
		logger.Errorf("Error running command: %s", err)
		return false
	}

	return true
}

func (d *clearLinuxDistro) InstallPackages(ctx context.Context, packages []string, logger clogger.CiaoLog) bool {
	return sudoFormatCommand(ctx, "swupd bundle-add %s", packages, logger)
}

// os-release *ubuntu*
type ubuntuDistro struct {
	CodeName string
}

func (d *ubuntuDistro) getID() string {
	return "ubuntu"
}

func (d *ubuntuDistro) InstallPackages(ctx context.Context, packages []string, logger clogger.CiaoLog) bool {
	return sudoFormatCommand(ctx, "apt-get --yes --force-yes install %s", packages, logger)
}

// Fedora
type fedoraDistro struct {
}

func (d *fedoraDistro) getID() string {
	return "fedora"
}

// Use dnf to install on Fedora
func (d *fedoraDistro) InstallPackages(ctx context.Context, packages []string, logger clogger.CiaoLog) bool {
	return sudoFormatCommand(ctx, "dnf install -y %s", packages, logger)
}
