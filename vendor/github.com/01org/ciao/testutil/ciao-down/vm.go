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

package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"text/tabwriter"
	"time"

	"github.com/01org/ciao/qemu"
)

func bootVM(ctx context.Context, ws *workspace, memGB, CPUs int) error {
	disconnectedCh := make(chan struct{})
	socket := path.Join(ws.instanceDir, "socket")
	qmp, _, err := qemu.QMPStart(ctx, socket, qemu.QMPConfig{}, disconnectedCh)
	if err == nil {
		qmp.Shutdown()
		return fmt.Errorf("VM is already running")
	}

	vmImage := path.Join(ws.instanceDir, "image.qcow2")
	isoPath := path.Join(ws.instanceDir, "config.iso")
	memParam := fmt.Sprintf("%dG", memGB)
	CPUsParam := fmt.Sprintf("cpus=%d", CPUs)
	fsdevParam := fmt.Sprintf("local,security_model=passthrough,id=fsdev0,path=%s",
		ws.GoPath)
	args := []string{
		"-qmp", fmt.Sprintf("unix:%s,server,nowait", socket),
		"-m", memParam, "-smp", CPUsParam,
		"-drive", fmt.Sprintf("file=%s,if=virtio,aio=threads,format=qcow2", vmImage),
		"-drive", fmt.Sprintf("file=%s,if=virtio,media=cdrom", isoPath),
		"-daemonize", "-enable-kvm", "-cpu", "host",
		"-net", "user,hostfwd=tcp::10022-:22,hostfwd=tcp::3000-:3000",
		"-net", "nic,model=virtio",
		"-fsdev", fsdevParam,
		"-device", "virtio-9p-pci,id=fs0,fsdev=fsdev0,mount_tag=hostgo",
	}
	if ws.UIPath != "" {
		fsdevParam := fmt.Sprintf("local,security_model=passthrough,id=fsdev1,path=%s",
			ws.UIPath)
		args = append(args, "-fsdev", fsdevParam)
		args = append(args, "-device", "virtio-9p-pci,id=fs1,fsdev=fsdev1,mount_tag=hostui")
	}
	args = append(args, "-display", "none", "-vga", "none")

	output, err := qemu.LaunchCustomQemu(ctx, "", args, nil, nil)
	if err != nil {
		return fmt.Errorf("Failed to launch qemu : %v, %s", err, output)
	}
	return nil
}

func executeQMPCommand(ctx context.Context, instanceDir string,
	cmd func(ctx context.Context, q *qemu.QMP) error) error {
	socket := path.Join(instanceDir, "socket")
	disconnectedCh := make(chan struct{})
	qmp, _, err := qemu.QMPStart(ctx, socket, qemu.QMPConfig{}, disconnectedCh)
	if err != nil {
		return fmt.Errorf("Failed to connect to VM : %v", err)
	}
	defer qmp.Shutdown()

	err = qmp.ExecuteQMPCapabilities(ctx)
	if err != nil {
		return fmt.Errorf("Unable to query QEMU caps : %v", err)
	}

	err = cmd(ctx, qmp)
	if err != nil {
		return fmt.Errorf("Unable to execute vm command : %v", err)
	}

	return nil
}

func stopVM(ctx context.Context, instanceDir string) error {
	return executeQMPCommand(ctx, instanceDir, func(ctx context.Context, q *qemu.QMP) error {
		return q.ExecuteSystemPowerdown(ctx)
	})
}

func quitVM(ctx context.Context, instanceDir string) error {
	return executeQMPCommand(ctx, instanceDir, func(ctx context.Context, q *qemu.QMP) error {
		return q.ExecuteQuit(ctx)
	})
}

func sshReady(ctx context.Context) bool {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", "127.0.0.1:10022")
	if err != nil {
		return false
	}
	_ = conn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
	scanner := bufio.NewScanner(conn)
	retval := scanner.Scan()
	_ = conn.Close()
	return retval
}

func vmStarted(ctx context.Context, instanceDir string) bool {
	socket := path.Join(instanceDir, "socket")
	disconnectedCh := make(chan struct{})
	qmp, _, err := qemu.QMPStart(ctx, socket, qemu.QMPConfig{}, disconnectedCh)
	if err != nil {
		return false
	}
	qmp.Shutdown()
	return true
}

func statusVM(ctx context.Context, instanceDir, keyPath string) {
	status := "ciao down"
	ssh := "N/A"
	if vmStarted(ctx, instanceDir) {
		if sshReady(ctx) {
			status = "ciao up"
			ssh = fmt.Sprintf("ssh -q -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i %s 127.0.0.1 -p %d", keyPath, 10022)
		} else {
			status = "ciao up (booting)"
		}
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintf(w, "Status\t:\t%s\n", status)
	fmt.Fprintf(w, "SSH\t:\t%s\n", ssh)
	w.Flush()
}

func startHTTPServer(listener net.Listener, errCh chan error) {
	finished := false
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var b bytes.Buffer
		_, err := io.Copy(&b, r.Body)
		if err != nil {
			// TODO: Figure out what to do here
			return
		}
		line := string(b.Bytes())
		if line == "FINISHED" {
			_ = listener.Close()
			finished = true
			return
		}
		if line == "OK" || line == "FAIL" {
			fmt.Printf("[%s]\n", line)
		} else {
			fmt.Printf("%s : ", line)
		}
	})

	server := &http.Server{}
	go func() {
		_ = server.Serve(listener)
		if finished {
			errCh <- nil
		} else {
			errCh <- fmt.Errorf("HTTP server exited prematurely")
		}
	}()
}

func manageInstallation(ctx context.Context, instanceDir string, ws *workspace) error {
	socket := path.Join(instanceDir, "socket")
	disconnectedCh := make(chan struct{})

	qmp, _, err := qemu.QMPStart(ctx, socket, qemu.QMPConfig{}, disconnectedCh)
	if err != nil {
		return fmt.Errorf("Unable to connect to VM : %v", err)
	}

	qemuShutdown := true
	defer func() {
		if qemuShutdown {
			ctx, cancelFn := context.WithTimeout(context.Background(), time.Second)
			_ = qmp.ExecuteQuit(ctx)
			<-disconnectedCh
			cancelFn()
		}
		qmp.Shutdown()
	}()

	err = qmp.ExecuteQMPCapabilities(ctx)
	if err != nil {
		return fmt.Errorf("Unable to query QEMU caps")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", ws.HTTPServerPort))
	if err != nil {
		return fmt.Errorf("Unable to create listener: %v", err)
	}

	errCh := make(chan error)
	startHTTPServer(listener, errCh)
	select {
	case <-ctx.Done():
		_ = listener.Close()
		<-errCh
		return ctx.Err()
	case err := <-errCh:
		if err == nil {
			qemuShutdown = false
		}
		return err
	case <-disconnectedCh:
		qemuShutdown = false
		_ = listener.Close()
		<-errCh
		return fmt.Errorf("Lost connection to QEMU instance")
	}
}
