package database

import (
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User model
type User struct {
	gorm.Model
	ID       uint `gorm:"primaryKey"`
	Name     string
	Username string
	ChatID   uint
	Notis    []Noti
}

// Noti model
type Noti struct {
	gorm.Model
	UserID uint
	User   User
	Start  uint
	End    uint
}

// TODO: Remove using lock because that would only be a problem on thousands of writes and goroutines
// Database encapsulates the gorm db and a mutex lock to never get a database is locked error
type Database struct {
	Conn  *gorm.DB
	Mutex sync.Mutex
}

// New returns a database connection which is migrated acording to the models
func New(dbPath string) *Database {
	// Connect to sqlite db and initialize gorm ORM
	gormDb, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	err = gormDb.AutoMigrate(&User{}, &Noti{})
	if err != nil {
		panic(err)
	}

	return &Database{
		Conn:  gormDb,
		Mutex: sync.Mutex{},
	}
}
