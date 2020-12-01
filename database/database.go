package database

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/laytan/go-fff-notifications-bot/fitforfree"
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

// Admin returns if the user is an admin
func (u User) Admin() bool {
	return os.Getenv("ADMIN_CHAT_ID") == fmt.Sprintf("%d", u.ChatID)
}

// Noti model
type Noti struct {
	gorm.Model
	UserID   uint
	User     User
	LessonID string
	Lesson   Lesson
}

// Lesson model
type Lesson struct {
	gorm.Model
	ID              string `gorm:"primaryKey"`
	Start           uint
	DurationSeconds uint
	ClassType       string
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

	err = gormDb.AutoMigrate(&User{}, &Noti{}, &Lesson{})
	if err != nil {
		panic(err)
	}

	return gormDb
}

// NewLogger returns a logger based on the environment
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

// CreateNoti creates a noti and a lesson if it does not already exist
func CreateNoti(db *gorm.DB, user User, lesson fitforfree.Lesson) error {
	l := Lesson{
		ID:              lesson.ID,
		Start:           lesson.StartTimestamp,
		DurationSeconds: lesson.DurationSeconds,
		ClassType:       lesson.ClassType,
	}
	db.FirstOrCreate(&l)

	// Check if there is already a noti for this relationship
	if err := db.Where("lesson_id = ? AND user_id = ?", l.ID, user.ID).First(&Noti{}).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// Did not find it, create it
		if err := db.Create(&Noti{UserID: user.ID, LessonID: l.ID}).Error; err != nil {
			return err
		}
	}

	// No error on query so it already exists
	return nil
}
