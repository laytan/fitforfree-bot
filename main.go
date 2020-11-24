package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gorm.io/gorm"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/laytan/go-fff-notifications-bot/bot"
	"github.com/laytan/go-fff-notifications-bot/database"
)

func main() {
	// TODO: Different log file per environment: production, development etc.
	logFile, err := os.OpenFile("fff.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	// Set up logging so it writes to stdout and to a file
	wrt := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(wrt)

	log.Println("Starting program")

	// Get database conn
	db := database.New("database/database.sqlite")

	// middlewares are ran on every chat update
	middleware := []bot.Middleware{
		bot.Middleware{
			IsSync:  true,
			Handler: assureUserExists(db),
		},
		bot.Middleware{
			IsSync:  false,
			Handler: logUpdate,
		},
	}

	// handlers handle specific messages
	handlers := []bot.Handler{
		bot.CommandHandler{
			Command: "help",
			Handler: helpHandler,
		},
	}

	// start bot with our middlewares and handlers
	bot.Start(middleware, handlers)

	// Channel to send to when we should exit the program
	stop := make(chan bool, 1)
	handleStop(stop)

	log.Println("Waiting for exit signal on main thread")
	// Wait for stop channel so program does not exit
	<-stop
	log.Println("Stopping program")
}

// func check() {
// 	ticker := time.NewTicker(10 * time.Second)
// 	go func() {
// 		for {
// 			<-ticker.C
// 			fmt.Println("tick")
// 		}
// 	}()
// }

func handleStop(stop chan bool) {
	// Set up signal channel and listen for sigint and sigterm
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a signal and send to stop channel
	go func() {
		sig := <-sigs
		log.Printf("Received stop signal: %s", sig)
		stop <- true
	}()
}

func helpHandler(p *bot.HandlePayload, _ []string) {
	msg := tgbotapi.NewMessage(p.Update.Message.Chat.ID, fmt.Sprintf("Geen stress %s", p.User.Name))
	p.Bot.Send(msg)
}

// assureUserExists checks if the update user id exists in our database
// if it does we set the handlePayload's user to it
// if it does not we create the user and assign it to the handlePayload
func assureUserExists(db *database.Database) func(*bot.HandlePayload) {
	return func(p *bot.HandlePayload) {
		db.Mutex.Lock()
		defer db.Mutex.Unlock()

		user := database.User{}
		// Try to get the user with given id, if err and err is because we did not find
		if err := db.Conn.First(&user, p.Update.Message.From.ID).Error; err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			// TODO: Add database.NewUserFromUpdate(tgbotapi.Update) on user model
			// Create new user
			user = database.User{
				ID:       uint(p.Update.Message.From.ID),
				Name:     fmt.Sprintf("%s %s", p.Update.Message.From.FirstName, p.Update.Message.From.LastName),
				Username: p.Update.Message.From.UserName,
				ChatID:   uint(p.Update.Message.Chat.ID),
			}

			if err := db.Conn.Create(&user).Error; err != nil {
				log.Printf("ERROR: %+v with UPDATE: %+v", err, p.Update)
			}

			log.Printf("[%s](%d) User Created", p.Update.Message.From.FirstName, p.Update.Message.From.ID)
		}
		p.User = user
	}
}

func logUpdate(p *bot.HandlePayload) {
	log.Printf("[%s](%d) Message: %s", p.Update.Message.From.FirstName, p.Update.Message.From.ID, p.Update.Message.Text)
}
