package database

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
func New(dbPath string, theLogger logger.Interface) *gorm.DB {
	// Connect to sqlite db and initialize gorm ORM
	gormDb, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: theLogger,
	})
	if err != nil {
		panic(err)
	}

	err = gormDb.AutoMigrate(&User{}, &Noti{})
	if err != nil {
		panic(err)
	}

	return gormDb
}

func NewLogger(logFile *os.File) logger.Interface {
	if os.Getenv("ENV") == "production" {
		// Set up logging so it writes to stdout and to a file
		wrt := io.MultiWriter(os.Stdout, logFile)

		return logger.New(
			log.New(wrt, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold: time.Millisecond * 100,
				LogLevel:      logger.Warn,
				Colorful:      false,
			},
		)
	}

	return logger.Default.LogMode(logger.Info)
}
