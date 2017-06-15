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
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/01org/ciao/bat"
	"github.com/vishvananda/netlink"
)

const externalIPPool = "k8s-pool"

var masterUDTmpl *template.Template
var workerUDTmpl *template.Template
var workloadTmpl *template.Template

type masterConfig struct {
	baseConfig
	ExternalIP  string
	PhoneHomeIP string
}

type workerConfig struct {
	baseConfig
	MasterIP string
}

type creator struct {
	opts            *options
	pk              string
	token           string
	pc              *proxyConfig
	mc              *masterConfig
	wc              *workerConfig
	cwd             string
	masterPath      string
	masterUDPath    string
	masterWkldUUID  string
	masterInstUUID  string
	masterIP        string
	workerPath      string
	workerUDPath    string
	workerWkldUUID  string
	workerInstUUIDs []string
	poolName        string
	poolCreated     string
	externalIP      string
	listener        net.Listener
	errCh           chan error
	adminPath       string
}

func init() {
	masterUDTmpl = template.Must(template.New("masterUD").Parse(
		udCommonTemplate + udMasterTemplate))
	workerUDTmpl = template.Must(template.New("workerUD").Parse(
		udCommonTemplate + udNodeTemplate))
	workloadTmpl = template.Must(template.New("workloadTmpl").Parse(workloadTemplate))
}

// TODO: Code copied from ciao-down needs to be refactored
func getProxy(upper, lower string) (string, error) {
	proxy := os.Getenv(upper)
	if proxy == "" {
		proxy = os.Getenv(lower)
	}

	if proxy == "" {
		return "", nil
	}

	if proxy[len(proxy)-1] == '/' {
		proxy = proxy[:len(proxy)-1]
	}

	proxyURL, err := url.Parse(proxy)
	if err != nil {
		return "", fmt.Errorf("Failed to parse %s : %v", proxy, err)
	}
	return proxyURL.String(), nil
}

func getProxyConfig() (*proxyConfig, error) {
	var err error
	pc := &proxyConfig{}
	pc.httpProxy, err = getProxy("http_proxy", "HTTP_PROXY")
	if err != nil {
		return nil, err
	}
	pc.httpsProxy, err = getProxy("https_proxy", "HTTPS_PROXY")
	if err != nil {
		return nil, err
	}
	pc.noProxy = os.Getenv("no_proxy")
	return pc, nil
}

// TODO: Code copied from networking

func validPhysicalLink(link netlink.Link) bool {
	phyDevice := true

	switch link.Type() {
	case "device":
	case "bond":
	case "vlan":
	case "macvlan":
	case "bridge":
		if strings.HasPrefix(link.Attrs().Name, "docker") ||
			strings.HasPrefix(link.Attrs().Name, "virbr") {
			phyDevice = false
		}
	default:
		phyDevice = false
	}

	if (link.Attrs().Flags & net.FlagLoopback) != 0 {
		return false
	}

	return phyDevice
}

func getFirstPhyDevice() string {
	links, err := netlink.LinkList()
	if err != nil {
		return ""
	}

	for _, link := range links {
		if !validPhysicalLink(link) {
			continue
		}

		addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil || len(addrs) == 0 {
			continue
		}

		return addrs[0].IP.String()
	}

	return ""
}

func genToken() (string, error) {
	var buf [11]byte
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s",
		hex.EncodeToString(buf[:3]), hex.EncodeToString(buf[3:])), nil
}

func (c *creator) createMasterConfig() {
	mc := &masterConfig{
		baseConfig: baseConfig{
			VCPUs:        c.opts.masterVM.vCPUs,
			RAMMiB:       c.opts.masterVM.memMiB,
			DiskGiB:      c.opts.masterVM.diskGiB,
			User:         c.opts.user,
			ImageUUID:    c.opts.imageUUID,
			PublicKey:    c.pk,
			Token:        c.token,
			UserDataFile: "k8s-master-ci.yaml",
			Description:  masterWorkloadName,
		},
		ExternalIP: c.opts.externalIP,
	}

	if mc.ExternalIP != "" {
		mc.PhoneHomeIP = getFirstPhyDevice()
	}

	mc.HTTPSProxy = c.pc.httpsProxy
	mc.HTTPProxy = c.pc.httpProxy
	mc.NoProxy = c.pc.noProxy
	c.mc = mc
}

func (c *creator) createWorkerConfig() {
	wc := &workerConfig{
		baseConfig: baseConfig{
			VCPUs:        c.opts.workerVM.vCPUs,
			RAMMiB:       c.opts.workerVM.memMiB,
			DiskGiB:      c.opts.workerVM.diskGiB,
			User:         c.opts.user,
			ImageUUID:    c.opts.imageUUID,
			PublicKey:    c.pk,
			Token:        c.token,
			UserDataFile: "k8s-worker-ci.yaml",
			Description:  workerWorkloadName,
		},
		MasterIP: c.masterIP,
	}

	wc.HTTPSProxy = c.pc.httpsProxy
	wc.HTTPProxy = c.pc.httpProxy
	wc.NoProxy = c.pc.noProxy
	c.wc = wc
}

func (c *creator) createMasterWorkloadDefinition() error {
	var buf bytes.Buffer

	err := workloadTmpl.Execute(&buf, c.mc)
	if err != nil {
		return err
	}

	// TODO might want to check to see if these files already
	// exist and error if a -f is not provided.

	c.masterPath = filepath.Join(c.cwd, "k8s-master.yaml")
	err = ioutil.WriteFile(c.masterPath, buf.Bytes(), 0600)
	if err != nil {
		return err
	}

	buf.Reset()

	err = masterUDTmpl.Execute(&buf, c.mc)
	if err != nil {
		_ = os.Remove(c.masterPath)
		return err
	}

	c.masterUDPath = filepath.Join(c.cwd, c.mc.UserDataFile)

	err = ioutil.WriteFile(c.masterUDPath, buf.Bytes(), 0600)
	if err != nil {
		_ = os.Remove(c.masterPath)
		return err
	}

	return nil
}

func (c *creator) createWorkerWorkloadDefinition() error {
	var buf bytes.Buffer

	err := workloadTmpl.Execute(&buf, c.wc)
	if err != nil {
		return err
	}

	// TODO might want to check to see if these files already
	// exist and error if a -f is not provided.

	c.workerPath = filepath.Join(c.cwd, "k8s-worker.yaml")
	err = ioutil.WriteFile(c.workerPath, buf.Bytes(), 0600)
	if err != nil {
		return err
	}

	buf.Reset()

	err = workerUDTmpl.Execute(&buf, c.wc)
	if err != nil {
		_ = os.Remove(c.workerPath)
		return err
	}

	c.workerUDPath = filepath.Join(c.cwd, c.wc.UserDataFile)

	err = ioutil.WriteFile(c.workerUDPath, buf.Bytes(), 0600)
	if err != nil {
		_ = os.Remove(c.workerPath)
		return err
	}

	return nil
}

func (c *creator) createMaster(ctx context.Context) error {
	c.createMasterConfig()
	c.startServer()
	err := c.createMasterWorkloadDefinition()
	if err != nil {
		return err
	}
	defer func() {
		if !c.opts.keep {
			_ = os.Remove(c.masterPath)
			_ = os.Remove(c.masterUDPath)
		}
	}()

	id, err := bat.CreateWorkloadFromFile(ctx, false, "", c.masterPath)
	if err != nil {
		return fmt.Errorf("Failed to create master workload: %s", c.masterPath)
	}
	c.masterWkldUUID = id
	if c.mc.ExternalIP != "" {
		c.poolName = fmt.Sprintf("%s-%s", externalIPPool, id)
	}

	ids, err := bat.LaunchInstances(ctx, "", c.masterWkldUUID, 1)
	if err != nil {
		return fmt.Errorf("Failed to launch master instance")
	}
	c.masterInstUUID = ids[0]

	_, err = bat.WaitForInstancesLaunch(ctx, "", ids, true)
	if err != nil {
		return fmt.Errorf("Master instance failed to launch")
	}

	instance, err := bat.GetInstance(ctx, "", c.masterInstUUID)
	if err != nil {
		return fmt.Errorf("Unable to retrieve information about master instance")
	}
	c.masterIP = instance.PrivateIP

	return err
}

func (c *creator) createWorkers(ctx context.Context) error {
	c.createWorkerConfig()
	err := c.createWorkerWorkloadDefinition()
	if err != nil {
		return err
	}
	defer func() {
		if !c.opts.keep {
			_ = os.Remove(c.workerPath)
			_ = os.Remove(c.workerUDPath)
		}
	}()

	id, err := bat.CreateWorkloadFromFile(ctx, false, "", c.workerPath)
	if err != nil {
		return err
	}
	c.workerWkldUUID = id

	ids, err := bat.LaunchInstances(ctx, "", c.workerWkldUUID, c.opts.workers)
	if err != nil {
		return err
	}
	c.workerInstUUIDs = ids

	_, err = bat.WaitForInstancesLaunch(ctx, "", c.workerInstUUIDs, true)
	if err != nil {
		return err
	}
	return err
}

func (c *creator) createExternalIP(ctx context.Context) error {
	err := bat.CreateExternalIPPool(ctx, "", c.poolName)
	if err != nil {
		return fmt.Errorf("Unable to create external-ip pool %s",
			c.poolName)
	}

	c.poolCreated = c.poolName

	err = bat.AddExternalIPToPool(ctx, "", c.poolCreated,
		c.mc.ExternalIP)
	if err != nil {
		return fmt.Errorf("Unable to add external-ip %s to pool %s",
			c.mc.ExternalIP, c.poolName)
	}

	err = bat.MapExternalIP(ctx, "", c.poolCreated, c.masterInstUUID)
	if err != nil {
		return fmt.Errorf("Unable to map external-ip %s to instance %s",
			c.mc.ExternalIP, c.masterInstUUID)
	}
	c.externalIP = c.mc.ExternalIP

	return nil
}

func (c *creator) status() {
	fmt.Println("\nk8s cluster successfully created")
	fmt.Println("--------------------------------")
	if c.opts.keep {
		fmt.Println("Created workload definition files:")
		fmt.Println(c.masterPath)
		fmt.Println(c.masterUDPath)
		fmt.Println(c.workerPath)
		fmt.Println(c.workerUDPath)
	}
	fmt.Println("Created master:")
	fmt.Printf(" - %s\n", c.masterInstUUID)
	fmt.Printf("Created %d workers:\n", c.opts.workers)
	for _, w := range c.workerInstUUIDs {
		fmt.Printf(" - %s\n", w)
	}

	if c.externalIP != "" {
		fmt.Println("Created external-ips:")
		fmt.Printf("- %s\n", c.externalIP)
		fmt.Println("Created pools:")
		fmt.Printf("- %s\n", c.poolCreated)
	}

	if c.adminPath != "" {
		fmt.Println("To access k8s cluster:")
		fmt.Printf("- export KUBECONFIG=%s\n", c.adminPath)
		fmt.Println("- If you use proxies, set")
		fmt.Printf("  - export no_proxy=$no_proxy,%s\n", c.mc.ExternalIP)
	}
}

func deleteInstance(ctx context.Context, i string) error {
	var err error
	for tries := 0; tries < 10; tries++ {
		tctx, cancel := context.WithTimeout(ctx, time.Second*15)
		err = bat.DeleteInstanceAndWait(tctx, "", i)
		cancel()
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			break
		case <-time.After(time.Second):
		}
	}

	return err
}

func (c *creator) cleanup() {
	fmt.Fprintln(os.Stderr, "Error Detected")

	if c.listener != nil {
		_ = c.listener.Close()
	}

	if c.externalIP != "" {
		fmt.Fprintf(os.Stderr, "Unmapping external-ip: %s\n", c.externalIP)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = bat.UnmapExternalIP(ctx, "", c.externalIP)
		cancel()
	}

	if c.poolCreated != "" {
		fmt.Fprintf(os.Stderr, "Deleting external-ip pool: %s\n", c.poolCreated)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = bat.DeleteExternalIPPool(ctx, "", c.poolCreated)
		cancel()
	}

	// TODO: This is slow
	for _, i := range c.workerInstUUIDs {
		fmt.Fprintf(os.Stderr, "Deleting worker instance: %s\n", i)
		_ = deleteInstance(context.Background(), i)
	}

	if c.workerWkldUUID != "" {
		fmt.Fprintf(os.Stderr, "Deleting worker workload: %s\n", c.workerWkldUUID)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = bat.DeleteWorkload(ctx, "", c.workerWkldUUID)
		cancel()
	}

	if c.masterInstUUID != "" {
		fmt.Fprintf(os.Stderr, "Deleting master instance: %s\n", c.masterInstUUID)
		_ = deleteInstance(context.Background(), c.masterInstUUID)
	}

	if c.masterWkldUUID != "" {
		fmt.Fprintf(os.Stderr, "Deleting master workload: %s\n", c.masterWkldUUID)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = bat.DeleteWorkload(ctx, "", c.masterWkldUUID)
		cancel()
	}
}

func startHTTPServer(adminPath string, listener net.Listener, errCh chan error) {
	finished := false
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		d, err := ioutil.ReadAll(r.Body)
		defer func() {
			_ = listener.Close()
		}()

		if err != nil {
			return
		}
		err = ioutil.WriteFile(adminPath, d, 0600)
		if err != nil {
			return
		}

		finished = true
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

func (c *creator) startServer() {
	if c.mc.PhoneHomeIP == "" {
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", c.mc.PhoneHomeIP, 9000))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Unable to create listener for HTTP server: %v", err)
		return
	}
	c.listener = listener
	c.errCh = make(chan error)
	c.adminPath = filepath.Join(c.cwd, "admin.conf")
	startHTTPServer(c.adminPath, listener, c.errCh)
}

func (c *creator) waitForAdminConf(ctx context.Context) error {
	var err error
	if c.listener == nil {
		return err
	}

	fmt.Println("Instances launched.  Waiting for k8s cluster to start")
	select {
	case <-ctx.Done():
		_ = c.listener.Close()
		<-c.errCh
		err = ctx.Err()
	case err = <-c.errCh:
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", err)
		}
	}

	return err
}

func (c *creator) alreadyExists(ctx context.Context) bool {
	_, err := getWorkloadUUIDs(ctx, workerWorkloadName)
	if err == nil {
		return true
	}

	_, err = getWorkloadUUIDs(ctx, masterWorkloadName)
	if err == nil {
		return true
	}

	return false
}

func newCreator() (*creator, error) {
	c := &creator{}

	var err error
	c.opts, err = createFlags()
	if err != nil {
		return nil, err
	}

	pk, err := ioutil.ReadFile(c.opts.publicKeyPath)
	if err != nil {
		return nil, err
	}
	c.pk = string(pk)

	c.token, err = genToken()
	if err != nil {
		return nil, err
	}

	c.pc, err = getProxyConfig()
	if err != nil {
		return nil, err
	}

	c.cwd, err = os.Getwd()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func createCluster(ctx context.Context) (err error) {
	var c *creator
	c, err = newCreator()
	if err != nil {
		return err
	}

	if c.alreadyExists(ctx) {
		return fmt.Errorf("kubicle has already created a cluster for this tenant")
	}

	defer func() {
		if err != nil {
			c.cleanup()
		}
	}()

	fmt.Println("Creating master")
	err = c.createMaster(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Creating workers")
	err = c.createWorkers(ctx)
	if err != nil {
		return err
	}

	if c.mc.ExternalIP != "" {
		fmt.Println("Mapping external-ip")
		err = c.createExternalIP(ctx)
		if err != nil {
			return
		}
	}

	err = c.waitForAdminConf(ctx)
	if err != nil {
		return err
	}

	c.status()

	return nil
}

func create(ctx context.Context, errCh chan error) {
	errCh <- createCluster(ctx)
}
