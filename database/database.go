package database

import (
	"fmt"
	"os"

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

func (u User) Admin() bool {
	return os.Getenv("ADMIN_CHAT_ID") == fmt.Sprintf("%d", u.ChatID)
}

// Noti model
type Noti struct {
	gorm.Model
	UserID    uint
	User      User
	Start     uint64
	End       uint64
	ClassType string
}

// New returns a database connection which is migrated acording to the models
func New(dbPath string) *gorm.DB {
	// Connect to sqlite db and initialize gorm ORM
	gormDb, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	err = gormDb.AutoMigrate(&User{}, &Noti{})
	if err != nil {
		panic(err)
	}

	return gormDb
}
