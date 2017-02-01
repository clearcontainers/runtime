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

package datastore

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"github.com/golang/glog"
	sqlite3 "github.com/mattn/go-sqlite3"
)

type sqliteDB struct {
	db            *sql.DB
	tdb           *sql.DB
	dbName        string
	tdbName       string
	tables        []persistentData
	tableInitPath string
	workloadsPath string
	dbLock        *sync.Mutex
	tdbLock       *sync.RWMutex
}

type persistentData interface {
	Init() error
	Populate() error
	Create(...string) error
	Name() string
	DB() *sql.DB
}

type namedData struct {
	ds   *sqliteDB
	name string
	db   *sql.DB
}

func (d namedData) Create(record ...string) (err error) {
	err = d.ds.create(d.name, record)
	return
}

func (d namedData) Populate() (err error) {
	return nil
}

func (d namedData) Name() (name string) {
	return d.name
}

func (d namedData) DB() *sql.DB {
	return d.db
}

// ReadCsv will return nil without an error if the file does not exist
func (d namedData) ReadCsv() ([][]string, error) {
	f, err := os.Open(fmt.Sprintf("%s/%s.csv", d.ds.tableInitPath, d.name))
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	r.Comment = '#'

	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	return records, nil
}

type logData struct {
	namedData
}

func (d logData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS log
		(
		id integer primary key,
		tenant_id varchar(32),
		type string,
		message string,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
		);`

	return d.ds.exec(d.db, cmd)
}

type subnetData struct {
	namedData
}

func (d subnetData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS tenant_network
		(
		tenant_id varchar(32),
		subnet int,
		rest int,
		foreign key(tenant_id) references tenants(id)
		);`

	return d.ds.exec(d.db, cmd)
}

// Handling of Limit specific Data
type limitsData struct {
	namedData
}

func (d limitsData) Populate() error {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		resourceID, _ := strconv.Atoi(line[0])
		tenantID := line[1]
		maxValue, _ := strconv.Atoi(line[2])
		err = d.ds.create(d.name, resourceID, tenantID, maxValue)
		if err != nil {
			glog.V(2).Info("could not add limit: ", err)
		}
	}

	return err
}

func (d limitsData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS limits
		(
		resource_id integer,
		tenant_id varchar(32),
		max_value integer,
		foreign key(resource_id) references resources(id),
		foreign key(tenant_id) references tenants(id)
		);`

	return d.ds.exec(d.db, cmd)
}

// Handling of Instance specific data
type instanceData struct {
	namedData
}

func (d instanceData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS instances
		(
		id string primary key,
		tenant_id string,
		workload_id string,
		mac_address string,
		ip string,
		create_time DATETIME,
		foreign key(tenant_id) references tenants(id),
		foreign key(workload_id) references workload_template(id),
		unique(tenant_id, ip, mac_address)
		);`

	return d.ds.exec(d.db, cmd)
}

// Volume Data
type blockData struct {
	namedData
}

func (d blockData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS block_data
		(
		id string primary_key,
		tenant_id string,
		size integer,
		state string,
		create_time DATETIME,
		name string,
		description string,
		foreign key(tenant_id) references tenants(id)
		);`

	return d.ds.exec(d.db, cmd)
}

type attachments struct {
	namedData
}

func (d attachments) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS attachments
		(
		id string primary key,
		instance_id string,
		block_id string,
		ephemeral int,
		boot int,
		foreign key(instance_id) references instances(id),
		foreign key(block_id) references block_data(id)
		);`

	return d.ds.exec(d.db, cmd)
}

// workload storage resources

type workloadStorage struct {
	namedData
}

func (d workloadStorage) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS workload_storage
	        (
		workload_id string,
		volume_id string,
		bootable int,
		ephemeral int,
		size integer,
		source_type string,
		source_id string,
		tag string,
		foreign key(workload_id) references workloads(id),
		foreign key(volume_id) references block_data(id)
		);`

	return d.ds.exec(d.db, cmd)
}

func (d workloadStorage) Populate() error {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		workloadID := line[0]
		volumeID := line[1]
		bootable := line[2]
		ephemeral := line[3]
		size := line[4]
		sourceType := line[5]
		sourceID := line[6]
		tag := line[7]

		err = d.ds.create(d.name, workloadID, volumeID, bootable, ephemeral, size, sourceType, sourceID, tag)
		if err != nil {
			glog.V(2).Info("could not add workload storage", err)
		}
	}

	return err
}

// Resources data
type resourceData struct {
	namedData
}

func (d resourceData) Populate() error {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		id, _ := strconv.Atoi(line[0])
		name := line[1]
		err = d.ds.create(d.name, id, name)
		if err != nil {
			glog.V(2).Info("could not add resource: ", err)
		}
	}

	return err
}

func (d resourceData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS resources
		(
		id int primary key,
		name text
		);`

	return d.ds.exec(d.db, cmd)
}

// Tenants data
type tenantData struct {
	namedData
}

func (d tenantData) Populate() error {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		id := line[0]
		name := line[1]
		mac := line[2]

		err = d.ds.create(d.name, id, name, "", mac, "")
		if err != nil {
			glog.V(2).Info("could not add tenant: ", err)
		}
	}

	return err
}

func (d tenantData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS tenants
		(
		id varchar(32) primary key,
		name text,
		cnci_id varchar(32) default null,
		cnci_mac string default null,
		cnci_ip string default null
		);`

	return d.ds.exec(d.db, cmd)
}

// usage data
type usageData struct {
	namedData
}

func (d usageData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS usage
		(
		instance_id string,
		resource_id int,
		value int,
		foreign key(instance_id) references instances(id),
		foreign key(resource_id) references resources(id)
		);
		CREATE UNIQUE INDEX IF NOT EXISTS myindex
		ON usage(instance_id, resource_id);`

	return d.ds.exec(d.db, cmd)
}

// workload resources
type workloadResourceData struct {
	namedData
}

func (d workloadResourceData) Populate() error {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		workloadID := line[0]
		resourceID, _ := strconv.Atoi(line[1])
		defaultValue, _ := strconv.Atoi(line[2])
		estimatedValue, _ := strconv.Atoi(line[3])
		mandatory, _ := strconv.Atoi(line[4])
		err = d.ds.create(d.name, workloadID, resourceID, defaultValue, estimatedValue, mandatory)
		if err != nil {
			glog.V(2).Info("could not add workload: ", err)
		}
	}

	return err
}

func (d workloadResourceData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS workload_resources
		(
		workload_id varchar(32),
		resource_id int,
		default_value int,
		estimated_value int,
		mandatory int,
		foreign key(workload_id) references workload_template(id),
		foreign key(resource_id) references resources(id)
		);
		CREATE UNIQUE INDEX IF NOT EXISTS wlr_index
		ON workload_resources(workload_id, resource_id);`

	return d.ds.exec(d.db, cmd)
}

// workload template data
type workloadTemplateData struct {
	namedData
}

func (d workloadTemplateData) Populate() error {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		id := line[0]
		description := line[1]
		filename := line[2]
		fwType := line[3]
		vmType := line[4]
		imageID := line[5]
		imageName := line[6]
		internal := line[7]
		err = d.ds.create(d.name, id, description, filename, fwType, vmType, imageID, imageName, internal)
		if err != nil {
			glog.V(2).Info("could not add workload: ", err)
		}
	}

	return err
}

func (d workloadTemplateData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS workload_template
		(
		id varchar(32) primary key,
		description text,
		filename text,
		fw_type text,
		vm_type text,
		image_id varchar(32),
		image_name text,
		internal integer
		);`

	return d.ds.exec(d.db, cmd)
}

// statistics
type nodeStatisticsData struct {
	namedData
}

func (d nodeStatisticsData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS node_statistics
		(
			id integer primary key autoincrement not null,
			node_id varchar(32),
			mem_total_mb int,
			mem_available_mb int,
			disk_total_mb int,
			disk_available_mb int,
			load int,
			cpus_online int,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
		);`

	return d.ds.exec(d.db, cmd)
}

type instanceStatisticsData struct {
	namedData
}

func (d instanceStatisticsData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS instance_statistics
		(
			id integer primary key autoincrement not null,
			instance_id varchar(32),
			memory_usage_mb int,
			disk_usage_mb int,
			cpu_usage int,
			state string,
			node_id varchar(32),
			ssh_ip string,
			ssh_port int,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
		);`

	return d.ds.exec(d.db, cmd)
}

type frameStatisticsData struct {
	namedData
}

func (d frameStatisticsData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS frame_statistics
		(
			id integer primary key autoincrement not null,
			label string,
			type string,
			operand string,
			start_timestamp DATETIME,
			end_timestamp DATETIME
		);`

	return d.ds.exec(d.db, cmd)
}

type traceData struct {
	namedData
}

func (d traceData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS trace_data
		(
			id integer primary key autoincrement not null,
			frame_id int,
			ssntp_uuid varchar(32),
			tx_timestamp DATETIME,
			rx_timestamp DATETIME,
			foreign key(frame_id) references frame_statistics(id)
		);`

	return d.ds.exec(d.db, cmd)
}

type poolData struct {
	namedData
}

func (d poolData) Populate() error {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		poolID := line[0]
		poolName := line[1]
		free, _ := strconv.Atoi(line[2])
		total, _ := strconv.Atoi(line[3])
		err = d.ds.create(d.name, poolID, poolName, free, total)
		if err != nil {
			glog.V(2).Info("could not add pool: ", err)
		}
	}

	return err
}

func (d poolData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS pools
		(
			id varchar(32),
			name string,
			free int,
			total int,
			PRIMARY KEY(id, name)
		);`

	return d.ds.exec(d.db, cmd)
}

type subnetPoolData struct {
	namedData
}

func (d subnetPoolData) Populate() error {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		subnetID := line[0]
		poolID := line[1]
		cidr := line[2]
		err = d.ds.create(d.name, subnetID, poolID, cidr)
		if err != nil {
			glog.V(2).Info("could not add subnet: ", err)
		}
	}

	return err
}

func (d subnetPoolData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS subnet_pool
		(
			id varchar(32) primary key,
			pool_id varchar(32),
			cidr string
		);`

	return d.ds.exec(d.db, cmd)
}

type addressData struct {
	namedData
}

func (d addressData) Populate() error {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		addressID := line[0]
		poolID := line[1]
		address := line[2]
		err = d.ds.create(d.name, addressID, poolID, address)
		if err != nil {
			glog.V(2).Info("could not add address: ", err)
		}
	}

	return err
}

func (d addressData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS address_pool
		(
			id varchar(32) primary key,
			pool_id varchar(32),
			address string
		);`

	return d.ds.exec(d.db, cmd)
}

type mappedIPData struct {
	namedData
}

func (d mappedIPData) Populate() error {
	lines, err := d.ReadCsv()
	if err != nil {
		return err
	}

	for _, line := range lines {
		mappingID := line[0]
		externalIP := line[1]
		instanceID := line[2]
		poolID := line[3]
		err = d.ds.create(d.name, mappingID, externalIP, instanceID, poolID)
		if err != nil {
			glog.V(2).Info("could not add mapping: ", err)
		}
	}

	return err
}

func (d mappedIPData) Init() error {
	cmd := `CREATE TABLE IF NOT EXISTS mapped_ips
		(
			id varchar(32) primary key,
			external_ip string,
			instance_id varchar(32),
			pool_id varchar(32)
		);`

	return d.ds.exec(d.db, cmd)
}

func (ds *sqliteDB) exec(db *sql.DB, cmd string) error {
	glog.V(2).Info("exec: ", cmd)

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(cmd)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	return err
}

// This function is deprecated and will be removed soon. It should not be used
// for newly written or updated code.
func (ds *sqliteDB) create(tableName string, record ...interface{}) error {
	// get database location of this table
	db := ds.getTableDB(tableName)

	if db == nil {
		return errors.New("Bad table name")
	}

	var values []string
	for _, val := range record {
		v := reflect.ValueOf(val)

		var newval string

		// enclose strings in quotes to not confuse sqlite
		if v.Kind() == reflect.String {
			newval = fmt.Sprintf("'%v'", val)
		} else {
			newval = fmt.Sprintf("%v", val)
		}

		values = append(values, newval)
	}

	args := strings.Join(values, ",")
	cmd := "INSERT or IGNORE into " + tableName + " VALUES (" + args + ");"

	return ds.exec(db, cmd)
}

func (ds *sqliteDB) getTableDB(name string) *sql.DB {
	for _, table := range ds.tables {
		n := table.Name()
		if n == name {
			return table.DB()
		}
	}
	return nil
}

// init initializes the private data for the database object.
// The sql tables are populated with initial data from csv
// files if this is the first time the database has been
// created.  The datastore caches are also filled.
func (ds *sqliteDB) init(config Config) error {
	u, err := url.Parse(config.PersistentURI)
	if err != nil {
		return fmt.Errorf("Invalid URL (%s) for persistent data store: %v", config.PersistentURI, err)
	}

	if u.Scheme == "file" {
		dbDir := filepath.Dir(u.Path)
		err = os.MkdirAll(dbDir, 0755)
		if err != nil && dbDir != "." {
			return fmt.Errorf("Unable to create db directory (%s) %v", dbDir, err)
		}
	}

	err = ds.Connect(config.PersistentURI, config.TransientURI)
	if err != nil {
		return err
	}

	ds.dbLock = &sync.Mutex{}
	ds.tdbLock = &sync.RWMutex{}

	ds.tables = []persistentData{
		resourceData{namedData{ds: ds, name: "resources", db: ds.db}},
		tenantData{namedData{ds: ds, name: "tenants", db: ds.db}},
		limitsData{namedData{ds: ds, name: "limits", db: ds.db}},
		instanceData{namedData{ds: ds, name: "instances", db: ds.db}},
		workloadTemplateData{namedData{ds: ds, name: "workload_template", db: ds.db}},
		workloadResourceData{namedData{ds: ds, name: "workload_resources", db: ds.db}},
		usageData{namedData{ds: ds, name: "usage", db: ds.db}},
		nodeStatisticsData{namedData{ds: ds, name: "node_statistics", db: ds.tdb}},
		logData{namedData{ds: ds, name: "log", db: ds.tdb}},
		subnetData{namedData{ds: ds, name: "tenant_network", db: ds.db}},
		instanceStatisticsData{namedData{ds: ds, name: "instance_statistics", db: ds.tdb}},
		frameStatisticsData{namedData{ds: ds, name: "frame_statistics", db: ds.tdb}},
		traceData{namedData{ds: ds, name: "trace_data", db: ds.tdb}},
		blockData{namedData{ds: ds, name: "block_data", db: ds.db}},
		attachments{namedData{ds: ds, name: "attachments", db: ds.db}},
		workloadStorage{namedData{ds: ds, name: "workload_storage", db: ds.db}},
		poolData{namedData{ds: ds, name: "pools", db: ds.db}},
		subnetPoolData{namedData{ds: ds, name: "subnet_pool", db: ds.db}},
		addressData{namedData{ds: ds, name: "address_pool", db: ds.db}},
		mappedIPData{namedData{ds: ds, name: "mapped_ips", db: ds.db}},
	}

	ds.tableInitPath = config.InitTablesPath
	ds.workloadsPath = config.InitWorkloadsPath

	for _, table := range ds.tables {
		err = table.Init()
		if err != nil {
			return err
		}
	}

	for _, table := range ds.tables {
		err = table.Populate()
		if err != nil {
			return err
		}
	}

	return nil
}

var pSQLLiteConfig = []string{
	"PRAGMA page_size = 32768",
	"PRAGMA synchronous = OFF",
	"PRAGMA temp_store = MEMORY",
	"PRAGMA busy_timeout = 1000",
	"PRAGMA journal_mode=WAL",
}

func (ds *sqliteDB) sqliteConnect(name string, URI string, config []string) (*sql.DB, error) {
	datastore, err := sql.Open(name, URI)
	if err != nil {
		return nil, err
	}

	for i := range config {
		_, err = datastore.Exec(config[i])
		if err != nil {
			glog.Warning(err)
		}
	}

	err = datastore.Ping()
	if err != nil {
		glog.Warning(err)
		return nil, err
	}

	return datastore, nil
}

// Connect creates two sqlite3 databases.  One database is for
// persistent state that needs to be restored on restart, the
// other is for transient data that does not need to be restored
// on restart.
func (ds *sqliteDB) Connect(persistentURI string, transientURI string) error {
	sql.Register(transientURI, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			cmd := fmt.Sprintf("ATTACH '%s' AS tdb", transientURI)
			conn.Exec(cmd, nil)
			return nil
		},
	})

	db, err := ds.sqliteConnect(transientURI, persistentURI, pSQLLiteConfig)
	if err != nil {
		return err
	}

	ds.db = db
	ds.dbName = persistentURI

	sql.Register(persistentURI, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			cmd := fmt.Sprintf("ATTACH '%s' AS db", persistentURI)
			conn.Exec(cmd, nil)
			return nil
		},
	})

	tdb, err := ds.sqliteConnect(persistentURI, transientURI, pSQLLiteConfig)
	if err != nil {
		return err
	}

	ds.tdb = tdb
	ds.tdbName = transientURI

	return err
}

// Disconnect is used to close the connection to the sql database
func (ds *sqliteDB) disconnect() {
	ds.db.Close()
}

func (ds *sqliteDB) logEvent(tenantID string, eventType string, message string) error {
	datastore := ds.getTableDB("log")

	ds.tdbLock.Lock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.tdbLock.Unlock()
		return err
	}

	_, err = tx.Exec("INSERT INTO log (tenant_id, type, message) VALUES (?, ?, ?)", tenantID, eventType, message)
	if err != nil {
		tx.Rollback()
		ds.tdbLock.Unlock()
		return err
	}

	tx.Commit()

	ds.tdbLock.Unlock()

	return err
}

// ClearLog will remove all the event entries from the event log
func (ds *sqliteDB) clearLog() error {
	db := ds.getTableDB("log")

	ds.tdbLock.Lock()

	err := ds.exec(db, "DELETE FROM log")
	if err != nil {
		glog.V(2).Info("could not clear log: ", err)
	}

	ds.tdbLock.Unlock()

	return err
}

// GetCNCIWorkloadID returns the UUID of the workload template
// for the CNCI workload
func (ds *sqliteDB) getCNCIWorkloadID() (string, error) {
	var ID string

	db := ds.getTableDB("workload_template")

	err := db.QueryRow("SELECT id FROM workload_template WHERE description = 'CNCI'").Scan(&ID)
	if err != nil {
		return "", err
	}

	return ID, nil
}

func (ds *sqliteDB) getConfig(ID string) (string, error) {
	var configFile string

	db := ds.getTableDB("workload_template")

	err := db.QueryRow("SELECT filename FROM workload_template where id = ?", ID).Scan(&configFile)

	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/%s", ds.workloadsPath, configFile)
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	config := string(bytes)

	return config, nil
}

func (ds *sqliteDB) getWorkloadDefaults(ID string) ([]payloads.RequestedResource, error) {
	query := `SELECT resources.name, default_value, mandatory FROM workload_resources
		  JOIN resources
		  ON workload_resources.resource_id=resources.id
		  WHERE workload_id = ?`

	db := ds.getTableDB("workload_resources")

	rows, err := db.Query(query, ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var defaults []payloads.RequestedResource

	for rows.Next() {
		var val int
		var rname string
		var mandatory bool

		err = rows.Scan(&rname, &val, &mandatory)
		if err != nil {
			return nil, err
		}
		r := payloads.RequestedResource{
			Type:      payloads.Resource(rname),
			Value:     val,
			Mandatory: mandatory,
		}
		defaults = append(defaults, r)
	}

	return defaults, nil
}

// lock must be held by caller
func (ds *sqliteDB) getResources(tx *sql.Tx) (map[string]int, error) {
	m := make(map[string]int)

	query := `SELECT id, name from resources`

	rows, err := tx.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		err := rows.Scan(&id, &name)
		if err != nil {
			return nil, err
		}

		m[name] = id
	}

	return m, nil
}

// lock must be held by caller
func (ds *sqliteDB) createWorkloadDefault(tx *sql.Tx, workloadID string, resourceID int, resource payloads.RequestedResource) error {

	_, err := tx.Exec("INSERT INTO workload_resources (workload_id, resource_id, default_value, estimated_value, mandatory) VALUES (?, ?, ?, ?, ?)", workloadID, resourceID, resource.Value, resource.Value, resource.Mandatory)

	return err
}

// lock must be held by caller
func (ds *sqliteDB) createWorkloadStorage(tx *sql.Tx, workloadID string, storage *types.StorageResource) error {
	_, err := tx.Exec("INSERT INTO workload_storage (workload_id, volume_id, bootable, ephemeral, size, source_type, source_id, tag) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", workloadID, storage.ID, storage.Bootable, storage.Ephemeral, storage.Size, string(storage.SourceType), storage.SourceID, storage.Tag)

	return err
}

func (ds *sqliteDB) getWorkloadStorage(ID string) ([]types.StorageResource, error) {
	query := `SELECT volume_id, bootable, ephemeral, size,
			 source_type, source_id, tag
		  FROM 	workload_storage
		  WHERE workload_id = ?`

	rows, err := ds.db.Query(query, ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := []types.StorageResource{}
	var sourceType string

	for rows.Next() {
		var r types.StorageResource
		err := rows.Scan(&r.ID, &r.Bootable, &r.Ephemeral, &r.Size, &sourceType, &r.SourceID, &r.Tag)

		if err != nil {
			return []types.StorageResource{}, err
		}
		r.SourceType = types.SourceType(sourceType)
		res = append(res, r)
	}
	return res, nil
}

func (ds *sqliteDB) addLimit(tenantID string, resourceID int, limit int) error {
	ds.dbLock.Lock()
	err := ds.create("limits", resourceID, tenantID, limit)
	ds.dbLock.Unlock()

	return err
}

func (ds *sqliteDB) getTenantResources(ID string) ([]*types.Resource, error) {
	query := `WITH instances_usage AS
		 (
			 SELECT resource_id, value
			 FROM usage
			 LEFT JOIN instances
			 ON usage.instance_id = instances.id
			 WHERE instances.tenant_id = ?
		 )
		 SELECT resources.name, resources.id, limits.max_value,
		 CASE resources.id
		 WHEN resources.id = 1 then
		 (
			 SELECT COUNT(instances.id)
			 FROM instances
			 WHERE instances.tenant_id = ?
		 )
		 ELSE SUM(instances_usage.value)
		 END
		 FROM resources
		 LEFT JOIN instances_usage
		 ON instances_usage.resource_id = resources.id
		 LEFT JOIN limits
		 ON resources.id=limits.resource_id
		 AND limits.tenant_id = ?
		 GROUP BY resources.id`

	datastore := ds.db

	rows, err := datastore.Query(query, ID, ID, ID)
	if err != nil {
		glog.Warning("Failed to get tenant usage")
		return nil, err
	}
	defer rows.Close()

	var resources []*types.Resource

	for rows.Next() {
		var id int
		var name string
		var sqlMaxVal sql.NullInt64
		var sqlCurVal sql.NullInt64
		var maxVal = -1
		var curVal = 0

		err = rows.Scan(&name, &id, &sqlMaxVal, &sqlCurVal)
		if err != nil {
			return nil, err
		}

		if sqlMaxVal.Valid {
			maxVal = int(sqlMaxVal.Int64)
		}

		if sqlCurVal.Valid {
			curVal = int(sqlCurVal.Int64)
		}

		r := types.Resource{
			Rname: name,
			Rtype: id,
			Limit: maxVal,
			Usage: curVal,
		}

		resources = append(resources, &r)
	}

	return resources, nil
}

func (ds *sqliteDB) addTenant(ID string, MAC string) error {
	ds.dbLock.Lock()
	err := ds.create("tenants", ID, "", "", MAC, "")
	ds.dbLock.Unlock()

	return err
}

func (ds *sqliteDB) getTenant(ID string) (*tenant, error) {
	query := `SELECT	tenants.id,
				tenants.name,
				tenants.cnci_id,
				tenants.cnci_mac,
				tenants.cnci_ip
		  FROM tenants
		  WHERE tenants.id = ?`

	datastore := ds.db

	row := datastore.QueryRow(query, ID)

	t := &tenant{}

	err := row.Scan(&t.ID, &t.Name, &t.CNCIID, &t.CNCIMAC, &t.CNCIIP)
	if err != nil {
		glog.Warning("unable to retrieve tenant from tenants")

		if err == sql.ErrNoRows {
			// not an error, it's just not there.
			err = nil
		}

		return nil, err
	}

	// for these items below, its ok to get err returned
	// because a tenant could simply not have used any
	// resources or networks yet.
	t.Resources, err = ds.getTenantResources(ID)
	if err != nil {
		glog.V(2).Info(err)
	}

	err = ds.getTenantNetwork(t)
	if err != nil {
		glog.V(2).Info(err)
	}

	t.instances, err = ds.getTenantInstances(t.ID)
	if err != nil {
		glog.V(2).Info(err)
	}

	t.devices, err = ds.getTenantDevices(t.ID)

	return t, err
}

func (ds *sqliteDB) getWorkload(id string) (*workload, error) {
	datastore := ds.db

	query := `SELECT id,
			 description,
			 filename,
			 fw_type,
			 vm_type,
			 image_id,
			 image_name
		  FROM workload_template
		  WHERE id = ?`

	work := new(workload)

	var VMType string

	err := datastore.QueryRow(query, id).Scan(&work.ID, &work.Description, &work.filename, &work.FWType, &VMType, &work.ImageID, &work.ImageName)
	switch {
	case err == sql.ErrNoRows:
		return nil, fmt.Errorf("Workload %q not found", id)
	case err != nil:
		return nil, err
	}

	work.VMType = payloads.Hypervisor(VMType)

	work.Config, err = ds.getConfig(id)
	if err != nil {
		return nil, err
	}

	work.Defaults, err = ds.getWorkloadDefaults(id)
	if err != nil {
		return nil, err
	}

	work.Storage, err = ds.getWorkloadStorage(id)
	if err != nil {
		return nil, err
	}

	return work, nil
}

func (ds *sqliteDB) getWorkloads() ([]*workload, error) {
	var workloads []*workload

	datastore := ds.db

	query := `SELECT id,
			 description,
			 filename,
			 fw_type,
			 vm_type,
			 image_id,
			 image_name
		  FROM workload_template
		  WHERE internal = 0`

	rows, err := datastore.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		wl := new(workload)

		var VMType string

		err = rows.Scan(&wl.ID, &wl.Description, &wl.filename, &wl.FWType, &VMType, &wl.ImageID, &wl.ImageName)
		if err != nil {
			return nil, err
		}

		wl.Config, err = ds.getConfig(wl.ID)
		if err != nil {
			return nil, err
		}

		wl.Defaults, err = ds.getWorkloadDefaults(wl.ID)
		if err != nil {
			return nil, err
		}

		wl.Storage, err = ds.getWorkloadStorage(wl.ID)
		if err != nil {
			return nil, err
		}

		wl.VMType = payloads.Hypervisor(VMType)

		workloads = append(workloads, wl)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return workloads, nil
}

func (ds *sqliteDB) updateWorkload(w workload) error {
	db := ds.getTableDB("workload_template")

	workloads, err := ds.getWorkloads()
	if err != nil {
		return err
	}

	m := make(map[string]bool)
	for _, work := range workloads {
		m[work.ID] = true
	}

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	resources, err := ds.getResources(tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	// if this is a new workload, put it in, otherwise just update.
	_, ok := m[w.ID]
	if !ok {
		// add in workload resources
		for _, d := range w.Defaults {
			err := ds.createWorkloadDefault(tx, w.ID, resources[string(d.Type)], d)
			if err != nil {
				tx.Rollback()
				return err
			}
		}

		// add in any workload storage resources
		if len(w.Storage) > 0 {
			for i := range w.Storage {
				err := ds.createWorkloadStorage(tx, w.ID, &w.Storage[i])
				if err != nil {
					tx.Rollback()
					return err
				}
			}
		}

		// write config to file.
		path := fmt.Sprintf("%s/%s", ds.workloadsPath, w.filename)
		err := ioutil.WriteFile(path, []byte(w.Config), 0644)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec("INSERT INTO workload_template (id, description, filename, fw_type, vm_type, image_id, image_name, internal) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", w.ID, w.Description, w.filename, w.FWType, string(w.VMType), w.ImageID, w.ImageName, false)
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		// update not supported yet.
		tx.Rollback()
		return errors.New("Workload Update not supported yet")
	}

	tx.Commit()
	return nil
}

func (ds *sqliteDB) updateTenant(t *tenant) error {
	db := ds.getTableDB("tenants")

	ds.dbLock.Lock()

	tx, err := db.Begin()
	if err != nil {
		ds.dbLock.Unlock()
		return err
	}

	_, err = tx.Exec("UPDATE tenants SET cnci_id = ?, cnci_mac = ?, cnci_ip = ? WHERE id = ?", t.CNCIID, t.CNCIMAC, t.CNCIIP, t.ID)
	if err != nil {
		tx.Rollback()
		ds.dbLock.Unlock()
		return err
	}

	tx.Commit()

	ds.dbLock.Unlock()

	return err
}

func (ds *sqliteDB) getTenants() ([]*tenant, error) {
	var tenants []*tenant

	datastore := ds.getTableDB("tenants")

	query := `SELECT	tenants.id,
				tenants.name,
				tenants.cnci_id,
				tenants.cnci_mac,
				tenants.cnci_ip
		  FROM tenants `

	rows, err := datastore.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id sql.NullString
		var name sql.NullString
		var cnciID sql.NullString
		var cnciMAC sql.NullString
		var cnciIP sql.NullString

		t := new(tenant)
		err = rows.Scan(&id, &name, &cnciID, &cnciMAC, &cnciIP)
		if err != nil {
			return nil, err
		}

		if id.Valid {
			t.ID = id.String
		}

		if name.Valid {
			t.Name = name.String
		}

		if cnciID.Valid {
			t.CNCIID = cnciID.String
		}

		if cnciMAC.Valid {
			t.CNCIMAC = cnciMAC.String
		}

		if cnciIP.Valid {
			t.CNCIIP = cnciIP.String
		}

		t.Resources, err = ds.getTenantResources(t.ID)
		if err != nil {
			return nil, err
		}

		err = ds.getTenantNetwork(t)
		if err != nil {
			return nil, err
		}

		t.instances, err = ds.getTenantInstances(t.ID)
		if err != nil {
			return nil, err
		}

		t.devices, err = ds.getTenantDevices(t.ID)
		if err != nil {
			return nil, err
		}

		tenants = append(tenants, t)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tenants, nil
}

func (ds *sqliteDB) claimTenantIP(tenantID string, subnetInt int, rest int) error {
	datastore := ds.getTableDB("tenant_network")
	ds.dbLock.Lock()
	tx, err := datastore.Begin()
	if err != nil {
		ds.dbLock.Unlock()
		return err
	}

	_, err = tx.Exec("INSERT INTO tenant_network VALUES(?, ?, ?)", tenantID, subnetInt, rest)
	if err != nil {
		tx.Rollback()
		ds.dbLock.Unlock()
		return err
	}

	tx.Commit()
	ds.dbLock.Unlock()

	return nil
}

func (ds *sqliteDB) releaseTenantIP(tenantID string, subnetInt int, rest int) error {
	datastore := ds.getTableDB("tenant_network")

	ds.dbLock.Lock()
	tx, err := datastore.Begin()
	if err != nil {
		ds.dbLock.Unlock()
		return err
	}

	_, err = tx.Exec("DELETE FROM tenant_network WHERE tenant_id = ? AND subnet = ? AND rest = ?", tenantID, subnetInt, rest)
	if err != nil {
		tx.Rollback()
		ds.dbLock.Unlock()
		return err
	}

	tx.Commit()
	ds.dbLock.Unlock()

	return err
}

func (ds *sqliteDB) getTenantNetwork(tenant *tenant) error {
	tenant.network = make(map[int]map[int]bool)

	ds.dbLock.Lock()

	datastore := ds.getTableDB("tenant_network")

	tx, err := datastore.Begin()
	if err != nil {
		ds.dbLock.Unlock()
		return err
	}

	// get all subnet,rest values for this tenant
	query := `SELECT subnet, rest
		  FROM tenant_network
		  WHERE tenant_id = ?`

	rows, err := tx.Query(query, tenant.ID)
	if err != nil {
		glog.Warning(err)
		tx.Rollback()
		ds.dbLock.Unlock()
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var subnetInt uint16
		var rest uint8

		err = rows.Scan(&subnetInt, &rest)
		if err != nil {
			glog.Warning(err)
			tx.Rollback()
			ds.dbLock.Unlock()
			return err
		}

		_, ok := tenant.network[int(subnetInt)]
		if !ok {
			sub := make(map[int]bool)
			tenant.network[int(subnetInt)] = sub
		}

		/* Only add to the subnet list for the first host */
		if len(tenant.network[int(subnetInt)]) == 0 {
			tenant.subnets = append(tenant.subnets, int(subnetInt))
		}

		tenant.network[int(subnetInt)][int(rest)] = true
	}

	tx.Commit()

	ds.dbLock.Unlock()

	return err
}

func (ds *sqliteDB) getInstances() ([]*types.Instance, error) {
	var instances []*types.Instance

	datastore := ds.getTableDB("instances")

	ds.tdbLock.RLock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.tdbLock.RUnlock()
		return nil, err
	}

	query := `
	WITH latest AS
	(
		SELECT 	max(tdb.instance_statistics.timestamp),
			tdb.instance_statistics.instance_id,
			tdb.instance_statistics.state,
			tdb.instance_statistics.ssh_ip,
			tdb.instance_statistics.ssh_port,
			tdb.instance_statistics.node_id
		FROM tdb.instance_statistics
		GROUP BY tdb.instance_statistics.instance_id
	)
	SELECT	instances.id,
		instances.tenant_id,
		IFNULL(latest.state, "` + payloads.ComputeStatusPending + `") AS state,
		workload_id,
		IFNULL(latest.ssh_ip, "Not Assigned") as ssh_ip,
		latest.ssh_port as ssh_port,
		IFNULL(latest.node_id, "Not Assigned") as node_id,
		mac_address,
		ip
	FROM instances
	LEFT JOIN latest
	ON instances.id = latest.instance_id
	`

	rows, err := tx.Query(query)
	if err != nil {
		tx.Rollback()
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var i types.Instance

		var sshPort sql.NullInt64

		err = rows.Scan(&i.ID, &i.TenantID, &i.State, &i.WorkloadID, &i.SSHIP, &sshPort, &i.NodeID, &i.MACAddress, &i.IPAddress)
		if err != nil {
			tx.Rollback()
			ds.tdbLock.RUnlock()
			return nil, err
		}

		if sshPort.Valid {
			i.SSHPort = int(sshPort.Int64)
		}

		defaults, err := ds.getWorkloadDefaults(i.WorkloadID)
		if err != nil {
			tx.Rollback()
			ds.tdbLock.RUnlock()
			return nil, err
		}

		// convert RequestedResources into a map[string]int
		// TBD: shouldn't we just change getWorkloadDefaults
		// to return a map?
		usage := make(map[string]int)
		for c := range defaults {
			usage[string(defaults[c].Type)] = defaults[c].Value
		}
		i.Usage = usage

		instances = append(instances, &i)
	}

	if err = rows.Err(); err != nil {
		tx.Rollback()
		ds.tdbLock.RUnlock()
		return nil, err
	}

	tx.Commit()

	ds.tdbLock.RUnlock()

	return instances, nil
}

func (ds *sqliteDB) getTenantInstances(tenantID string) (map[string]*types.Instance, error) {
	datastore := ds.getTableDB("instances")

	ds.tdbLock.RLock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.tdbLock.RUnlock()
		return nil, err
	}

	query := `
	WITH latest AS
	(
		SELECT 	max(tdb.instance_statistics.timestamp),
			tdb.instance_statistics.instance_id,
			tdb.instance_statistics.state,
			tdb.instance_statistics.ssh_ip,
			tdb.instance_statistics.ssh_port,
			tdb.instance_statistics.node_id
		FROM tdb.instance_statistics
		GROUP BY tdb.instance_statistics.instance_id
	)
	SELECT	instances.id,
		instances.tenant_id,
		IFNULL(latest.state, "` + payloads.ComputeStatusPending + `") AS state,
		IFNULL(latest.ssh_ip, "Not Assigned") AS ssh_ip,
		latest.ssh_port AS ssh_port,
		workload_id,
		latest.node_id,
		mac_address,
		ip
	FROM instances
	LEFT JOIN latest
	ON instances.id = latest.instance_id
	WHERE instances.tenant_id = ?
	`

	rows, err := tx.Query(query, tenantID)
	if err != nil {
		tx.Rollback()
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

	instances := make(map[string]*types.Instance)
	for rows.Next() {
		var nodeID sql.NullString
		var sshIP sql.NullString
		var sshPort sql.NullInt64

		i := &types.Instance{}

		err = rows.Scan(&i.ID, &i.TenantID, &i.State, &sshIP, &sshPort, &i.WorkloadID, &nodeID, &i.MACAddress, &i.IPAddress)
		if err != nil {
			tx.Rollback()
			ds.tdbLock.RUnlock()
			return nil, err
		}

		if nodeID.Valid {
			i.NodeID = nodeID.String
		}

		if sshIP.Valid {
			i.SSHIP = sshIP.String
		}

		if sshPort.Valid {
			i.SSHPort = int(sshPort.Int64)
		}

		defaults, err := ds.getWorkloadDefaults(i.WorkloadID)
		if err != nil {
			return nil, err
		}

		// convert RequestedResources into a map[string]int
		// TBD: shouldn't we just change getWorkloadDefaults
		// to return a map?
		usage := make(map[string]int)
		for c := range defaults {
			usage[string(defaults[c].Type)] = defaults[c].Value
		}
		i.Usage = usage

		instances[i.ID] = i
	}

	if err = rows.Err(); err != nil {
		tx.Rollback()
		ds.tdbLock.RUnlock()
		return nil, err
	}

	tx.Commit()

	ds.tdbLock.RUnlock()

	return instances, nil
}

func (ds *sqliteDB) addInstance(instance *types.Instance) error {
	ds.dbLock.Lock()

	err := ds.create("instances", instance.ID, instance.TenantID, instance.WorkloadID, instance.MACAddress, instance.IPAddress, instance.CreateTime.Format(time.RFC3339Nano))

	ds.dbLock.Unlock()

	if err != nil {
		return err
	}

	return ds.addUsage(instance.ID, instance.Usage)
}

func (ds *sqliteDB) deleteInstance(instanceID string) error {
	datastore := ds.getTableDB("instances")

	ds.dbLock.Lock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.tdbLock.Unlock()
		return err
	}

	_, err = tx.Exec("DELETE FROM instances WHERE id = ?", instanceID)
	if err != nil {
		tx.Rollback()
		ds.dbLock.Unlock()
		return err
	}

	_, err = tx.Exec("DELETE FROM usage WHERE instance_id = ?", instanceID)
	if err != nil {
		tx.Rollback()
		ds.dbLock.Unlock()
		return err
	}

	tx.Commit()

	ds.dbLock.Unlock()

	return err
}

func (ds *sqliteDB) addUsage(instanceID string, usage map[string]int) error {
	datastore := ds.getTableDB("usage")

	ds.dbLock.Lock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.dbLock.Unlock()
		return err
	}

	cmd := `INSERT INTO usage (instance_id, resource_id, value)
		SELECT ?, resources.id, ?
		FROM resources
		WHERE name = ?`

	stmt, err := tx.Prepare(cmd)
	if err != nil {
		tx.Rollback()
		ds.tdbLock.Unlock()
		return err
	}

	defer stmt.Close()

	for key, val := range usage {
		_, err := stmt.Exec(instanceID, val, key)

		if err != nil {
			glog.V(2).Info(err)
			// but keep going
		}
	}

	tx.Commit()

	ds.dbLock.Unlock()

	return nil
}

func (ds *sqliteDB) addNodeStat(stat payloads.Stat) error {
	datastore := ds.getTableDB("node_statistics")

	ds.tdbLock.Lock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.tdbLock.Unlock()
		return err
	}

	_, err = tx.Exec("INSERT INTO node_statistics (node_id, mem_total_mb, mem_available_mb, disk_total_mb, disk_available_mb, load, cpus_online) VALUES(?, ?, ?, ?, ?, ?, ?)", stat.NodeUUID, stat.MemTotalMB, stat.MemAvailableMB, stat.DiskTotalMB, stat.DiskAvailableMB, stat.Load, stat.CpusOnline)
	if err != nil {
		tx.Rollback()
		ds.tdbLock.Unlock()
		return err
	}

	tx.Commit()

	ds.tdbLock.Unlock()

	return err
}

func (ds *sqliteDB) addInstanceStats(stats []payloads.InstanceStat, nodeID string) error {
	datastore := ds.getTableDB("instance_statistics")

	ds.tdbLock.Lock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.tdbLock.Unlock()
		return err
	}

	cmd := `INSERT INTO instance_statistics (instance_id, memory_usage_mb, disk_usage_mb, cpu_usage, state, node_id, ssh_ip, ssh_port)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(cmd)
	if err != nil {
		tx.Rollback()
		ds.tdbLock.Unlock()
		return err
	}

	defer stmt.Close()

	for index := range stats {
		stat := stats[index]

		_, err = stmt.Exec(stat.InstanceUUID, stat.MemoryUsageMB, stat.DiskUsageMB, stat.CPUUsage, stat.State, nodeID, stat.SSHIP, stat.SSHPort)
		if err != nil {
			glog.Warning(err)
			// but keep going
		}
	}

	tx.Commit()

	ds.tdbLock.Unlock()

	return err
}

func (ds *sqliteDB) addFrameStat(stat payloads.FrameTrace) error {
	datastore := ds.getTableDB("frame_statistics")

	ds.tdbLock.Lock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.tdbLock.Unlock()
		return err
	}

	query := `INSERT INTO frame_statistics (label, type, operand, start_timestamp, end_timestamp)
		  VALUES(?, ?, ?, ?, ?)`

	_, err = tx.Exec(query, stat.Label, stat.Type, stat.Operand, stat.StartTimestamp, stat.EndTimestamp)
	if err != nil {
		tx.Rollback()
		ds.tdbLock.Unlock()
		return err
	}

	var id int

	err = tx.QueryRow("SELECT last_insert_rowid();").Scan(&id)
	if err != nil {
		tx.Rollback()
		ds.tdbLock.Unlock()
		return err
	}

	for index := range stat.Nodes {
		t := stat.Nodes[index]

		cmd := `INSERT INTO trace_data (frame_id, ssntp_uuid, tx_timestamp, rx_timestamp)
			VALUES(?, ?, ?, ?);`

		_, err = tx.Exec(cmd, id, t.SSNTPUUID, t.TxTimestamp, t.RxTimestamp)
		if err != nil {
			tx.Rollback()
			ds.tdbLock.Unlock()
			return err
		}
	}

	tx.Commit()

	ds.tdbLock.Unlock()

	return err
}

// GetEventLog retrieves all the log entries stored in the datastore.
func (ds *sqliteDB) getEventLog() ([]*types.LogEntry, error) {
	var logEntries []*types.LogEntry

	datastore := ds.getTableDB("log")

	ds.tdbLock.RLock()

	rows, err := datastore.Query("SELECT timestamp, tenant_id, type, message FROM log")
	if err != nil {
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

	logEntries = make([]*types.LogEntry, 0)
	for rows.Next() {
		var e types.LogEntry
		err = rows.Scan(&e.Timestamp, &e.TenantID, &e.EventType, &e.Message)
		if err != nil {
			ds.tdbLock.RUnlock()
			return nil, err
		}
		logEntries = append(logEntries, &e)
	}

	ds.tdbLock.RUnlock()

	return logEntries, err
}

// GetNodeSummary provides a summary the state and count of instances running per node.
func (ds *sqliteDB) getNodeSummary() ([]*types.NodeSummary, error) {
	var summary []*types.NodeSummary

	datastore := ds.getTableDB("instance_statistics")

	ds.tdbLock.RLock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.tdbLock.RUnlock()
		return nil, err
	}

	query := `
WITH instances AS
(
	WITH latest AS
	(
		SELECT 	max(timestamp),
			instance_id,
			node_id,
			state
		FROM instance_statistics
		GROUP BY instance_id
	)
	SELECT db.instances.id AS instance_id,
	       IFNULL(latest.state, "` + payloads.ComputeStatusPending + `") AS state,
	       IFNULL(latest.node_id, "Not Assigned") AS node_id
	FROM db.instances
	LEFT JOIN latest
	ON db.instances.id = latest.instance_id
),
total_instances AS
(
	SELECT 	IFNULL(instances.node_id, "Not Assigned to Node") AS node_id,
		count(instances.instance_id) AS total
	FROM instances
	GROUP BY node_id
),
total_running AS
(
	SELECT	instances.node_id AS node_id,
		count(instances.instance_id) AS total
	FROM instances
	WHERE state='` + payloads.ComputeStatusRunning + `'
	GROUP BY node_id
),
total_pending AS
(
	SELECT	instances.node_id AS node_id,
		count(instances.instance_id) AS total
	FROM instances
	WHERE state='` + payloads.ComputeStatusPending + `'
	GROUP BY node_id
),
total_exited AS
(
	SELECT	instances.node_id,
		count(instances.instance_id) AS total
	FROM instances
	WHERE state='` + payloads.ComputeStatusStopped + `'
	GROUP BY node_id
)
SELECT	total_instances.node_id,
	total_instances.total,
        IFNULL(total_running.total, 0),
	IFNULL(total_pending.total, 0),
	IFNULL(total_exited.total, 0)
FROM total_instances
LEFT JOIN total_running
ON total_instances.node_id = total_running.node_id
LEFT JOIN total_pending
ON total_instances.node_id = total_pending.node_id
LEFT JOIN total_exited
ON total_instances.node_id = total_exited.node_id
`

	rows, err := tx.Query(query)
	if err != nil {
		glog.V(2).Info("Failed to get Node Summary: ", err)
		tx.Rollback()
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

	summary = make([]*types.NodeSummary, 0)

	for rows.Next() {
		var n types.NodeSummary

		err = rows.Scan(&n.NodeID, &n.TotalInstances, &n.TotalRunningInstances, &n.TotalPendingInstances, &n.TotalPausedInstances)
		if err != nil {
			tx.Rollback()
			ds.tdbLock.RUnlock()
			return nil, err
		}

		summary = append(summary, &n)
	}

	tx.Commit()

	ds.tdbLock.RUnlock()

	return summary, err
}

// GetBatchFrameSummary will retieve the count of traces we have for a specific label
func (ds *sqliteDB) getBatchFrameSummary() ([]types.BatchFrameSummary, error) {
	var stats []types.BatchFrameSummary

	datastore := ds.getTableDB("frame_statistics")

	ds.tdbLock.RLock()

	query := `SELECT label, count(id)
		  FROM frame_statistics
		  GROUP BY label;`

	rows, err := datastore.Query(query)
	if err != nil {
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

	stats = make([]types.BatchFrameSummary, 0)

	for rows.Next() {
		var stat types.BatchFrameSummary

		err = rows.Scan(&stat.BatchID, &stat.NumInstances)
		if err != nil {
			ds.tdbLock.RUnlock()
			return nil, err
		}

		stats = append(stats, stat)
	}

	ds.tdbLock.RUnlock()

	return stats, err
}

// GetBatchFrameStatistics will show individual trace data per instance for a batch of trace data.
// The batch is identified by the label.
func (ds *sqliteDB) getBatchFrameStatistics(label string) ([]types.BatchFrameStat, error) {
	var stats []types.BatchFrameStat

	datastore := ds.getTableDB("frame_statistics")

	query := `WITH total AS
		 (
			SELECT	id,
				start_timestamp,
				end_timestamp,
				(julianday(end_timestamp) - julianday(start_timestamp)) * 24 * 60 * 60 AS total_elapsed
			FROM frame_statistics
			WHERE label = ?
		),
		total_start AS
		(
			SELECT	trace_data.frame_id,
				trace_data.ssntp_uuid,
				(julianday(trace_data.tx_timestamp) - julianday(total.start_timestamp)) * 24 * 60 * 60 AS total_elapsed
			FROM trace_data
			JOIN total
			WHERE rx_timestamp = '' and trace_data.frame_id = total.id
		),
		total_end AS
		(
			SELECT 	trace_data.frame_id,
				trace_data.ssntp_uuid,
				(julianday(total.end_timestamp) - julianday(trace_data.rx_timestamp)) * 24 * 60 * 60 AS total_elapsed
			FROM trace_data
			JOIN total
			WHERE tx_timestamp = '' and trace_data.frame_id = total.id
		),
		total_per_node AS
		(
			SELECT	trace_data.frame_id,
				trace_data.ssntp_uuid,
				(julianday(trace_data.tx_timestamp) - julianday(trace_data.rx_timestamp)) * 24 * 60 *60 AS total_elapsed
			FROM trace_data
			WHERE tx_timestamp != '' and rx_timestamp != ''
		),
		diffs AS
		(
			SELECT 	total.id AS id,
				total.total_elapsed AS total_elapsed,
				total_start.total_elapsed AS controller_elapsed,
				total_end.total_elapsed AS launcher_elapsed,
				total_per_node.total_elapsed AS scheduler_elapsed
			FROM total
			LEFT JOIN total_start
			ON total.id = total_start.frame_id
			LEFT JOIN total_end
			ON total_start.frame_id = total_end.frame_id
			LEFT JOIN total_per_node
			ON total_start.frame_id = total_per_node.frame_id
		),
		averages AS
		(
			SELECT	avg(diffs.total_elapsed) AS avg_total_elapsed,
				avg(diffs.controller_elapsed) AS avg_controller,
				avg(diffs.launcher_elapsed) AS avg_launcher,
				avg(diffs.scheduler_elapsed) AS avg_scheduler
			FROM diffs
		),
		variance AS
		(
			SELECT	avg((total_start.total_elapsed - averages.avg_controller) * (total_start.total_elapsed - averages.avg_controller)) AS controller,
				avg((total_end.total_elapsed - averages.avg_launcher) * (total_end.total_elapsed - averages.avg_launcher)) AS launcher,
				avg((total_per_node.total_elapsed - averages.avg_scheduler) * (total_per_node.total_elapsed - averages.avg_scheduler)) AS scheduler
			FROM total_start
			LEFT JOIN total_end
			ON total_start.frame_id = total_end.frame_id
			LEFT JOIN total_per_node
			ON total_start.frame_id = total_per_node.frame_id
			JOIN averages
		)
		SELECT	count(total.id) AS num_instances,
			(julianday(max(total.end_timestamp)) - julianday(min(total.start_timestamp))) * 24 * 60 * 60 AS total_elapsed,
			averages.avg_total_elapsed AS average_total_elapsed,
			averages.avg_controller AS average_controller_elapsed,
			averages.avg_launcher AS average_launcher_elapsed,
			averages.avg_scheduler AS average_scheduler_elapsed,
			variance.controller AS controller_variance,
			variance.launcher AS launcher_variance,
			variance.scheduler AS scheduler_variance
		FROM variance
		JOIN total
		JOIN averages;`

	ds.tdbLock.RLock()

	rows, err := datastore.Query(query, label)
	if err != nil {
		ds.tdbLock.RUnlock()
		return nil, err
	}
	defer rows.Close()

	stats = make([]types.BatchFrameStat, 0)

	for rows.Next() {
		var stat types.BatchFrameStat
		var numInstances sql.NullInt64
		var totalElapsed sql.NullFloat64
		var averageElapsed sql.NullFloat64
		var averageControllerElapsed sql.NullFloat64
		var averageLauncherElapsed sql.NullFloat64
		var averageSchedulerElapsed sql.NullFloat64
		var varianceController sql.NullFloat64
		var varianceLauncher sql.NullFloat64
		var varianceScheduler sql.NullFloat64

		err = rows.Scan(&numInstances, &totalElapsed, &averageElapsed, &averageControllerElapsed, &averageLauncherElapsed, &averageSchedulerElapsed, &varianceController, &varianceLauncher, &varianceScheduler)
		if err != nil {
			ds.tdbLock.RUnlock()
			return nil, err
		}

		if numInstances.Valid {
			stat.NumInstances = int(numInstances.Int64)
		}

		if totalElapsed.Valid {
			stat.TotalElapsed = totalElapsed.Float64
		}

		if averageElapsed.Valid {
			stat.AverageElapsed = averageElapsed.Float64
		}

		if averageControllerElapsed.Valid {
			stat.AverageControllerElapsed = averageControllerElapsed.Float64
		}

		if averageLauncherElapsed.Valid {
			stat.AverageLauncherElapsed = averageLauncherElapsed.Float64
		}

		if averageSchedulerElapsed.Valid {
			stat.AverageSchedulerElapsed = averageSchedulerElapsed.Float64
		}

		if varianceController.Valid {
			stat.VarianceController = varianceController.Float64
		}

		if varianceLauncher.Valid {
			stat.VarianceLauncher = varianceLauncher.Float64
		}

		if varianceScheduler.Valid {
			stat.VarianceScheduler = varianceScheduler.Float64
		}

		stats = append(stats, stat)
	}

	ds.tdbLock.RUnlock()

	return stats, err
}

func (ds *sqliteDB) getTenantDevices(tenantID string) (map[string]types.BlockData, error) {
	devices := make(map[string]types.BlockData)

	datastore := ds.getTableDB("block_data")

	ds.dbLock.Lock()

	query := `SELECT	block_data.id,
				block_data.tenant_id,
				block_data.size,
				block_data.state,
				block_data.create_time,
				block_data.name,
				block_data.description
		  FROM	block_data
		  WHERE block_data.tenant_id = ?`

	rows, err := datastore.Query(query, tenantID)
	if err != nil {
		ds.dbLock.Unlock()
		return devices, err
	}
	defer rows.Close()

	for rows.Next() {
		var state string
		var data types.BlockData

		err = rows.Scan(&data.ID, &data.TenantID, &data.Size, &state, &data.CreateTime, &data.Name, &data.Description)
		if err != nil {
			continue
		}

		data.State = types.BlockState(state)
		devices[data.ID] = data
	}
	if err = rows.Err(); err != nil {
		ds.dbLock.Unlock()
		return devices, err
	}

	ds.dbLock.Unlock()

	return devices, nil
}

func (ds *sqliteDB) getAllBlockData() (map[string]types.BlockData, error) {
	devices := make(map[string]types.BlockData)

	datastore := ds.getTableDB("block_data")

	query := `SELECT	block_data.id,
				block_data.tenant_id,
				block_data.size,
				block_data.state,
				block_data.create_time,
				block_data.name,
				block_data.description
		  FROM	block_data `

	rows, err := datastore.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var data types.BlockData
		var state string

		err = rows.Scan(&data.ID, &data.TenantID, &data.Size, &state, &data.CreateTime, &data.Name, &data.Description)
		if err != nil {
			continue
		}

		data.State = types.BlockState(state)
		devices[data.ID] = data
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return devices, nil
}

func (ds *sqliteDB) addBlockData(data types.BlockData) error {
	ds.dbLock.Lock()
	err := ds.create("block_data", data.ID, data.TenantID, data.Size, string(data.State), data.CreateTime.Format(time.RFC3339Nano), data.Name, data.Description)
	ds.dbLock.Unlock()

	return err
}

// For now we only support updating the state.
func (ds *sqliteDB) updateBlockData(data types.BlockData) error {
	db := ds.getTableDB("block_data")

	ds.dbLock.Lock()

	tx, err := db.Begin()
	if err != nil {
		ds.dbLock.Unlock()
		return err
	}

	_, err = tx.Exec("UPDATE block_data SET state = ? WHERE id = ?", string(data.State), data.ID)
	if err != nil {
		tx.Rollback()
		ds.dbLock.Unlock()
		return err
	}

	tx.Commit()

	ds.dbLock.Unlock()

	return err
}

func (ds *sqliteDB) deleteBlockData(ID string) error {
	datastore := ds.getTableDB("block_data")

	ds.dbLock.Lock()
	tx, err := datastore.Begin()
	if err != nil {
		ds.dbLock.Unlock()
		return err
	}

	_, err = tx.Exec("DELETE FROM block_data WHERE id = ?", ID)
	if err != nil {
		tx.Rollback()
		ds.dbLock.Unlock()
		return err
	}

	tx.Commit()
	ds.dbLock.Unlock()

	return err
}

func (ds *sqliteDB) addStorageAttachment(a types.StorageAttachment) error {
	datastore := ds.getTableDB("attachments")

	ds.dbLock.Lock()

	tx, err := datastore.Begin()
	if err != nil {
		ds.dbLock.Unlock()
		return err
	}

	_, err = tx.Exec("INSERT INTO attachments (id, instance_id, block_id, ephemeral, boot) VALUES (?, ?, ?, ?, ?)", a.ID, a.InstanceID, a.BlockID, a.Ephemeral, a.Boot)
	if err != nil {
		tx.Rollback()
		ds.dbLock.Unlock()
		return err
	}

	tx.Commit()

	ds.dbLock.Unlock()
	return err
}

func (ds *sqliteDB) getAllStorageAttachments() (map[string]types.StorageAttachment, error) {
	attachments := make(map[string]types.StorageAttachment)

	datastore := ds.getTableDB("attachments")

	query := `SELECT	attachments.id,
				attachments.instance_id,
				attachments.block_id,
				attachments.ephemeral,
				attachments.boot
		  FROM	attachments `

	rows, err := datastore.Query(query)
	if err != nil {
		return attachments, err
	}
	defer rows.Close()

	for rows.Next() {
		var a types.StorageAttachment

		err = rows.Scan(&a.ID, &a.InstanceID, &a.BlockID, &a.Ephemeral, &a.Boot)
		if err != nil {
			continue
		}
		attachments[a.ID] = a
	}

	if err = rows.Err(); err != nil {
		return attachments, err
	}

	return attachments, nil
}

func (ds *sqliteDB) deleteStorageAttachment(ID string) error {
	datastore := ds.getTableDB("attachments")

	ds.dbLock.Lock()
	tx, err := datastore.Begin()
	if err != nil {
		ds.dbLock.Unlock()
		return err
	}

	_, err = tx.Exec("DELETE FROM attachments WHERE id = ?", ID)
	if err != nil {
		tx.Rollback()
		ds.dbLock.Unlock()
		return err
	}

	tx.Commit()
	ds.dbLock.Unlock()

	return err
}

// this is here just for readability.
func (ds *sqliteDB) addPool(pool types.Pool) error {
	return ds.updatePool(pool)
}

// lock must be held by caller. Any rollbacks will need to be handled
// by caller.
func (ds *sqliteDB) updateSubnets(tx *sql.Tx, pool types.Pool) error {
	// get currently known subnets.
	subnets, err := ds.getPoolSubnets(pool.ID)
	if err != nil {
		// TBD: what about row not found?
		return err
	}

	// make a map of pool subnets by ID
	subMap := make(map[string]bool)
	for _, sub := range pool.Subnets {
		subMap[sub.ID] = true
	}

	// do we have any subnets that need deleting?
	for _, sub := range subnets {
		_, ok := subMap[sub.ID]
		if !ok {
			_, err = tx.Exec("DELETE FROM subnet_pool WHERE id = ?", sub.ID)
			if err != nil {
				return err
			}
		}
	}

	// any subnets that already exist in the table will be ignored,
	// new ones will be added.
	for _, subnet := range pool.Subnets {
		_, err = tx.Exec("INSERT OR IGNORE INTO subnet_pool (id, pool_id, cidr) VALUES (?, ?, ?)", subnet.ID, pool.ID, subnet.CIDR)
		if err != nil {
			return err
		}
	}

	return nil
}

// lock must be held by caller. Any rollbacks will need to be handled
// by caller.
func (ds *sqliteDB) updateAddresses(tx *sql.Tx, pool types.Pool) error {
	// get currently known individual addresses.
	addresses, err := ds.getPoolAddresses(pool.ID)
	if err != nil {
		// TBD: what about row not found?
		return err
	}

	// make a map of pool addresses by ID
	addrMap := make(map[string]bool)
	for _, addr := range pool.IPs {
		addrMap[addr.ID] = true
	}

	// do we have any individual IPs that need deleting?
	for _, addr := range addresses {
		_, ok := addrMap[addr.ID]
		if !ok {
			_, err = tx.Exec("DELETE FROM address_pool WHERE id = ?", addr.ID)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// any addresses that already exist in the table will be ignored,
	// new ones will be added.
	for _, IP := range pool.IPs {
		_, err = tx.Exec("INSERT OR IGNORE INTO address_pool (id, pool_id, address) VALUES (?, ?, ?)", IP.ID, pool.ID, IP.Address)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return nil
}

// updatePool is used to update all pool related fields even if they
// are in different tables.
func (ds *sqliteDB) updatePool(pool types.Pool) error {
	datastore := ds.getTableDB("pools")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	pools := ds.getAllPools()

	// do the below as a single transaction.
	tx, err := datastore.Begin()
	if err != nil {
		return err
	}

	err = ds.updateSubnets(tx, pool)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = ds.updateAddresses(tx, pool)
	if err != nil {
		tx.Rollback()
		return err
	}

	// if this is a new pool, put it in, otherwise just update.
	_, ok := pools[pool.ID]
	if !ok {
		_, err = tx.Exec("INSERT INTO pools (id, name, free, total) VALUES (?, ?, ?, ?)", pool.ID, pool.Name, pool.Free, pool.TotalIPs)
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		// update free and total counts.
		_, err = tx.Exec("UPDATE pools SET free = ?, total = ? WHERE id = ?", pool.Free, pool.TotalIPs, pool.ID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()

	return nil
}

func (ds *sqliteDB) getAllPools() map[string]types.Pool {
	pools := make(map[string]types.Pool)

	datastore := ds.getTableDB("pools")

	query := `SELECT	id,
				name,
				free,
				total
		  FROM	pools`

	rows, err := datastore.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var pool types.Pool

		err = rows.Scan(&pool.ID, &pool.Name, &pool.Free, &pool.TotalIPs)
		if err != nil {
			continue
		}

		pool.Subnets, err = ds.getPoolSubnets(pool.ID)
		if err != nil {
			continue
		}

		pool.IPs, err = ds.getPoolAddresses(pool.ID)
		if err != nil {
			continue
		}

		pools[pool.ID] = pool
	}

	if err = rows.Err(); err != nil {
		return nil
	}

	return pools
}

func (ds *sqliteDB) deletePool(ID string) error {
	datastore := ds.getTableDB("pools")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	tx, err := datastore.Begin()
	if err != nil {
		return err
	}

	// lock is held here and ok because the
	// get functions don't hold a lock.
	subnets, err := ds.getPoolSubnets(ID)
	if err != nil {
		return err
	}

	IPs, err := ds.getPoolAddresses(ID)
	if err != nil {
		return err
	}

	for _, subnet := range subnets {
		_, err = tx.Exec("DELETE FROM subnet_pool WHERE id = ?", subnet.ID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, addr := range IPs {
		_, err = tx.Exec("DELETE FROM address_pool WHERE id = ?", addr.ID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = tx.Exec("DELETE FROM pools WHERE id = ?", ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	return err
}

func (ds *sqliteDB) getPoolSubnets(poolID string) ([]types.ExternalSubnet, error) {
	var subnets []types.ExternalSubnet

	datastore := ds.getTableDB("subnet_pool")

	query := `SELECT	id,
				cidr
		  FROM	subnet_pool
		  WHERE pool_id = ?`

	rows, err := datastore.Query(query, poolID)
	if err != nil {
		return subnets, err
	}
	defer rows.Close()

	for rows.Next() {
		var subnet types.ExternalSubnet

		err = rows.Scan(&subnet.ID, &subnet.CIDR)
		if err != nil {
			continue
		}

		subnets = append(subnets, subnet)
	}

	if err = rows.Err(); err != nil {
		return subnets, err
	}

	return subnets, nil
}

func (ds *sqliteDB) getPoolAddresses(poolID string) ([]types.ExternalIP, error) {
	var IPs []types.ExternalIP

	datastore := ds.getTableDB("address_pool")

	query := `SELECT	id,
				address
		  FROM	address_pool
		  WHERE pool_id = ?`

	rows, err := datastore.Query(query, poolID)
	if err != nil {
		return IPs, err
	}
	defer rows.Close()

	for rows.Next() {
		var IP types.ExternalIP

		err = rows.Scan(&IP.ID, &IP.Address)
		if err != nil {
			continue
		}

		IPs = append(IPs, IP)
	}

	if err = rows.Err(); err != nil {
		return IPs, err
	}

	return IPs, nil
}

func (ds *sqliteDB) addMappedIP(m types.MappedIP) error {
	datastore := ds.getTableDB("mapped_ips")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	tx, err := datastore.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO mapped_ips (id, pool_id, external_ip, instance_id) VALUES (?, ?, ?, ?)", m.ID, m.PoolID, m.ExternalIP, m.InstanceID)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	return nil
}

func (ds *sqliteDB) deleteMappedIP(ID string) error {
	datastore := ds.getTableDB("mapped_ips")

	ds.dbLock.Lock()
	defer ds.dbLock.Unlock()

	tx, err := datastore.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM mapped_ips WHERE id = ?", ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	return err
}

func (ds *sqliteDB) getMappedIPs() map[string]types.MappedIP {
	IPs := make(map[string]types.MappedIP)

	datastore := ds.getTableDB("mapped_ips")

	query := `SELECT	mapped_ips.id,
				mapped_ips.pool_id,
				mapped_ips.external_ip,
				mapped_ips.instance_id,
				instances.ip,
				instances.tenant_id,
				pools.name
		  FROM	mapped_ips
		  JOIN instances
		  ON instances.id = mapped_ips.instance_id
		  JOIN pools
		  ON pools.id = mapped_ips.pool_id`

	rows, err := datastore.Query(query)
	if err != nil {
		fmt.Println(err)
		return IPs
	}
	defer rows.Close()

	for rows.Next() {
		var IP types.MappedIP

		err = rows.Scan(&IP.ID, &IP.PoolID, &IP.ExternalIP, &IP.InstanceID, &IP.InternalIP, &IP.TenantID, &IP.PoolName)
		if err != nil {
			continue
		}

		IPs[IP.ExternalIP] = IP
	}

	if err = rows.Err(); err != nil {
		fmt.Println(err)
	}

	return IPs
}
