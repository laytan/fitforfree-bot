package bot

import (
	"log"
	"strings"

	"github.com/laytan/go-fff-notifications-bot/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// HandlePayload wraps update and bot for convenience
type HandlePayload struct {
	Update tgbotapi.Update
	Bot    *tgbotapi.BotAPI
	User   database.User
}

// Handler is the interface used to handle bot updates
type Handler interface {
	isMatch(update tgbotapi.Update) bool
	handle(payload *HandlePayload)
}

// CommandHandler listens for any command with the given Command with a slash prepended
type CommandHandler struct {
	Command string
	Handler func(payload *HandlePayload, arguments []string)
}

// isMatch determines if this update should be handled by this handler
func (c CommandHandler) isMatch(update tgbotapi.Update) bool {
	return update.Message.IsCommand() && update.Message.Command() == c.Command
}

func (c CommandHandler) handle(p *HandlePayload) {
	words := strings.Split(p.Update.Message.Text, " ")
	c.Handler(p, words[1:])
}

// Middleware is ran on every request, the handler must only change the handlepayload when IsSync is true
// If IsSync is false and the handlepayload is changed the handler will probably not get the updated values
type Middleware struct {
	IsSync  bool
	Handler func(p *HandlePayload)
}

// Start sets up the bot and starts retrieving updates
func Start(middleware []Middleware, handlers []Handler) {
	// TODO: Move API key to env
	bot, err := tgbotapi.NewBotAPI("")
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Bot %s authorized\n", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	// Listen for new updates over the updates channel
	for update := range updates {
		// Loop through our handlers
		for _, handler := range handlers {
			// Check if the handler should respond to this update
			if handler.isMatch(update) {
				// handle update in seperate goroutine
				go func() {
					p := HandlePayload{
						Bot:    bot,
						Update: update,
					}

					// run all our middlewares in a seperate goroutine if allowed by middleware
					for _, m := range middleware {
						if m.IsSync == true {
							m.Handler(&p)
						} else {
							go m.Handler(&p)
						}
					}

					// handle the update
					handler.handle(&p)
				}()
				// Break because the update is handled
				break
			}
		}
	}
}
