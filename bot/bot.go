package bot

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/laytan/go-fff-notifications-bot/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Sender interface {
	Send(tgbotapi.Chattable) (tgbotapi.Message, error)
}

// HandlePayload wraps update and bot for convenience
type HandlePayload struct {
	Update tgbotapi.Update
	Bot    Sender
	User   database.User
}

func (p HandlePayload) Respond(text string) {
	var chatID int64
	if p.Update.Message != nil {
		chatID = p.Update.Message.Chat.ID
	} else if p.Update.CallbackQuery != nil {
		chatID = p.Update.CallbackQuery.Message.Chat.ID
	} else {
		return
	}

	msg := tgbotapi.NewMessage(chatID, text)
	p.Bot.Send(msg)
}

// Handler is the interface used to handle bot updates
type Handler interface {
	isMatch(payload *HandlePayload) bool
	handle(payload *HandlePayload)
}

// CommandHandler listens for any command with the given Command with a slash prepended
type CommandHandler struct {
	Command []string
	Handler func(payload *HandlePayload, arguments []string)
}

// isMatch determines if this update should be handled by this handler
func (c *CommandHandler) isMatch(p *HandlePayload) bool {
	if p.Update.Message != nil && p.Update.Message.IsCommand() {
		for _, command := range c.Command {
			if command == p.Update.Message.Command() {
				return true
			}
		}
		return false
	}
	return false
}

func (c *CommandHandler) handle(p *HandlePayload) {
	c.Handler(p, parseArgs(p.Update.Message.Text))
}

// Middleware is ran on every request, the handler must only change the handlepayload when IsSync is true
// If IsSync is false and the handlepayload is changed the handler will probably not get the updated values
type Middleware struct {
	IsSync  bool
	Handler func(p *HandlePayload)
}

// Start sets up the bot and starts retrieving updates
func Start(middleware []Middleware, handlers []Handler) {
	botToken, isSet := os.LookupEnv("BOT_TOKEN")
	if !isSet {
		log.Panic("ERROR: BOT_TOKEN environment variable not set")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panicf("ERROR: can't connect to telegram api: %+v", err)
	}

	log.Printf("Bot %s authorized\n", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	syncMiddleware, asyncMiddleware := splitMiddleware(middleware)

	for update := range updates {
		handle(update, bot, syncMiddleware, asyncMiddleware, handlers)
	}
}

func handle(update tgbotapi.Update, sender Sender, syncMiddleware []Middleware, asyncMiddleware []Middleware, handlers []Handler) {
	if update.Message == nil && update.CallbackQuery == nil {
		return
	}

	p := HandlePayload{
		Bot:    sender,
		Update: update,
	}

	fmt.Println(syncMiddleware, asyncMiddleware)

	// run all our middlewares
	for _, s := range syncMiddleware {
		s.Handler(&p)
	}

	for _, a := range asyncMiddleware {
		go a.Handler(&p)
	}

	// Find the right handler and call it
	for _, handler := range handlers {
		// Check if the handler should respond to this update
		if handler.isMatch(&p) {
			// handle update in seperate goroutine
			go handler.handle(&p)
			// Break because the update is handled
			break
		}
	}
}

func parseArgs(text string) []string {
	words := strings.Split(text, " ")
	return words[1:]
}

func splitMiddleware(middleware []Middleware) ([]Middleware, []Middleware) {
	asyncMiddleware := make([]Middleware, 0)
	syncMiddleware := make([]Middleware, 0)
	for _, m := range middleware {
		if m.IsSync {
			syncMiddleware = append(syncMiddleware, m)
		} else {
			asyncMiddleware = append(asyncMiddleware, m)
		}
	}

	return syncMiddleware, asyncMiddleware
}
