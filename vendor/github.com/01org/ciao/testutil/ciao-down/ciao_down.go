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

/* TODO

5. Install kernel
12. Make most output from osprepare optional
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s [prepare|start|stop|quit|status|connect|delete]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "- prepare : creates a new VM")
		fmt.Fprintln(os.Stderr, "- start : boots a stopped VM")
		fmt.Fprintln(os.Stderr, "- stop : cleanly powers down a running VM")
		fmt.Fprintln(os.Stderr, "- quit : quits a running VM")
		fmt.Fprintln(os.Stderr, "- status : prints status information about the ciao-down VM")
		fmt.Fprintln(os.Stderr, "- connect : connects to the VM via SSH")
		fmt.Fprintln(os.Stderr, "- delete : shuts down and deletes the VM")
	}
}

func vmFlags(fs *flag.FlagSet, memGB, CPUs *int) {
	*memGB, *CPUs = getMemAndCpus()
	fs.IntVar(memGB, "mem", *memGB, "Gigabytes of RAM allocated to VM")
	fs.IntVar(CPUs, "cpus", *CPUs, "VCPUs assignged to VM")
}

func checkDirectory(dir string) error {
	if dir == "" {
		return nil
	}

	if !path.IsAbs(dir) {
		return fmt.Errorf("%s is not an absolute path", dir)
	}

	fi, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("Unable to stat %s : %v", dir, err)
	}

	if !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	return nil
}

func prepareFlags() (memGB int, CPUs int, debug bool, uiPath, runCmd string, err error) {
	fs := flag.NewFlagSet("prepare", flag.ExitOnError)
	vmFlags(fs, &memGB, &CPUs)
	fs.BoolVar(&debug, "debug", false, "Enables debug mode")
	fs.StringVar(&uiPath, "ui-path", "", "Host path of cloned ciao-webui repo")
	fs.StringVar(&runCmd, "runcmd", "", "Path to a file containing additional commands to execute when preparing the VM")
	if err := fs.Parse(flag.Args()[1:]); err != nil {
		return -1, -1, false, "", "", err
	}

	if err := checkDirectory(uiPath); err != nil {
		return -1, -1, false, "", "", err
	}

	if uiPath != "" {
		uiPath = filepath.Clean(uiPath)
	}

	return memGB, CPUs, debug, uiPath, runCmd, nil
}

func startFlags() (memGB int, CPUs int, err error) {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	vmFlags(fs, &memGB, &CPUs)
	if err := fs.Parse(flag.Args()[1:]); err != nil {
		return -1, -1, err
	}

	return memGB, CPUs, nil
}

func downloadProgress(p progress) {
	if p.totalMB >= 0 {
		fmt.Printf("Downloaded %d MB of %d\n", p.downloadedMB, p.totalMB)
	} else {
		fmt.Printf("Downloaded %d MB\n", p.downloadedMB)
	}
}

func prepare(ctx context.Context, errCh chan error) {
	var err error

	defer func() {
		errCh <- err
	}()

	if !hostSupportsNestedKVM() {
		err = fmt.Errorf("nested KVM is not enabled.  Please enable and try again")
		return
	}

	fmt.Println("Checking environment")

	memGB, CPUs, debug, uiPath, runCmd, err := prepareFlags()
	if err != nil {
		return
	}

	ws, err := prepareEnv(ctx)
	if err != nil {
		return
	}

	_, err = os.Stat(ws.instanceDir)
	if err == nil {
		err = fmt.Errorf("instance already exists")
		return
	}

	fmt.Println("Installing host dependencies")
	installDeps(ctx)

	err = os.MkdirAll(ws.instanceDir, 0755)
	if err != nil {
		err = fmt.Errorf("unable to create cache dir: %v", err)
		return
	}

	defer func() {
		if err != nil {
			_ = os.RemoveAll(ws.instanceDir)
		}
	}()

	err = ioutil.WriteFile(path.Join(ws.instanceDir, "ui_path.txt"),
		[]byte(uiPath), 0600)
	if err != nil {
		err = fmt.Errorf("Unable to write ui_path.txt : %v", err)
		return
	}
	ws.UIPath = uiPath

	err = prepareSSHKeys(ctx, ws)
	if err != nil {
		return
	}

	err = prepareRunCmd(ws, runCmd)
	if err != nil {
		return
	}

	qcowPath, err := downloadUbuntu(ctx, ws.ciaoDir, downloadProgress)
	if err != nil {
		return
	}

	err = buildISOImage(ctx, ws.instanceDir, ws, debug)
	if err != nil {
		return
	}

	err = createRootfs(ctx, qcowPath, ws.instanceDir)
	if err != nil {
		return
	}

	fmt.Printf("Booting VM with %d GB RAM and %d cpus\n", memGB, CPUs)

	err = bootVM(ctx, ws, memGB, CPUs)
	if err != nil {
		return
	}

	err = manageInstallation(ctx, ws.instanceDir, ws)
	if err != nil {
		return
	}
	fmt.Println("VM successfully created!")
	fmt.Println("Type ciao-down connect to start using it.")
}

func start(ctx context.Context, errCh chan error) {
	if !hostSupportsNestedKVM() {
		errCh <- fmt.Errorf("nested KVM is not enabled.  Please enable and try again")
		return
	}

	memGB, CPUs, err := startFlags()
	if err != nil {
		errCh <- err
		return
	}

	ws, err := prepareEnv(ctx)
	if err != nil {
		errCh <- err
		return
	}

	fmt.Printf("Booting VM with %d GB RAM and %d cpus\n", memGB, CPUs)

	err = bootVM(ctx, ws, memGB, CPUs)
	if err != nil {
		errCh <- err
		return
	}

	fmt.Println("VM Started")

	errCh <- err
}

func stop(ctx context.Context, errCh chan error) {
	ws, err := prepareEnv(ctx)
	if err != nil {
		errCh <- err
		return
	}

	err = stopVM(ctx, ws.instanceDir)
	if err != nil {
		errCh <- err
		return
	}

	fmt.Println("VM Stopped")

	errCh <- err
}

func quit(ctx context.Context, errCh chan error) {
	ws, err := prepareEnv(ctx)
	if err != nil {
		errCh <- err
		return
	}

	err = quitVM(ctx, ws.instanceDir)
	if err != nil {
		errCh <- err
		return
	}

	fmt.Println("VM Quit")

	errCh <- err
}

func status(ctx context.Context, errCh chan error) {
	ws, err := prepareEnv(ctx)
	if err != nil {
		errCh <- err
		return
	}

	statusVM(ctx, ws.instanceDir, ws.keyPath)
	errCh <- err
}

func connect(ctx context.Context, errCh chan error) {
	ws, err := prepareEnv(ctx)
	if err != nil {
		errCh <- err
		return
	}

	path, err := exec.LookPath("ssh")
	if err != nil {
		errCh <- fmt.Errorf("Unable to locate ssh binary")
		return
	}

	if !vmStarted(ctx, ws.instanceDir) {
		errCh <- fmt.Errorf("VM is not running.  Try ciao-down start")
		return
	}

	if !sshReady(ctx) {
		fmt.Printf("Waiting for VM to boot ")
	DONE:
		for {
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
				errCh <- fmt.Errorf("Cancelled")
				return
			}

			if !vmStarted(ctx, ws.instanceDir) {
				errCh <- fmt.Errorf("VM is not running.  Try ciao-down start")
				return
			}

			if sshReady(ctx) {
				break DONE
			}

			fmt.Print(".")
		}
		fmt.Println()
	}

	err = syscall.Exec(path, []string{path,
		"-q", "-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
		"-i", ws.keyPath,
		"127.0.0.1", "-p", "10022"},
		os.Environ())
	errCh <- err
}

func delete(ctx context.Context, errCh chan error) {
	ws, err := prepareEnv(ctx)
	if err != nil {
		errCh <- err
		return
	}

	_ = quitVM(ctx, ws.instanceDir)
	err = os.RemoveAll(ws.instanceDir)
	if err != nil {
		errCh <- fmt.Errorf("unable to delete instance: %v", err)
		return
	}

	errCh <- nil
}

func runCommand(signalCh <-chan os.Signal) error {
	var err error

	errCh := make(chan error)
	ctx, cancelFunc := context.WithCancel(context.Background())
	switch os.Args[1] {
	case "prepare":
		go prepare(ctx, errCh)
	case "start":
		go start(ctx, errCh)
	case "stop":
		go stop(ctx, errCh)
	case "quit":
		go quit(ctx, errCh)
	case "status":
		go status(ctx, errCh)
	case "connect":
		go connect(ctx, errCh)
	case "delete":
		go delete(ctx, errCh)
	}
	select {
	case <-signalCh:
		cancelFunc()
		err = <-errCh
	case err = <-errCh:
		cancelFunc()
	}

	return err
}

func main() {
	flag.Parse()
	if len(os.Args) < 2 ||
		!(os.Args[1] == "prepare" || os.Args[1] == "start" || os.Args[1] == "stop" ||
			os.Args[1] == "quit" || os.Args[1] == "status" ||
			os.Args[1] == "connect" || os.Args[1] == "delete") {
		flag.Usage()
		os.Exit(1)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	if err := runCommand(signalCh); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
