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

package database

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/boltdb/bolt"
)

// BoltDB database structure
type BoltDB struct {
	Name string
	DB   *bolt.DB
}

func newBoltDb() *BoltDB {
	return &BoltDB{
		Name: "bolt.DB",
	}
}

//NewBoltDBProvider returns a bolt based database that conforms
//to the DBProvider interface
func NewBoltDBProvider() *BoltDB {
	return newBoltDb()
}

// DbInit initialize Bolt database
func (db *BoltDB) DbInit(dbDir, dbFile string) error {

	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("Unable to create db directory (%s) %v", dbDir, err)
	}

	dbPath := path.Join(dbDir, dbFile)

	options := bolt.Options{
		Timeout: 3 * time.Second,
	}

	var err error
	db.DB, err = bolt.Open(dbPath, 0644, &options)
	if err != nil {
		return fmt.Errorf("initDb failed %v", err)
	}

	return err
}

// DbClose closes Bolt database
func (db *BoltDB) DbClose() error {
	return db.DB.Close()
}

// DbTableRebuild builds bolt table into memory
func (db *BoltDB) DbTableRebuild(table DbTable) error {
	tables := []string{table.Name()}
	if err := db.DbTablesInit(tables); err != nil {
		return fmt.Errorf("dbInit failed %v", err)
	}

	table.NewTable()

	err := db.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(table.Name()))

		err := b.ForEach(func(k, v []byte) error {
			vr := bytes.NewReader(v)

			val := table.NewElement()
			if err := gob.NewDecoder(vr).Decode(val); err != nil {
				return fmt.Errorf("Decode Error: %v %v %v", string(k), string(v), err)
			}
			Logger.Infof("%v key=%v, value=%v\n", table, string(k), val)

			return table.Add(string(k), val)
		})
		return err
	})
	return err
}

// DbTablesInit initializes list of tables in Bolt
func (db *BoltDB) DbTablesInit(tables []string) (err error) {

	Logger.Infof("dbInit Tables")
	for i, table := range tables {
		Logger.Infof("table[%v] := %v, %v", i, table, []byte(table))
	}

	err = db.DB.Update(func(tx *bolt.Tx) error {
		for _, table := range tables {
			_, err := tx.CreateBucketIfNotExists([]byte(table))
			if err != nil {
				return fmt.Errorf("Bucket creation error: %v %v", table, err)
			}
		}
		return nil
	})

	if err != nil {
		Logger.Errorf("Table creation error %v", err)
	}

	return err
}

// DbAdd adds a new element to table in Bolt database
func (db *BoltDB) DbAdd(table string, key string, value interface{}) (err error) {

	err = db.DB.Update(func(tx *bolt.Tx) error {
		var v bytes.Buffer

		if err := gob.NewEncoder(&v).Encode(value); err != nil {
			Logger.Errorf("Encode Error: %v %v", err, value)
			return err
		}

		bucket := tx.Bucket([]byte(table))
		if bucket == nil {
			return fmt.Errorf("Bucket %v not found", table)
		}

		err = bucket.Put([]byte(key), v.Bytes())
		if err != nil {
			return fmt.Errorf("Key Store error: %v %v %v %v", table, key, value, err)
		}
		return nil
	})

	return err
}

// DbDelete deletes an element from table in Bolt database
func (db *BoltDB) DbDelete(table string, key string) (err error) {

	err = db.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(table))
		if bucket == nil {
			return fmt.Errorf("Bucket %v not found", table)
		}

		data := bucket.Get([]byte(key))
		if len(data) == 0 {
			return fmt.Errorf("Key is not found: %v", key)
		}

		err = bucket.Delete([]byte(key))
		if err != nil {
			return fmt.Errorf("Key Delete error: %v %v ", key, err)
		}
		return nil
	})

	return err
}

// DbGet obtains value by key from table
func (db *BoltDB) DbGet(table string, key string, dbTable DbTable) (interface{}, error) {

	var elem interface{}

	err := db.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(table))
		if bucket == nil {
			return fmt.Errorf("Bucket %v not found", table)
		}
		elem = dbTable.NewElement()
		data := bucket.Get([]byte(key))

		if data == nil {
			return nil
		}

		vr := bytes.NewReader(data)
		if err := gob.NewDecoder(vr).Decode(elem); err != nil {
			return err
		}
		return nil
	})

	return elem, err
}

// DbGetAll gets all elements from specific table
func (db *BoltDB) DbGetAll(table string, dbTable DbTable) (elements []interface{}, err error) {
	err = db.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(table))

		err := b.ForEach(func(key, value []byte) error {
			vr := bytes.NewReader(value)
			elem := dbTable.NewElement()
			if err := gob.NewDecoder(vr).Decode(elem); err != nil {
				return err
			}
			elements = append(elements, elem)
			return nil
		})
		return err
	})

	return elements, err
}
