database
========

``database`` is a package that provides a Generic DB Abstraction layer. It establishes the 2 main interfaces for defininf a new database provider and a new table definition.

DB Providers
------------
In order to add a new database provider, you will be required to implement the following interface:
```golang
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
```

### Current supported DB Providers
- BoltDB (key/value database)

DB Table
--------
There is a generic table interface that needs to be implemented in order to define a generic table structure.
```golang
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
```
