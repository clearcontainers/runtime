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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
)

var caServerPath string
var serverPath string
var serverURL string
var traceCommands bool
var computeNet string
var mgmtNet string
var diskLimit bool
var memLimit bool
var cephID string

var ssntpServer = &ssntp.Server{}

func init() {
	flag.StringVar(&caServerPath, "ca-server-cert", "", "Path to CAServer certificate")
	flag.StringVar(&serverPath, "server-cert", "", "Path to server certificate")
	flag.BoolVar(&traceCommands, "trace", false, "Turn on ssntp command tracing")
	flag.StringVar(&serverURL, "server", "127.0.0.1:9000", "IP port of server")
	flag.StringVar(&computeNet, "compute-net", "", "Compute Subnet")
	flag.StringVar(&mgmtNet, "mgmt-net", "", "Management Subnet")
	flag.BoolVar(&diskLimit, "disk-limit", true, "Use disk usage limits")
	flag.BoolVar(&memLimit, "mem-limit", true, "Use memory usage limits")
	flag.StringVar(&cephID, "ceph-id", "ciao", "ceph client id")
}

type client struct {
	stats  *payloads.Stat
	status *payloads.Ready
	events []interface{}
}

var server = struct {
	sync.Mutex // Protects Map
	clients    map[string]*client
}{
	clients: make(map[string]*client),
}

type testServer struct{}

func (ts *testServer) ConnectNotify(uuid string, role ssntp.Role) {
	server.Lock()
	defer server.Unlock()

	if !(role.IsAgent() || role.IsNetAgent()) {
		fmt.Fprintf(os.Stderr, "Ignoring connection from unexpected role: %s.\n", role.String())
		return
	}

	if _, exists := server.clients[uuid]; exists {
		return
	}

	server.clients[uuid] = new(client)
}

func (ts *testServer) DisconnectNotify(uuid string, role ssntp.Role) {
	server.Lock()
	defer server.Unlock()

	if _, exists := server.clients[uuid]; exists {
		delete(server.clients, uuid)
	}
}

func (ts *testServer) StatusNotify(uuid string, status ssntp.Status, frame *ssntp.Frame) {
	var ready payloads.Ready
	err := yaml.Unmarshal(frame.Payload, &ready)
	if err == nil {
		server.Lock()
		if server.clients[uuid] != nil {
			server.clients[uuid].status = &ready
		}
		server.Unlock()
	}
}

func (ts *testServer) CommandNotify(uuid string, command ssntp.Command, frame *ssntp.Frame) {
	switch command {
	case ssntp.STATS:
		var stats payloads.Stat
		err := yaml.Unmarshal(frame.Payload, &stats)
		if err == nil {
			server.Lock()
			if server.clients[uuid] != nil {
				server.clients[uuid].stats = &stats
			}
			server.Unlock()
		}
	}
}

func (ts *testServer) ErrorNotify(uuid string, err ssntp.Error, frame *ssntp.Frame) {
	server.Lock()
	defer server.Unlock()

	c := server.clients[uuid]
	if c == nil {
		return
	}

	if c.events == nil {
		c.events = make([]interface{}, 0, 32)
	}

	// TODO is there a better way to do this with reflection?

	var e interface{}
	switch err {
	case ssntp.StartFailure:
		payload := payloads.ErrorStartFailure{}
		err := yaml.Unmarshal(frame.Payload, &payload)
		if err == nil {
			e = &payload
		}
	case ssntp.DeleteFailure:
		payload := payloads.ErrorDeleteFailure{}
		err := yaml.Unmarshal(frame.Payload, &payload)
		if err == nil {
			e = &payload
		}
	case ssntp.AttachVolumeFailure:
		payload := payloads.ErrorAttachVolumeFailure{}
		err := yaml.Unmarshal(frame.Payload, &payload)
		if err == nil {
			e = &payload
		}
	case ssntp.DetachVolumeFailure:
		payload := payloads.ErrorDetachVolumeFailure{}
		err := yaml.Unmarshal(frame.Payload, &payload)
		if err == nil {
			e = &payload
		}
	}

	c.events = append(c.events, e)
}

func (ts *testServer) EventNotify(uuid string, event ssntp.Event, frame *ssntp.Frame) {
	server.Lock()
	defer server.Unlock()

	c := server.clients[uuid]
	if c == nil {
		return
	}

	if c.events == nil {
		c.events = make([]interface{}, 0, 32)
	}

	var e interface{}

	switch event {
	case ssntp.TenantAdded:
		payload := payloads.EventTenantAdded{}
		err := yaml.Unmarshal(frame.Payload, &payload)
		if err == nil {
			e = &payload
		}
	case ssntp.TenantRemoved:
		payload := payloads.EventTenantRemoved{}
		err := yaml.Unmarshal(frame.Payload, &payload)
		if err == nil {
			e = &payload
		}
	case ssntp.InstanceDeleted:
		payload := payloads.EventInstanceDeleted{}
		err := yaml.Unmarshal(frame.Payload, &payload)
		if err == nil {
			e = &payload
		}
	case ssntp.InstanceStopped:
		payload := payloads.EventInstanceStopped{}
		err := yaml.Unmarshal(frame.Payload, &payload)
		if err == nil {
			e = &payload
		}
	}

	c.events = append(c.events, e)
}

func getCertPaths(tmpDir string) (string, string) {

	var caPath, sPath string

	caPath = path.Join(tmpDir, "CACertServer")
	sPath = path.Join(tmpDir, "CertServer")

	for _, s := range []struct{ path, data string }{{caPath, caCertServer}, {sPath, certServer}} {
		err := ioutil.WriteFile(s.path, []byte(s.data), 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create certfile %s %v\n", s.path, err)
			os.Exit(1)
		}
	}

	return caPath, sPath
}

func dumpYaml(w http.ResponseWriter, data interface{}) {
	payload, err := yaml.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(payload)
}

func clients(w http.ResponseWriter, r *http.Request) {
	if r.Method != "" && r.Method != "GET" {
		http.Error(w, "GET expected", http.StatusBadRequest)
		return
	}
	server.Lock()
	defer server.Unlock()
	i := 0
	clients := make([]string, len(server.clients))
	for k := range server.clients {
		clients[i] = k
		i++
	}

	dumpYaml(w, &clients)
}

func getClient(clientP string) string {
	if len(server.clients) == 0 {
		return ""
	} else if len(server.clients) == 1 && clientP == "" {
		for k := range server.clients {
			clientP = k
		}
	} else {
		if clientP == "" {
			return ""
		}
	}

	return clientP
}

func yamlCommand(w http.ResponseWriter, r *http.Request, command ssntp.Command) {
	if r.Method != "POST" || r.Body == nil {
		http.Error(w, "POST expected", http.StatusBadRequest)
		return
	}

	values := r.URL.Query()
	clientP := values.Get("client")
	server.Lock()
	defer server.Unlock()

	clientP = getClient(clientP)
	c := server.clients[clientP]
	if c == nil {
		http.Error(w, "Invalid client", http.StatusBadRequest)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = ssntpServer.SendCommand(clientP, command, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func instances(w http.ResponseWriter, r *http.Request) {
	if r.Method != "" && r.Method != "GET" {
		http.Error(w, "GET expected", http.StatusBadRequest)
		return
	}

	values := r.URL.Query()
	clientP := values.Get("client")
	filterP := values.Get("filter")
	server.Lock()
	defer server.Unlock()

	clientP = getClient(clientP)
	c := server.clients[clientP]
	if c == nil {
		http.Error(w, "Invalid client", http.StatusBadRequest)
		return
	}

	var instances []string
	if c.stats != nil {
		instances = make([]string, 0, len(c.stats.Instances))
		for _, i := range c.stats.Instances {
			if filterP == "" || (filterP == i.State) {
				instances = append(instances, i.InstanceUUID)
			}
		}
	} else {
		instances = []string{}
	}

	dumpYaml(w, instances)
}

func stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "" && r.Method != "GET" {
		http.Error(w, "GET expected", http.StatusBadRequest)
		return
	}

	values := r.URL.Query()
	clientP := values.Get("client")
	filterP := values.Get("filter")
	server.Lock()
	defer server.Unlock()

	clientP = getClient(clientP)
	c := server.clients[clientP]
	if c == nil {
		http.Error(w, "Invalid client", http.StatusBadRequest)
		return
	}

	if c.stats == nil {
		http.Error(w, "Stats not available", http.StatusNotFound)
		return
	}

	var stats *payloads.Stat

	if filterP == "" {
		stats = c.stats
	} else {
		tmpStats := *c.stats
		counter := 0
		for _, i := range c.stats.Instances {
			if filterP == "" || (filterP == i.State) {
				tmpStats.Instances[counter] = i
				counter++
			}
		}
		tmpStats.Instances = tmpStats.Instances[:counter]
		stats = &tmpStats
	}

	dumpYaml(w, stats)
}

func status(w http.ResponseWriter, r *http.Request) {
	if r.Method != "" && r.Method != "GET" {
		http.Error(w, "GET expected", http.StatusBadRequest)
		return
	}

	values := r.URL.Query()
	clientP := values.Get("client")
	server.Lock()
	defer server.Unlock()

	clientP = getClient(clientP)
	c := server.clients[clientP]
	if c == nil {
		http.Error(w, "Invalid client", http.StatusBadRequest)
		return
	}

	if c.status == nil {
		http.Error(w, "Statuss not available", http.StatusNotFound)
		return
	}

	dumpYaml(w, c.status)
}

func drain(w http.ResponseWriter, r *http.Request) {
	if r.Method != "" && r.Method != "GET" {
		http.Error(w, "GET expected", http.StatusBadRequest)
		return
	}

	values := r.URL.Query()
	clientP := values.Get("client")
	server.Lock()
	defer server.Unlock()

	clientP = getClient(clientP)
	c := server.clients[clientP]
	if c == nil {
		http.Error(w, "Invalid client", http.StatusBadRequest)
		return
	}

	dumpYaml(w, c.events)
	c.events = nil
}

func serve(done chan os.Signal) {
	listener, err := net.Listen("tcp", serverURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create listener: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		http.HandleFunc("/clients", clients)
		http.HandleFunc("/instances", instances)
		http.HandleFunc("/start",
			func(w http.ResponseWriter, r *http.Request) {
				yamlCommand(w, r, ssntp.START)
			})
		http.HandleFunc("/delete",
			func(w http.ResponseWriter, r *http.Request) {
				yamlCommand(w, r, ssntp.DELETE)
			})
		http.HandleFunc("/attach",
			func(w http.ResponseWriter, r *http.Request) {
				yamlCommand(w, r, ssntp.AttachVolume)
			})
		http.HandleFunc("/detach",
			func(w http.ResponseWriter, r *http.Request) {
				yamlCommand(w, r, ssntp.DetachVolume)
			})
		http.HandleFunc("/stats", stats)
		http.HandleFunc("/status", status)
		http.HandleFunc("/drain", drain)
		if err := http.Serve(listener, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to listen: %v\n", err)
		}
		wg.Done()
	}()
	<-done
	listener.Close()
	wg.Wait()
}

func createConfigFile(confPath string) error {
	conf := payloads.Configure{}
	conf.InitDefaults()

	conf.Configure.Scheduler.ConfigStorageURI = "file://" + confPath
	conf.Configure.Controller.HTTPSCACert = "n/a"
	conf.Configure.Controller.HTTPSKey = "n/a"
	conf.Configure.Controller.IdentityUser = "n/a"
	conf.Configure.Controller.IdentityPassword = "n/a"
	conf.Configure.IdentityService.URL = "http://127.0.0.1"

	conf.Configure.Launcher.DiskLimit = diskLimit
	conf.Configure.Launcher.MemoryLimit = memLimit
	if computeNet != "" {
		conf.Configure.Launcher.ComputeNetwork = []string{computeNet}
	}
	if mgmtNet != "" {
		conf.Configure.Launcher.ManagementNetwork = []string{mgmtNet}
	}
	conf.Configure.Storage.CephID = cephID

	d, err := yaml.Marshal(&conf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to marshall configuration data: %v", err)
		return err
	}

	return ioutil.WriteFile(confPath, d, 0755)
}

func main() {
	flag.Parse()

	cfg := new(ssntp.Config)
	confDir, err := ioutil.TempDir("", "ciao-server-launcher")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to create temporary conf directory.")
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(confDir) }()

	confPath := path.Join(confDir, "conf.yaml")
	if err = createConfigFile(confPath); err != nil {
		fmt.Fprintln(os.Stderr, "Unable to create conf file.")
		os.Exit(1)
	}

	cfg.ConfigURI = "file://" + confPath

	if (caServerPath == "" && serverPath != "") || (caServerPath != "" && serverPath == "") {
		fmt.Fprintln(os.Stderr, "Either both or neither certificate paths must be defined")
		os.Exit(1)
	} else if caServerPath == "" && serverPath == "" {
		tmpDir, err := ioutil.TempDir("", "launcher-server")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create temporary Dir %v\n", err)
			os.Exit(1)
		}
		defer func() {
			_ = os.RemoveAll(tmpDir)
		}()
		cfg.CAcert, cfg.Cert = getCertPaths(tmpDir)
	} else {
		cfg.CAcert, cfg.Cert = caServerPath, serverPath
	}
	cfg.Trace = &ssntp.TraceConfig{PathTrace: traceCommands, Start: time.Now()}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		err := ssntpServer.Serve(cfg, &testServer{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to start server %v\n", err)
			os.Exit(1)
		}
		wg.Done()
	}()

	serve(signalCh)
	ssntpServer.Stop()
	wg.Wait()
}
