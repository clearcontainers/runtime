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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
)

const (
	masterWorkloadName = "k8s master"
	workerWorkloadName = "k8s worker"
)

type usageError string

func (e usageError) Error() string {
	return string(e)
}

type vmOptions struct {
	vCPUs   int
	memMiB  int
	diskGiB int
}

type options struct {
	masterVM      vmOptions
	workerVM      vmOptions
	user          string
	publicKeyPath string
	workers       int
	imageUUID     string
	externalIP    string
	keep          bool
}

type baseConfig struct {
	VCPUs        int
	RAMMiB       int
	DiskGiB      int
	User         string
	ImageUUID    string
	HTTPSProxy   string
	HTTPProxy    string
	NoProxy      string
	Token        string
	PublicKey    string
	UserDataFile string
	Description  string
}

type proxyConfig struct {
	httpProxy  string
	httpsProxy string
	noProxy    string
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s create image-uuid [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s delete\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "- create : creates a k8s cluster")
		fmt.Fprintln(os.Stderr, "- delete : removes kubicle created instances and workloads")
	}
}

func createFlags() (*options, error) {
	opts := options{
		masterVM: vmOptions{
			vCPUs:   1,
			memMiB:  1024,
			diskGiB: 10,
		},
		workerVM: vmOptions{
			vCPUs:   1,
			memMiB:  2048,
			diskGiB: 10,
		},
		workers: 1,
	}

	opts.user = os.Getenv("USER")
	home := os.Getenv("HOME")
	if home != "" {
		opts.publicKeyPath = path.Join(home, "local", "testkey.pub")
	}

	fs := flag.NewFlagSet("create", flag.ExitOnError)

	fs.IntVar(&opts.masterVM.memMiB, "mmem", opts.masterVM.memMiB,
		"Mebibytes of RAM allocated to master VM")
	fs.IntVar(&opts.masterVM.vCPUs, "mcpus", opts.masterVM.vCPUs, "VCPUs assignged to master VM")
	fs.IntVar(&opts.masterVM.diskGiB, "mdisk", opts.masterVM.diskGiB,
		"Gibibytes of disk allocated to master VM")

	fs.IntVar(&opts.workerVM.memMiB, "wmem", opts.workerVM.memMiB,
		"Mebibytes of RAM allocated to worker VMs")
	fs.IntVar(&opts.workerVM.vCPUs, "wcpus", opts.workerVM.vCPUs, "VCPUs assignged to worker VM")
	fs.IntVar(&opts.workerVM.diskGiB, "wdisk", opts.workerVM.diskGiB,
		"Gibibytes of disk allocated to worker VMs")

	fs.IntVar(&opts.workers, "workers", opts.workers, "Number of worker nodes to create")

	fs.StringVar(&opts.publicKeyPath, "key", opts.publicKeyPath, "Path to public key used to ssh into nodes")
	fs.StringVar(&opts.user, "user", opts.user, "Name of user account to create on the nodes")
	fs.StringVar(&opts.externalIP, "external-ip", opts.externalIP,
		"External-ip to associate with the master node")
	fs.BoolVar(&opts.keep, "keep", false, "Retains workload definition files if set to true")

	if err := fs.Parse(flag.Args()[1:]); err != nil {
		return nil, err
	}

	if len(fs.Args()) < 1 {
		return nil, usageError("No image-uuid specified!")
	}
	opts.imageUUID = fs.Args()[0]

	return &opts, nil
}
func runCommand(signalCh <-chan os.Signal) error {
	var err error

	errCh := make(chan error)
	ctx, cancelFunc := context.WithCancel(context.Background())
	switch os.Args[1] {
	case "create":
		go create(ctx, errCh)
	case "delete":
		go destroy(ctx, errCh)
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
	if len(flag.Args()) < 1 ||
		!(os.Args[1] == "create" || os.Args[1] == "delete") {
		flag.Usage()
		os.Exit(1)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	if err := runCommand(signalCh); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		if _, ok := err.(usageError); ok {
			fmt.Fprintln(os.Stderr)
			flag.Usage()
		}
		os.Exit(1)
	}
}
