package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"github.com/laytan/go-fff-notifications-bot/bot"
	"github.com/laytan/go-fff-notifications-bot/checker"
	"github.com/laytan/go-fff-notifications-bot/database"
	"github.com/laytan/go-fff-notifications-bot/handlers"
	"github.com/laytan/go-fff-notifications-bot/logs"
	"github.com/laytan/go-fff-notifications-bot/middleware"
	"github.com/laytan/go-fff-notifications-bot/times"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	// Setup logs and close log file once the program stops
	logFile := logs.SetupLogs()
	defer logFile.Close()

	log.Println("Starting program")

	// Get database conn
	db := database.New("database/database.sqlite", logs.NewDatabaseLogger(logFile))

	// middlewares are ran on every chat update
	middleware := []bot.Middleware{
		{
			IsSync:  true,
			Handler: middleware.AssureUserExists(db),
		},
		{
			IsSync:  false,
			Handler: middleware.LogUpdate,
		},
	}

	// handlers handle specific messages
	handlers := []bot.Handler{
		&bot.CommandHandler{
			Command: []string{"help", "start"},
			Handler: handlers.HelpHandler,
		},
		&bot.CommandHandler{
			Command: []string{"notificaties", "notifications"},
			Handler: handlers.ListNotisHandler(db),
		},
		&bot.CommandHandler{
			Command: []string{"verwijder", "remove"},
			Handler: handlers.RemoveHandler(db),
		},
		&bot.CommandHandler{
			Command: []string{"clear"},
			Handler: handlers.ClearHandler(db),
		},
		bot.NewConversationHandler(
			[]string{"noti"},
			[]bot.ConversationHandlerFunc{
				// Ask for date
				handlers.StartNotiHandler,
				// Ask for group or free
				handlers.DateNotiHandler,
				// Show lessons on that day and ask for choise
				handlers.TypeNotiHandler,
				// Get specific class
				handlers.ClassNotiHandler,
			},
			handlers.NotiHandler(db),
		),
	}

	// start bot with our middlewares and handlers
	bot := bot.Start(middleware, handlers)

	// Setup checker
	checkerT := time.NewTicker(time.Second * 100)
	shouldNotify := make(chan database.Noti)
	// Wait for checker in other goroutine
	go func() {
		for {
			<-checkerT.C
			// Initiate the check
			checker.AvailabilityCheck(db, []string{os.Getenv("VENUE")}, os.Getenv("FIT_FOR_FREE_TOKEN"), shouldNotify)
		}
	}()

	go func() {
		for {
			available := <-shouldNotify

			// Construct message
			msg := fmt.Sprintf(
				`
				Snel er is plek vrij!
				
				Les: %s
				Datum: %s
				Start: %s
				Eind: %s
				`,
				available.Lesson.Name,
				times.FormatTimestamp(available.Lesson.Start, times.DateLayout),
				times.FormatTimestamp(available.Lesson.Start, times.TimeLayout),
				times.FormatTimestamp(available.Lesson.Start+available.Lesson.DurationSeconds, times.TimeLayout),
			)

			// Send msg to user
			bot.Send(tgbotapi.NewMessage(int64(available.User.ChatID), msg))
		}
	}()

	// Channel to send to when we should exit the program
	stop := handleStop()

	log.Println("Waiting for exit signal on main thread")
	// Wait for stop channel so program does not exit
	<-stop
	log.Println("Stopping program")
}

// handleStop sends true to the returned channel when sigint or sigterm is received
func handleStop() chan bool {
	stop := make(chan bool, 1)
	// Set up signal channel and listen for sigint and sigterm
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a signal and send to stop channel
	go func() {
		sig := <-sigs
		log.Printf("Received stop signal: %s", sig)
		stop <- true
	}()

	return stop
}
