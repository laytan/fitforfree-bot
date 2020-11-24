package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/laytan/go-fff-notifications-bot/bot"
	"github.com/laytan/go-fff-notifications-bot/database"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	// Setup logs and close log file once the program stops
	logFile := setupLogs()
	defer logFile.Close()

	log.Println("Starting program")

	// Get database conn
	db := database.New("database/database.sqlite")

	// middlewares are ran on every chat update
	middleware := []bot.Middleware{
		{
			IsSync:  true,
			Handler: AssureUserExists(db),
		},
		{
			IsSync:  false,
			Handler: LogUpdate,
		},
	}

	// handlers handle specific messages
	handlers := []bot.Handler{
		&bot.CommandHandler{
			Command: "help",
			Handler: HelpHandler,
		},
		&bot.CommandHandler{
			Command: "notificaties",
			Handler: ListNotisHandler(db),
		},
		bot.NewConversationHandler(
			"noti",
			[]bot.ConversationHandlerFunc{
				StartNotiHandler,
				DateNotiHandler,
				StartTimeNotiHandler,
				EndTimeNotiHandler,
				TypeNotiHandler,
			},
			NotiHandler(db),
		),
	}

	// start bot with our middlewares and handlers
	go bot.Start(middleware, handlers)

	// Channel to send to when we should exit the program
	stop := handleStop()

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

// setupLogs sets up the logger to log to file and stdout. Filename is dependant on the environment
func setupLogs() *os.File {
	env, isSet := os.LookupEnv("ENV")
	if !isSet {
		env = "development"
	}

	logFile, err := os.OpenFile(fmt.Sprintf("logs/%s.log", env), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	// Set up logging so it writes to stdout and to a file
	wrt := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(wrt)

	return logFile
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
