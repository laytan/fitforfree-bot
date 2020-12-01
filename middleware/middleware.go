package middleware

import (
	"fmt"
	"log"

	"github.com/laytan/go-fff-notifications-bot/bot"
	"github.com/laytan/go-fff-notifications-bot/database"
	"gorm.io/gorm"
)

// AssureUserExists checks if the update user id exists in our database
// if it does we set the handlePayload's user to it
// if it does not we create the user and assign it to the handlePayload
func AssureUserExists(db *gorm.DB) func(*bot.HandlePayload) {
	return func(p *bot.HandlePayload) {
		if p.Update.Message == nil {
			return
		}

		user := database.User{
			ID:       uint(p.Update.Message.From.ID),
			Name:     fmt.Sprintf("%s %s", p.Update.Message.From.FirstName, p.Update.Message.From.LastName),
			Username: p.Update.Message.From.UserName,
			ChatID:   uint(p.Update.Message.Chat.ID),
		}

		if err := db.FirstOrCreate(&user).Error; err != nil {
			log.Printf("ERROR: Error creating/getting user in middleware %+v", err)
		}

		p.User = user
	}
}

// LogUpdate logs all messages sent to the program
func LogUpdate(p *bot.HandlePayload) {
	if p.Update.Message != nil {
		log.Printf("[%s](%d) Message: %s", p.Update.Message.From.FirstName, p.Update.Message.From.ID, p.Update.Message.Text)
	}
}
