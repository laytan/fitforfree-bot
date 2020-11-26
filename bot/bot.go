package bot

import (
	"log"
	"os"
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

func (p HandlePayload) Respond(text string) {
	var chatID int64
	if p.Update.Message != nil {
		chatID = p.Update.Message.Chat.ID
	}

	if p.Update.CallbackQuery != nil {
		chatID = p.Update.CallbackQuery.Message.Chat.ID
	}

	msg := tgbotapi.NewMessage(chatID, text)
	p.Bot.Send(msg)
}

// Handler is the interface used to handle bot updates
type Handler interface {
	isMatch(update tgbotapi.Update) bool
	handle(payload *HandlePayload)
}

// CommandHandler listens for any command with the given Command with a slash prepended
type CommandHandler struct {
	Command []string
	Handler func(payload *HandlePayload, arguments []string)
}

// isMatch determines if this update should be handled by this handler
func (c *CommandHandler) isMatch(update tgbotapi.Update) bool {
	if update.Message != nil && update.Message.IsCommand() {
		for _, command := range c.Command {
			if command == update.Message.Command() {
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

// ConversationState is
type ConversationState struct {
	Value interface{}
}

// ConversationHandlerFunc is a function used as a handler in the conversation handler
type ConversationHandlerFunc func(payload *HandlePayload, state []ConversationState) (interface{}, bool)

// ConversationFinalizerFunc is a function that gets passed the state of the conversation after it is finished
type ConversationFinalizerFunc func(payload *HandlePayload, state []ConversationState)

type conversationHandler struct {
	startCommand []string
	inProgress   bool
	// The result from all ran handlers
	state []ConversationState
	// handlers to run, ran in order
	handlers []ConversationHandlerFunc
	// func that gets the full conversation state when all handlers have ran
	finalizer ConversationFinalizerFunc
}

// isMatch will tell the caller we want to handle the update when we are in progress of a conversation or the start command is given
func (c *conversationHandler) isMatch(update tgbotapi.Update) bool {
	if c.inProgress {
		return true
	}

	if update.Message != nil && update.Message.IsCommand() {
		for _, command := range c.startCommand {
			if command == update.Message.Command() {
				return true
			}
		}
		return false
	}
	return false
}

// handle determines which handler to run and what to pass it
func (c *conversationHandler) handle(p *HandlePayload) {
	c.inProgress = true

	// get handler to pass to now and check if the handler exists
	curr := len(c.state)

	res, valid := c.handlers[curr](p, c.state)
	if valid == false {
		// Return without changing a thing so we stay in this handler for the user to try again
		return
	}

	// Append result from handler to the conversation state
	c.state = append(c.state, ConversationState{
		Value: res,
	})

	if len(c.handlers)-1 == curr {
		// Run the finalizer with the state retrieved from the conversation
		c.finalizer(p, c.state)
		// Reset because we are done with the conversation
		c.inProgress = false
		c.state = make([]ConversationState, 0)
		return
	}
}

// NewConversationHandler returns a conversationhandler with specified options
func NewConversationHandler(startCommand []string, handlers []ConversationHandlerFunc, finalizer ConversationFinalizerFunc) *conversationHandler {
	return &conversationHandler{
		inProgress:   false,
		state:        make([]ConversationState, 0),
		startCommand: startCommand,
		handlers:     handlers,
		finalizer:    finalizer,
	}
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

	// Listen for new updates over the updates channel
	for update := range updates {
		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}

		p := HandlePayload{
			Bot:    bot,
			Update: update,
		}

		// run all our middlewares
		for _, m := range middleware {
			if m.IsSync == true {
				m.Handler(&p)
			} else {
				go m.Handler(&p)
			}
		}

		// Find the right handler and call it
		for _, handler := range handlers {
			// Check if the handler should respond to this update
			if handler.isMatch(update) {
				// handle update in seperate goroutine
				go handler.handle(&p)
				// Break because the update is handled
				break
			}
		}
	}
}

func parseArgs(text string) []string {
	words := strings.Split(text, " ")
	return words[1:]
}
