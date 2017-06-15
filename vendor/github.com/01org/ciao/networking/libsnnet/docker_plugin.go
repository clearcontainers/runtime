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

package libsnnet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/01org/ciao/database"
	"github.com/01org/ciao/ssntp/uuid"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/docker/libnetwork/drivers/remote/api"
	ipamapi "github.com/docker/libnetwork/ipams/remote/api"
	"github.com/gorilla/mux"
)

/*
The plugin acts as a slave to the ciao networking framework.
The goal here is to do no work in the plugin except inform the docker
daemon about the veth interface it needs to place inside the container

Hence the real flow will be as follows

0. Laucher creates and starts the DockerPlugin thread

Note: The launcher should be launched prior to the docker daemon.
      Also we need to configure docker daemon to not create its default
	  bridge and host networks as they cause problems.

1. Launcher gets a request to launch a container
   The request from the Controller to launcher already has the following
   information (IP Address, MAC Address and subnet for the VNIC)
   Note: Based on the current ciao design the gateway for the
   subnet can be inferred.

2. Launcher invokes ciao networking to create a Container Vnic

3. ciao Networking
     a. Creates a veth pair
     b. Assigns the macaddress to the container side of veth pair
     c. Attaches the veth to the tenant bridge (creating it if needed)
     d. Returns the fully configured docker side veth pair to Launcher
     e. Also returns if the subnet needs to be created
        (Note: This is the docker logical subnet)

4. (Launcher) if a subnet creation request was returned. Uses docker API
   or command line to
  docker network create -d=ciao
			--ipam-driver=ciao
			--subnet=<subnet.IPnet>
			--gateway=<gatewayIP>
			--opt "bridge"=<subnet.BridgeName>
			subnet.Name

	Note: Our custom IPAM driver is needed only when we start using overlapping
	subnets between tenants. Otherwise the default IPAM driver meets our needs.


	Note: Fully specifying the network creation and handing control to the
	ciao driver (-d) makes docker a passthrough for networking.
	Note: The docker IPAM seems to try to setup its own gateway. WHY?

	In the future any more information we need can also be sent as more
	options. e.g.
			--opt "cnci"=<subnet.cnciIP>


	- This in turn will result in a callback to our HTTP plugin.
 	  We will just return success as we are already aware of this
	  (Yes. We call docker and docker calls us back)

5. (Launcher) will then request docker to create & launch the container,
   again fully specifying the networking configuration.

   docker run -it --net=<subnet.Name> --ip=<instance.IP> --mac-address=<instance.MacAddresss> busybox

	WARNING. WARNING: There is a bug in the latest docker 1.10.03 (which has been fixed
	in the 1.11 dev version) which does not pass the --ip parameter to the
	remote IPAM plugin. Without this we cannot use our IPAM driver

6. The ciao docker plugin acts as both a network and IPAM remote plugin.
   It handles all the requests. Some of the more important ones are
     a. EndPointCreate: If the container is being created for the first time
        As we have already created the VNIC, we only need to cache the endpoint id to instance map
	 b. Join: When the end point is being placed inside the container
	    On Join the plugin will return back to docker the following information
           - name of the veth pair to place within the container
	       - the Ethernet device name prefix to be assigned to the logic interface
		     within the container (e.g. eth or eno)
	  	   - the default gw for the container
		   - any other static routes to be added within the container (if needed)

		   Note: We will delete only when the launcher tells us to tear down networking.
		         Not when docker logically tears down the network.

7. The docker daemon will use the values sent back by the plugin to launch the container
   Move the veth into the docker container and give it the logical name.
   Setup the IP address and gateway

*/

//DockerPluginCfg controls plugin attributes
//these may be overridden by the caller if needed
var DockerPluginCfg = struct {
	Name    string
	Dir     string
	Addr    string
	DataDir string
	DbFile  string
	Timeout time.Duration
}{
	Name:    "ciao",
	Dir:     "/etc/docker/plugins",
	Addr:    "127.0.0.1:9999",
	DataDir: "/var/lib/ciao/networking",
	DbFile:  "docker_plugin.db",
	Timeout: 1 * time.Second,
}

//DockerEpVal stores ciao VNIC info for a particular docker endpoint
type DockerEpVal struct {
	ID    string
	IP    string
	Cveth string
	Hveth string
}

//DockerNwVal stores ciao CN tenant bridge mapping
type DockerNwVal struct {
	Bridge  string
	Gateway net.IPNet
}

const (
	tableNetworkMap  = "NetworkMap"
	tableEndPointMap = "EndPointMap"
)

//DockerEpMap maintains the Endpoint UUID to ciao VNIC mapping
type DockerEpMap struct {
	sync.Mutex
	m map[string]*DockerEpVal //index: Docker End Point UUID
}

//NewTable creates a new map
func (d *DockerEpMap) NewTable() {
	d.m = make(map[string]*DockerEpVal)
}

//Name provides the name of the map
func (d *DockerEpMap) Name() string {
	return tableNetworkMap
}

//NewElement allocates and returns an endpoint value
func (d *DockerEpMap) NewElement() interface{} {
	return &DockerEpVal{}
}

//Add adds a value to the map with the specified key
func (d *DockerEpMap) Add(k string, v interface{}) error {
	val, ok := v.(*DockerEpVal)
	if !ok {
		return fmt.Errorf("Invalid value type %t", v)
	}
	d.m[k] = val
	return nil
}

//DockerNwMap maintains the Docker Network UUID to ciao Network mappings
type DockerNwMap struct {
	sync.Mutex
	m map[string]*DockerNwVal //index: Docker Network UUID
}

//NewTable creates a new map
func (d *DockerNwMap) NewTable() {
	d.m = make(map[string]*DockerNwVal)
}

//Name provides the name of the map
func (d *DockerNwMap) Name() string {
	return tableEndPointMap
}

//NewElement allocates and returns an network value
func (d *DockerNwMap) NewElement() interface{} {
	return &DockerNwVal{}
}

//Add adds a value to the map with the specified key
func (d *DockerNwMap) Add(k string, v interface{}) error {
	val, ok := v.(*DockerNwVal)
	if !ok {
		return fmt.Errorf("Invalid value type %t", v)
	}
	d.m[k] = val
	return nil
}

// DockerPlugin describes a single instance of a docker plugin
// In the current design the plugin acts as an IPAM and Network Plugin
type DockerPlugin struct {
	database.DbProvider //Database used to persist the Docker to ciao Mapping
	//This is needed as the Docker Daemon and ciao have
	//different life cycles and UUIDs
	*mux.Router
	*http.Server
	DockerEpMap
	DockerNwMap
	wg       sync.WaitGroup
	listener net.Listener
}

func sendResponse(resp interface{}, w http.ResponseWriter) {
	rb, err := json.Marshal(resp)
	Logger.Infof("Sending response := %v, %v", resp, err)
	fmt.Fprintf(w, "%s", rb)
	return
}

func getBody(r *http.Request) ([]byte, error) {
	body, err := ioutil.ReadAll(r.Body)
	Logger.Infof("URL [%s] Body [%s] Error [%v]", r.URL.Path[1:], string(body), err)
	return body, err
}

func handler(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	body, _ := getBody(r)
	resp := api.Response{}
	resp.Err = "Unhandled API request " + string(r.URL.Path[1:]) + " " + string(body)
	sendResponse(resp, w)
}

func handlerPluginActivate(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	_, _ = getBody(r)
	//TODO: Where is this encoding?
	resp := `{
    "Implements": ["NetworkDriver", "IpamDriver"]
}`
	fmt.Fprintf(w, "%s", resp)
}

func handlerGetCapabilities(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	_, _ = getBody(r)
	resp := api.GetCapabilityResponse{Scope: "local"}
	sendResponse(resp, w)
}

func handlerCreateNetwork(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := api.CreateNetworkResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := api.CreateNetworkRequest{}
	if err := json.Unmarshal(body, &req); err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	v, ok := req.Options["com.docker.network.generic"].(map[string]interface{})
	if !ok {
		resp.Err = "Error: network options incorrect or unspecified. Please provide bridge info"
		sendResponse(resp, w)
		return
	}

	bridge, ok := v["bridge"].(string)
	if !ok {
		resp.Err = "Error: network incorrect or unspecified. Please provide bridge info"
		sendResponse(resp, w)
		return
	}

	d.DockerNwMap.Lock()
	defer d.DockerNwMap.Unlock()

	/* Record the docker network UUID to ciao bridge mapping
	   This has to survive a plugin crash/restart and needs to be persisted
	*/
	d.DockerNwMap.m[req.NetworkID] = &DockerNwVal{
		Bridge:  bridge,
		Gateway: *req.IPv4Data[0].Gateway,
	}

	if err := d.DbAdd(tableNetworkMap, req.NetworkID, d.DockerNwMap.m[req.NetworkID]); err != nil {
		Logger.Errorf("Unable to update db %v", err)
	}

	sendResponse(resp, w)
}

func handlerDeleteNetwork(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := api.DeleteNetworkResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := api.DeleteNetworkRequest{}
	if err = json.Unmarshal(body, &req); err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	Logger.Infof("Delete Network := %v", req.NetworkID)

	/* Actual network delete would have already been done in ciao
	   Remove the UUID to bridge mapping in cache and in the
	   persistent data store
	*/
	d.DockerNwMap.Lock()
	bridge := d.DockerNwMap.m[req.NetworkID].Bridge
	delete(d.DockerNwMap.m, req.NetworkID)
	if err := d.DbDelete(tableNetworkMap, req.NetworkID); err != nil {
		Logger.Errorf("Unable to update db %v %v", bridge, err)
	}
	d.DockerNwMap.Unlock()

	sendResponse(resp, w)

	return
}

func handlerEndpointOperInfof(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := api.EndpointInfoResponse{}
	body, err := getBody(r)

	if err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := api.EndpointInfoRequest{}
	err = json.Unmarshal(body, &req)

	if err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	sendResponse(resp, w)
}

func handlerCreateEndpoint(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := api.CreateEndpointResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := api.CreateEndpointRequest{}
	if err = json.Unmarshal(body, &req); err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	if req.Interface.Address == "" {
		resp.Err = "Error: IP Address parameter not provided in docker run"
		sendResponse(resp, w)
		return
	}

	d.DockerNwMap.Lock()
	bridge := d.DockerNwMap.m[req.NetworkID].Bridge
	d.DockerNwMap.Unlock()

	if bridge == "" {
		resp.Err = "Error: incompatible network"
		sendResponse(resp, w)
		return
	}

	// These are already setup by the SDN controller
	// Get the alias for the VNIC based on the bridge and IP
	subnetTuple := strings.TrimPrefix(bridge, bridgePrefix)
	ip, _, err := net.ParseCIDR(req.Interface.Address)
	if err != nil {
		resp.Err = "Error: Invalid IP Address " + err.Error()
		sendResponse(resp, w)
		return
	}

	vnicID := fmt.Sprintf("%s%s##%s", vnicPrefix, subnetTuple, ip.String())

	//We can also get this directly from the SDN controller.
	//However that will prevent the plugin from being its own service
	//in the future
	vnic, err := NewContainerVnic(vnicID)
	if err != nil {
		resp.Err = "Error: invalid interface " + err.Error()
		sendResponse(resp, w)
		return
	}

	if err := vnic.GetDevice(); err != nil {
		resp.Err = "Error: invalid interface " + err.Error()
		sendResponse(resp, w)
		return
	}

	d.DockerEpMap.Lock()
	defer d.DockerEpMap.Unlock()

	d.DockerEpMap.m[req.EndpointID] = &DockerEpVal{
		ID:    vnicID,
		IP:    req.Interface.Address,
		Hveth: vnic.InterfaceName(),
		Cveth: vnic.PeerName(),
	}

	if err := d.DbAdd(tableEndPointMap, req.EndpointID, d.DockerEpMap.m[req.EndpointID]); err != nil {
		Logger.Errorf("Unable to update db %v", err)
	}

	sendResponse(resp, w)
}

func handlerDeleteEndpoint(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := api.DeleteEndpointResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := api.DeleteEndpointRequest{}
	if err = json.Unmarshal(body, &req); err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	d.DockerEpMap.Lock()
	//ID := d.DockerEpMap.m[req.EndpointID].ID
	delete(d.DockerEpMap.m, req.EndpointID)
	if err := d.DbDelete(tableEndPointMap, req.EndpointID); err != nil {
		Logger.Errorf("Unable to update db %v", err)
	}
	d.DockerEpMap.Unlock()

	/*
		// This will be done in the SDN controller once the
		// container is deleted. However at this point there is
		// a disconnect between the docker data base and SDN database
		vnic, err := NewContainerVnic(ID)
		if err != nil {
			if err := vnic.GetDevice(); err != nil {
				Logger.Errorf("Link has not been deleted %v", err)
				if err := vnic.Destroy(); err != nil {
					Logger.Errorf("Unable to delete link %v", err)
				}
			}
		}
	*/

	sendResponse(resp, w)
}

func handlerJoin(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := api.JoinResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := api.JoinRequest{}
	if err := json.Unmarshal(body, &req); err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	d.DockerNwMap.Lock()
	d.DockerEpMap.Lock()
	nm := d.DockerNwMap.m[req.NetworkID]
	em := d.DockerEpMap.m[req.EndpointID]
	d.DockerNwMap.Unlock()
	d.DockerEpMap.Unlock()

	resp.Gateway = nm.Gateway.IP.String()
	resp.InterfaceName = &api.InterfaceName{
		SrcName:   em.Cveth,
		DstPrefix: "eth",
	}
	sendResponse(resp, w)
}

func handlerLeave(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := api.LeaveResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := api.LeaveRequest{}
	if err := json.Unmarshal(body, &req); err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	sendResponse(resp, w)
}

func handlerDiscoverNew(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := api.DiscoveryResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := api.DiscoveryNotification{}
	if err := json.Unmarshal(body, &req); err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	sendResponse(resp, w)
}

func handlerDiscoverDelete(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := api.DiscoveryResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := api.DiscoveryNotification{}
	if err := json.Unmarshal(body, &req); err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	sendResponse(resp, w)
}

func handlerExternalConnectivity(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := api.ProgramExternalConnectivityResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := api.ProgramExternalConnectivityRequest{}
	if err := json.Unmarshal(body, &req); err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	sendResponse(resp, w)
}

func handlerRevokeExternalConnectivity(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := api.RevokeExternalConnectivityResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := api.RevokeExternalConnectivityResponse{}
	if err := json.Unmarshal(body, &req); err != nil {
		resp.Err = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	sendResponse(resp, w)
}

func ipamGetCapabilities(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	_, _ = getBody(r)
	resp := ipamapi.GetCapabilityResponse{RequiresMACAddress: true}
	sendResponse(resp, w)
}

func ipamGetDefaultAddressSpaces(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := ipamapi.GetAddressSpacesResponse{}
	_, _ = getBody(r)

	resp.GlobalDefaultAddressSpace = ""
	resp.LocalDefaultAddressSpace = ""
	sendResponse(resp, w)
}

func ipamRequestPool(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := ipamapi.RequestPoolResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Error = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := ipamapi.RequestPoolRequest{}
	if err := json.Unmarshal(body, &req); err != nil {
		resp.Error = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	resp.PoolID = uuid.Generate().String()
	resp.Pool = req.Pool
	sendResponse(resp, w)
}

func ipamReleasePool(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := ipamapi.ReleasePoolResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Error = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := ipamapi.ReleasePoolRequest{}
	if err := json.Unmarshal(body, &req); err != nil {
		resp.Error = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	sendResponse(resp, w)
}

func ipamRequestAddress(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := ipamapi.RequestAddressResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Error = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := ipamapi.RequestAddressRequest{}
	if err := json.Unmarshal(body, &req); err != nil {
		resp.Error = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	//TODO: Should come from the subnet mask for the subnet
	if req.Address != "" {
		resp.Address = req.Address + "/24"
	} else {
		//DOCKER BUG: The preferred address supplied in --ip does not show up.
		//Bug fixed in docker 1.11
		resp.Error = "Error: Request does not have IP address. Specify using --ip"
	}
	sendResponse(resp, w)
}

func ipamReleaseAddress(d *DockerPlugin, w http.ResponseWriter, r *http.Request) {
	resp := ipamapi.ReleaseAddressResponse{}

	body, err := getBody(r)
	if err != nil {
		resp.Error = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	req := ipamapi.ReleaseAddressRequest{}
	if err := json.Unmarshal(body, &req); err != nil {
		resp.Error = "Error: " + err.Error()
		sendResponse(resp, w)
		return
	}

	sendResponse(resp, w)
}

//DockerHandler is the default handler for unhandled events
//It returns error to the caller
func DockerHandler(d *DockerPlugin,
	fn func(*DockerPlugin, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(d, w, r)
	}
}

//NewDockerPlugin instantiates a new Docker Plugin instance
func NewDockerPlugin() *DockerPlugin {
	database.Logger = Logger

	return &DockerPlugin{
		DbProvider: database.NewBoltDBProvider(),
	}
}

func createDockerPluginConfig(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("Unable to create plugin directory (%s) %v", dir, err)
	}

	pluginSpecFile := path.Join(dir, DockerPluginCfg.Name+".json")

	var spec struct {
		Name      string
		Addr      string
		TLSConfig *tlsconfig.Options
	}
	spec.Name = DockerPluginCfg.Name
	spec.Addr = "http://" + DockerPluginCfg.Addr
	spec.TLSConfig = &tlsconfig.Options{}
	b, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("Unable to create plugin spec (%v) %v", spec, err)
	}
	err = ioutil.WriteFile(pluginSpecFile, b, 0644)
	if err != nil {
		return fmt.Errorf("Unable to create plugin file (%s) %v", pluginSpecFile, err)
	}
	return nil
}

//Init initializes the docker Plugin. Has to be called after init, but before
//start.
func (d *DockerPlugin) Init() error {
	d.DockerEpMap.m = make(map[string]*DockerEpVal)
	d.DockerNwMap.m = make(map[string]*DockerNwVal)

	if err := createDockerPluginConfig(DockerPluginCfg.Dir); err != nil {
		return err
	}

	if err := d.DbInit(DockerPluginCfg.DataDir, DockerPluginCfg.DbFile); err != nil {
		return err
	}
	if err := d.DbTableRebuild(&d.DockerNwMap); err != nil {
		return err
	}
	if err := d.DbTableRebuild(&d.DockerEpMap); err != nil {
		return err
	}

	if d.Router == nil {
		d.Router = mux.NewRouter()
	}
	r := d.Router
	r.HandleFunc("/Plugin.Activate", DockerHandler(d, handlerPluginActivate))
	r.HandleFunc("/NetworkDriver.GetCapabilities", DockerHandler(d, handlerGetCapabilities))
	r.HandleFunc("/NetworkDriver.CreateNetwork", DockerHandler(d, handlerCreateNetwork))
	r.HandleFunc("/NetworkDriver.DeleteNetwork", DockerHandler(d, handlerDeleteNetwork))
	r.HandleFunc("/NetworkDriver.CreateEndpoint", DockerHandler(d, handlerCreateEndpoint))
	r.HandleFunc("/NetworkDriver.DeleteEndpoint", DockerHandler(d, handlerDeleteEndpoint))
	r.HandleFunc("/NetworkDriver.EndpointOperInfo", DockerHandler(d, handlerEndpointOperInfof))
	r.HandleFunc("/NetworkDriver.Join", DockerHandler(d, handlerJoin))
	r.HandleFunc("/NetworkDriver.Leave", DockerHandler(d, handlerLeave))
	r.HandleFunc("/NetworkDriver.DiscoverNew", DockerHandler(d, handlerDiscoverNew))
	r.HandleFunc("/NetworkDriver.DiscoverDelete", DockerHandler(d, handlerDiscoverDelete))
	r.HandleFunc("/NetworkDriver.ProgramExternalConnectivity", DockerHandler(d, handlerExternalConnectivity))
	r.HandleFunc("/NetworkDriver.RevokeExternalConnectivity", DockerHandler(d, handlerRevokeExternalConnectivity))

	r.HandleFunc("/IpamDriver.GetCapabilities", DockerHandler(d, ipamGetCapabilities))
	r.HandleFunc("/IpamDriver.GetDefaultAddressSpaces", DockerHandler(d, ipamGetDefaultAddressSpaces))
	r.HandleFunc("/IpamDriver.RequestPool", DockerHandler(d, ipamRequestPool))
	r.HandleFunc("/IpamDriver.ReleasePool", DockerHandler(d, ipamReleasePool))
	r.HandleFunc("/IpamDriver.RequestAddress", DockerHandler(d, ipamRequestAddress))
	r.HandleFunc("/IpamDriver.ReleaseAddress", DockerHandler(d, ipamReleaseAddress))

	r.HandleFunc("/", DockerHandler(d, handler))
	return nil
}

// Start the DockerPlugin. This activates the HTTP/HTTPS server
// The DockerPlugin has to be started prior to the launch of the
// Docker Daemon
func (d *DockerPlugin) Start() error {
	var err error
	d.listener, err = net.Listen("tcp", DockerPluginCfg.Addr)
	if err != nil {
		return fmt.Errorf("unable to create listener: %v", err)
	}

	d.Server = &http.Server{
		Handler: d.Router,
	}

	d.wg.Add(1)
	go func() {
		Logger.Infof("Starting HTTP Server")
		err := d.Server.Serve(d.listener)
		if err != nil {
			Logger.Errorf("Unable to start HTTP Server [%v]", err)
		}
		d.wg.Done()
	}()
	return nil
}

//Stop the DockerPlugin
//The DockerPlugin has to be stopped after the Docker Daemon
//has been stopped. If the plugin is stopped when the docker
//daemon is still active the docker daemon has a timeout and
//retry mechanism. Hence if the docker plugin is restarted
//within the retry windows, the docker APIs will still succeed
func (d *DockerPlugin) Stop() error {
	if err := d.listener.Close(); err != nil {
		return fmt.Errorf("Unable to shutdown http server: %v", err)
	}
	d.wg.Wait()
	Logger.Infof("Docker plugin has shut down")
	return nil
}

//Close the DockerPlugin
//This has to be called if the caller has ever performed an Init()
//failing to close may lead to database corruption
func (d *DockerPlugin) Close() error {
	return d.DbClose()
}
