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
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"text/tabwriter"

	"github.com/01org/ciao/payloads"
	"gopkg.in/yaml.v2"
)

var serverURL string

func init() {
	flag.StringVar(&serverURL, "server", "127.0.0.1:9000", "IP port of server")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "ciaolc is a command line tool for testing ciao-launcher")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "\tciaolc command")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Where commands are:")
		fmt.Fprintln(os.Stderr, "\tstartf")
		fmt.Fprintln(os.Stderr, "\tdelete")
		fmt.Fprintln(os.Stderr, "\tstop")
		fmt.Fprintln(os.Stderr, "\trestart")
		fmt.Fprintln(os.Stderr, "\tdrain")
		fmt.Fprintln(os.Stderr, "\tstats")
		fmt.Fprintln(os.Stderr, "\tistats")
		fmt.Fprintln(os.Stderr, "\tstatus")
		fmt.Fprintln(os.Stderr, "\tattach")
		fmt.Fprintln(os.Stderr, "\tdetach")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
	}
}

type filter string

func (f *filter) String() string {
	return string(*f)
}

func (f *filter) Set(val string) error {
	if val != "none" &&
		val != payloads.ComputeStatusStopped &&
		val != payloads.ComputeStatusPending &&
		val != payloads.ComputeStatusRunning {
		return fmt.Errorf("exited, pending, active expected")
	}
	*f = filter(val)

	return nil
}

func grabBody(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func clients(host string) error {
	u := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   "clients",
	}

	body, err := grabBody(u.String())
	if err != nil {
		return err
	}
	clients := []string{}
	err = yaml.Unmarshal(body, &clients)
	if err != nil {
		return err
	}

	for _, c := range clients {
		fmt.Println(c)
	}

	return nil
}

func instances(host string) error {
	u, err := queryStatsURL("instances", "instances", host)
	if err != nil {
		return err
	}
	body, err := grabBody(u)
	if err != nil {
		return err
	}
	clients := []string{}
	err = yaml.Unmarshal(body, &clients)
	if err != nil {
		return err
	}

	for _, c := range clients {
		fmt.Println(c)
	}

	return nil
}

func queryURL(host string, cmd string, c string, f string) string {
	u := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   cmd,
	}

	q := u.Query()
	if c != "" {
		q.Set("client", c)
	}
	if f != "" {
		q.Set("filter", f)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func getStats(u string) (*payloads.Stat, error) {
	body, err := grabBody(u)
	if err != nil {
		return nil, err
	}

	var stats payloads.Stat
	err = yaml.Unmarshal(body, &stats)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func queryStatsURL(cmd, remoteCmd, host string) (string, error) {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	fp := filter("none")
	fs.Var(&fp, "filter", "Can be exited, pending or running")
	cp := ""
	fs.StringVar(&cp, "client", "", "UUID of client")

	if err := fs.Parse(flag.Args()[1:]); err != nil {
		return "", err
	}

	f := ""
	if fp != filter("none") {
		f = fp.String()
	}

	return queryURL(host, remoteCmd, cp, f), nil
}

func istats(host string) error {
	u, err := queryStatsURL("istats", "stats", host)
	if err != nil {
		return err
	}
	stats, err := getStats(u)
	if err != nil {
		return err
	}

	if len(stats.Instances) == 0 {
		return nil
	}

	w := new(tabwriter.Writer)

	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintln(w, "UUID\tStatus\tSSH\tMem\tDisk\tCPU\tVolumes")
	for _, i := range stats.Instances {
		fmt.Fprintf(w, "%s\t%s\t%s:%d\t%d MB\t%d MB\t%d%%\t%s\n",
			i.InstanceUUID,
			i.State,
			i.SSHIP, i.SSHPort,
			i.MemoryUsageMB,
			i.DiskUsageMB,
			i.CPUUsage,
			i.Volumes)
	}
	w.Flush()

	return nil
}

func stats(host string) error {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	cp := ""
	fs.StringVar(&cp, "client", "", "UUID of client")

	if err := fs.Parse(flag.Args()[1:]); err != nil {
		return err
	}

	u := queryURL(host, "stats", cp, "")
	stats, err := getStats(u)
	if err != nil {
		return err
	}

	if len(stats.Instances) == 0 {
		return nil
	}

	w := new(tabwriter.Writer)

	running := 0
	pending := 0
	exited := 0
	unknown := 0

	for i := range stats.Instances {
		switch stats.Instances[i].State {
		case payloads.Pending:
			pending++
		case payloads.Running:
			running++
		case payloads.Exited:
			exited++
		default:
			unknown++
		}
	}

	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintf(w, "NodeUUID:\t %s\n", stats.NodeUUID)
	fmt.Fprintf(w, "Status:\t %s\n", stats.Status)
	fmt.Fprintf(w, "MemTotal:\t %d MB\n", stats.MemTotalMB)
	fmt.Fprintf(w, "MemAvailable:\t %d MB\n", stats.MemAvailableMB)
	fmt.Fprintf(w, "DiskTotal:\t %d MB\n", stats.DiskTotalMB)
	fmt.Fprintf(w, "DiskAvailable:\t %d MB\n", stats.DiskAvailableMB)
	fmt.Fprintf(w, "Load:\t %d\n", stats.Load)
	fmt.Fprintf(w, "CpusOnline:\t %d\n", stats.CpusOnline)
	fmt.Fprintf(w, "NodeHostName:\t %s\n", stats.NodeHostName)
	if len(stats.Networks) == 1 {
		fmt.Fprintf(w, "NodeIP:\t %s\n", stats.Networks[0].NodeIP)
		fmt.Fprintf(w, "NodeMAC:\t %s\n", stats.Networks[0].NodeMAC)
	}
	for i, n := range stats.Networks {
		fmt.Fprintf(w, "NodeIP-%d:\t %s\n", i+1, n.NodeIP)
		fmt.Fprintf(w, "NodeMAC-%d:\t %s\n", i+1, n.NodeMAC)
	}
	if unknown == 0 {
		fmt.Fprintf(w, "Instances:\t %d (%d running %d exited %d pending)\n",
			len(stats.Instances), running, exited, pending)
	} else {
		fmt.Fprintf(w, "Instances:\t %d (%d running %d exited %d pending %d other)\n",
			len(stats.Instances), running, exited, pending, unknown)
	}
	w.Flush()

	return nil
}

func status(host string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	cp := ""
	fs.StringVar(&cp, "client", "", "UUID of client")

	if err := fs.Parse(flag.Args()[1:]); err != nil {
		return err
	}
	u := queryURL(host, "status", cp, "")

	body, err := grabBody(u)
	if err != nil {
		return err
	}

	var status payloads.Ready
	err = yaml.Unmarshal(body, &status)
	if err != nil {
		return err
	}

	w := new(tabwriter.Writer)

	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintf(w, "NodeUUID:\t %s\n", status.NodeUUID)
	fmt.Fprintf(w, "MemTotal:\t %d MB\n", status.MemTotalMB)
	fmt.Fprintf(w, "MemAvailable:\t %d MB\n", status.MemAvailableMB)
	fmt.Fprintf(w, "DiskTotal:\t %d MB\n", status.DiskTotalMB)
	fmt.Fprintf(w, "DiskAvailable:\t %d MB\n", status.DiskAvailableMB)
	fmt.Fprintf(w, "Load:\t %d\n", status.Load)
	fmt.Fprintf(w, "CpusOnline:\t %d\n", status.CpusOnline)
	w.Flush()

	return nil
}

func drain(host string) error {
	fs := flag.NewFlagSet("drain", flag.ExitOnError)
	cp := ""
	fs.StringVar(&cp, "client", "", "UUID of client")

	if err := fs.Parse(flag.Args()[1:]); err != nil {
		return err
	}
	u := queryURL(host, "drain", cp, "")

	body, err := grabBody(u)
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	return nil
}

func getSimplePostArgs(cmd string) (string, string, error) {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	cp := ""
	fs.StringVar(&cp, "client", "", "UUID of client")

	if err := fs.Parse(flag.Args()[1:]); err != nil {
		return "", "", err
	}

	instance := fs.Arg(0)
	if instance == "" {
		return "", "", fmt.Errorf("Missing instance-uuid")
	}

	return cp, instance, nil
}

func getVolumePostArgs(cmd string) (string, string, string, error) {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	cp := ""
	fs.StringVar(&cp, "client", "", "UUID of client")

	if err := fs.Parse(flag.Args()[1:]); err != nil {
		return "", "", "", err
	}

	instance := fs.Arg(0)
	if instance == "" {
		return "", "", "", fmt.Errorf("Missing instance-uuid")
	}

	volume := fs.Arg(1)
	if volume == "" {
		return "", "", "", fmt.Errorf("Missing volume-uuid")
	}

	return cp, instance, volume, nil
}

func postYaml(host, cmd, client string, data interface{}) error {
	u := queryURL(host, cmd, client, "")
	payload, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	resp, err := http.Post(u, "text/yaml", bytes.NewBuffer(payload))
	resp.Body.Close()
	return err
}

func startf(host string) error {
	client, path, err := getSimplePostArgs("start")
	if err != nil {
		return err
	}

	payload, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	u := queryURL(host, "start", client, "")
	resp, err := http.Post(u, "text/yaml", bytes.NewBuffer(payload))
	resp.Body.Close()
	return err
}

func stop(host string) error {
	var stop payloads.Stop

	client, instance, err := getSimplePostArgs("stop")
	if err != nil {
		return err
	}

	stop.Stop.InstanceUUID = instance
	return postYaml(host, "stop", client, &stop)
}

func restart(host string) error {
	var restart payloads.Restart

	client, instance, err := getSimplePostArgs("restart")
	if err != nil {
		return err
	}

	restart.Restart.InstanceUUID = instance
	return postYaml(host, "restart", client, &restart)
}

func del(host string) error {
	var del payloads.Delete

	client, instance, err := getSimplePostArgs("delete")
	if err != nil {
		return err
	}

	del.Delete.InstanceUUID = instance
	return postYaml(host, "delete", client, &del)
}

func attach(host string) error {
	var attach payloads.AttachVolume
	client, instance, volume, err := getVolumePostArgs("attach")
	if err != nil {
		return err
	}

	attach.Attach.InstanceUUID = instance
	attach.Attach.VolumeUUID = volume
	return postYaml(host, "attach", client, &attach)
}

func detach(host string) error {
	var detach payloads.DetachVolume
	client, instance, volume, err := getVolumePostArgs("detach")
	if err != nil {
		return err
	}

	detach.Detach.InstanceUUID = instance
	detach.Detach.VolumeUUID = volume
	return postYaml(host, "detach", client, &detach)
}

func main() {

	flag.Parse()
	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	cmdMap := map[string]func(string) error{
		"clients":   clients,
		"instances": instances,
		"istats":    istats,
		"stats":     stats,
		"status":    status,
		"stop":      stop,
		"restart":   restart,
		"delete":    del,
		"drain":     drain,
		"startf":    startf,
		"attach":    attach,
		"detach":    detach,
	}

	cmd := cmdMap[os.Args[1]]
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", os.Args[1])
		os.Exit(1)
	}

	if err := cmd(serverURL); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
