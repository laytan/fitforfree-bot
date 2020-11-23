package database

import (
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Database encapsulates the gorm db and a mutex lock to never get a database is locked error
type Database struct {
	Conn  *gorm.DB
	Mutex sync.Mutex
}

// New returns a database connection which is migrated acording to the models
func New(dbPath string, models ...interface{}) Database {
	// Connect to sqlite db and initialize gorm ORM
	gormDb, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	err = gormDb.AutoMigrate(models...)
	if err != nil {
		panic(err)
	}

	return Database{
		Conn:  gormDb,
		Mutex: sync.Mutex{},
	}
}
