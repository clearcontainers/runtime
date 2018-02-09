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

package tests

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// Docker command
	Docker = "docker"

	// Image used to run containers
	Image = "busybox"

	// AlpineImage is the alpine image
	AlpineImage = "alpine"

	// PostgresImage is the postgres image
	PostgresImage = "postgres"

	// DebianImage is the debian image
	DebianImage = "debian"

	// FedoraImage is the fedora image
	FedoraImage = "fedora"
)

func runDockerCommandWithTimeout(timeout time.Duration, command string, args ...string) (string, string, int) {
	return runDockerCommandWithTimeoutAndPipe(nil, timeout, command, args...)
}

func runDockerCommandWithTimeoutAndPipe(stdin *bytes.Buffer, timeout time.Duration, command string, args ...string) (string, string, int) {
	a := []string{command}
	a = append(a, args...)

	cmd := NewCommand(Docker, a...)
	cmd.Timeout = timeout

	return cmd.RunWithPipe(stdin)
}

func runDockerCommand(command string, args ...string) (string, string, int) {
	return runDockerCommandWithTimeout(time.Duration(Timeout), command, args...)
}

func runDockerCommandWithPipe(stdin *bytes.Buffer, command string, args ...string) (string, string, int) {
	return runDockerCommandWithTimeoutAndPipe(stdin, time.Duration(Timeout), command, args...)
}

// LogsDockerContainer returns the container logs
func LogsDockerContainer(name string) (string, error) {
	args := []string{name}

	stdout, _, exitCode := runDockerCommand("logs", args...)

	if exitCode != 0 {
		return "", fmt.Errorf("failed to run docker logs command")
	}

	return strings.TrimSpace(stdout), nil
}

// StatusDockerContainer returns the container status
func StatusDockerContainer(name string) string {
	args := []string{"-a", "-f", "name=" + name, "--format", "{{.Status}}"}

	stdout, _, exitCode := runDockerCommand("ps", args...)

	if exitCode != 0 || stdout == "" {
		return ""
	}

	state := strings.Split(stdout, " ")
	return state[0]
}

// hasExitedDockerContainer checks if the container has exited.
func hasExitedDockerContainer(name string) (bool, error) {
	args := []string{"--format={{.State.Status}}", name}

	stdout, _, exitCode := runDockerCommand("inspect", args...)

	if exitCode != 0 || stdout == "" {
		return false, fmt.Errorf("failed to run docker inspect command")
	}

	status := strings.TrimSpace(stdout)

	if status == "exited" {
		return true, nil
	}

	return false, nil
}

// ExitCodeDockerContainer returns the container exit code
func ExitCodeDockerContainer(name string, waitForExit bool) (int, error) {
	// It makes no sense to try to retrieve the exit code of the container
	// if it is still running. That's why this infinite loop takes care of
	// waiting for the status to become "exited" before to ask for the exit
	// code.
	// However, we might want to bypass this check on purpose, that's why
	// we check waitForExit boolean.
	if waitForExit {
		errCh := make(chan error)
		exitCh := make(chan bool)

		go func() {
			for {
				exited, err := hasExitedDockerContainer(name)
				if err != nil {
					errCh <- err
				}

				if exited {
					break
				}

				time.Sleep(time.Second)
			}

			close(exitCh)
		}()

		select {
		case <-exitCh:
			break
		case err := <-errCh:
			return -1, err
		case <-time.After(time.Duration(Timeout) * time.Second):
			return -1, fmt.Errorf("Timeout reached after %ds", Timeout)
		}
	}

	args := []string{"--format={{.State.ExitCode}}", name}

	stdout, _, exitCode := runDockerCommand("inspect", args...)

	if exitCode != 0 || stdout == "" {
		return -1, fmt.Errorf("failed to run docker inspect command")
	}

	return strconv.Atoi(strings.TrimSpace(stdout))
}

func WaitForRunningDockerContainer(name string, running bool) error {
	ch := make(chan bool)
	go func() {
		if IsRunningDockerContainer(name) == running {
			close(ch)
			return
		}

		time.Sleep(time.Second)
	}()

	select {
	case <-ch:
	case <-time.After(time.Duration(Timeout) * time.Second):
		return fmt.Errorf("Timeout reached after %ds", Timeout)
	}

	return nil
}

// IsRunningDockerContainer inspects a container
// returns true if is running
func IsRunningDockerContainer(name string) bool {
	stdout, _, exitCode := runDockerCommand("inspect", "--format={{.State.Running}}", name)

	if exitCode != 0 {
		return false
	}

	output := strings.TrimSpace(stdout)
	LogIfFail("container running: " + output)
	if output == "false" {
		return false
	}

	return true
}

// ExistDockerContainer returns true if any of next cases is true:
// - 'docker ps -a' command shows the container
// - the VM is running (qemu)
// else false is returned
func ExistDockerContainer(name string) bool {
	state := StatusDockerContainer(name)
	if state != "" {
		return true
	}

	return IsVMRunning(name)
}

// RemoveDockerContainer removes a container using docker rm -f
func RemoveDockerContainer(name string) bool {
	_, _, exitCode := DockerRm("-f", name)
	if exitCode != 0 {
		return false
	}

	return true
}

// StopDockerContainer stops a container
func StopDockerContainer(name string) bool {
	_, _, exitCode := DockerStop(name)
	if exitCode != 0 {
		return false
	}

	return true
}

// KillDockerContainer kills a container
func KillDockerContainer(name string) bool {
	_, _, exitCode := DockerKill(name)
	if exitCode != 0 {
		return false
	}

	return true
}

// DockerRm removes a container
func DockerRm(args ...string) (string, string, int) {
	return runDockerCommand("rm", args...)
}

// DockerStop stops a container
// returns true on success else false
func DockerStop(args ...string) (string, string, int) {
	// docker stop takes ~15 seconds
	return runDockerCommand("stop", args...)
}

// DockerPull downloads the specific image
func DockerPull(args ...string) (string, string, int) {
	// 10 minutes should be enough to download a image
	return runDockerCommandWithTimeout(600, "pull", args...)
}

// DockerRun runs a container
func DockerRun(args ...string) (string, string, int) {
	if Runtime != "" {
		args = append(args, []string{"", ""}...)
		copy(args[2:], args[:])
		args[0] = "--runtime"
		args[1] = Runtime
	}

	return runDockerCommand("run", args...)
}

// DockerRunWithPipe runs a container with stdin
func DockerRunWithPipe(stdin *bytes.Buffer, args ...string) (string, string, int) {
	if Runtime != "" {
		args = append(args, []string{"", ""}...)
		copy(args[2:], args[:])
		args[0] = "--runtime"
		args[1] = Runtime
	}

	return runDockerCommandWithPipe(stdin, "run", args...)
}

// DockerKill kills a container
func DockerKill(args ...string) (string, string, int) {
	return runDockerCommand("kill", args...)
}

// DockerVolume manages volumes
func DockerVolume(args ...string) (string, string, int) {
	return runDockerCommand("volume", args...)
}

// DockerAttach attach to a running container
func DockerAttach(args ...string) (string, string, int) {
	return runDockerCommand("attach", args...)
}

// DockerCommit creates a new image from a container's changes
func DockerCommit(args ...string) (string, string, int) {
	return runDockerCommand("commit", args...)
}

// DockerImages list images
func DockerImages(args ...string) (string, string, int) {
	return runDockerCommand("images", args...)
}

// DockerRmi removes one or more images
func DockerRmi(args ...string) (string, string, int) {
	// docker takes more than 5 seconds to remove an image, it depends
	// of the image size and this operation does not involve to the
	// runtime
	return runDockerCommand("rmi", args...)
}

// DockerCp copies files/folders between a container and the local filesystem
func DockerCp(args ...string) (string, string, int) {
	return runDockerCommand("cp", args...)
}

// DockerExec runs a command in a running container
func DockerExec(args ...string) (string, string, int) {
	return runDockerCommand("exec", args...)
}

// DockerPs list containers
func DockerPs(args ...string) (string, string, int) {
	return runDockerCommand("ps", args...)
}

// DockerSearch searchs docker hub images
func DockerSearch(args ...string) (string, string, int) {
	return runDockerCommand("search", args...)
}

// DockerCreate creates a new container
func DockerCreate(args ...string) (string, string, int) {
	return runDockerCommand("create", args...)
}

// DockerDiff inspect changes to files or directories on a container’s filesystem
func DockerDiff(args ...string) (string, string, int) {
	return runDockerCommand("diff", args...)
}

// DockerBuild builds an image from a Dockerfile
func DockerBuild(args ...string) (string, string, int) {
	return runDockerCommand("build", args...)
}

// DockerNetwork manages networks
func DockerNetwork(args ...string) (string, string, int) {
	return runDockerCommand("network", args...)
}

// DockerExport will export a container’s filesystem as a tar archive
func DockerExport(args ...string) (string, string, int) {
	return runDockerCommand("export", args...)
}

// DockerImport imports the contents from a tarball to create a filesystem image
func DockerImport(args ...string) (string, string, int) {
	return runDockerCommand("import", args...)
}

// DockerInfo displays system-wide information
func DockerInfo() (string, string, int) {
	return runDockerCommand("info")
}

// DockerSwarm manages swarm
func DockerSwarm(args ...string) (string, string, int) {
	return runDockerCommand("swarm", args...)
}

// DockerService manages services
func DockerService(args ...string) (string, string, int) {
	return runDockerCommand("service", args...)
}

// DockerStart starts one or more stopped containers
func DockerStart(args ...string) (string, string, int) {
	return runDockerCommand("start", args...)
}

// DockerPause pauses all processes within one or more containers
func DockerPause(args ...string) (string, string, int) {
	return runDockerCommand("pause", args...)
}

// DockerUnpause unpauses all processes within one or more containers
func DockerUnpause(args ...string) (string, string, int) {
	return runDockerCommand("unpause", args...)
}

// DockerTop displays the running processes of a container
func DockerTop(args ...string) (string, string, int) {
	return runDockerCommand("top", args...)
}
