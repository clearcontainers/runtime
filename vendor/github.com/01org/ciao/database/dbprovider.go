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

// DbTable defines basic table operations
type DbTable interface {
	// Creates the backing map
	NewTable()
	// Name of the table as stored in the database
	Name() string
	// Allocates and returns a single value in the table
	NewElement() interface{}
	// Add an value to the in memory table
	Add(k string, v interface{}) error
}

// DbProvider represents a persistent database provider
type DbProvider interface {
	// Initializes the Database
	DbInit(dbDir, dbFile string) error
	// Closes the database
	DbClose() error
	// Creates the tables if the tables do not already exist in the database
	DbTablesInit(tables []string) error
	// Populates the in-memory table from the database
	DbTableRebuild(table DbTable) error
	// Adds the key/value pair to the table
	DbAdd(table string, key string, value interface{}) error
	//Deletes the key/value pair from the table
	DbDelete(table string, key string) error
	//Retrives the value corresponding to the key from the table
	DbGet(table string, key string, dbTable DbTable) (interface{}, error)
	//Retrieves all values from a table
	DbGetAll(table string, dbTable DbTable) ([]interface{}, error)
}
