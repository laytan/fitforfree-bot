package middleware

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/laytan/go-fff-notifications-bot/bot"
	"github.com/laytan/go-fff-notifications-bot/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TestCaseAssureUserExists struct {
	user *database.User
	from *tgbotapi.User
}

func TestAssureUserExists(t *testing.T) {
	cases := []TestCaseAssureUserExists{
		{
			user: &database.User{
				ID: 1,
			},
			from: &tgbotapi.User{
				ID: 1,
			},
		},
		{
			user: &database.User{
				ID: 0,
			},
			from: &tgbotapi.User{
				ID: 0,
			},
		},
		{
			user: &database.User{
				ID: 1123,
			},
			from: &tgbotapi.User{
				ID: 1123,
			},
		},
	}

	// Connect to db
	db, err := gorm.Open(sqlite.Open("../database/test.sqlite"), &gorm.Config{})
	if err != nil {
		t.Error(err)
	}
	db.AutoMigrate(&database.User{})
	middleware := AssureUserExists(db)

	for _, tCase := range cases {
		// Clear db and add user
		db.Where("1 = 1").Delete(&database.User{})
		db.Create(tCase.user)

		p := bot.HandlePayload{
			Update: tgbotapi.Update{
				Message: &tgbotapi.Message{
					From: tCase.from,
					Chat: &tgbotapi.Chat{
						ID: 1,
					},
				},
			},
		}

		// run middleware
		middleware(&p)

		// check payload for user
		if p.User.ID == 0 && tCase.user.ID != 0 {
			t.Error("User should be here")
		}
	}
}

func TestNoMessageAssureUserExists(t *testing.T) {
	// Connect to db
	db, err := gorm.Open(sqlite.Open("../database/test.sqlite"), &gorm.Config{})
	if err != nil {
		t.Error(err)
	}
	db.AutoMigrate(&database.User{})
	middleware := AssureUserExists(db)
	p := bot.HandlePayload{
		Update: tgbotapi.Update{},
	}
	middleware(&p)
	if p.User.ID != 0 {
		t.Error("Should not have a user here")
	}
}
