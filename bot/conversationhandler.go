package bot

import (
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type conversationHandler struct {
	startCommand []string
	instances    *conversationInstances
	handlers     []ConversationHandlerFunc
	finalizer    ConversationFinalizerFunc
}

// NewConversationHandler returns a conversationhandler with specified options
func NewConversationHandler(startCommand []string, handlers []ConversationHandlerFunc, finalizer ConversationFinalizerFunc) *conversationHandler {
	return &conversationHandler{
		instances: &conversationInstances{
			interMap: sync.Map{},
		},
		startCommand: startCommand,
		handlers:     handlers,
		finalizer:    finalizer,
	}
}

// isMatch checks if any running instance want to handle the update or if the message is it's start command which will add a new instance
func (c *conversationHandler) isMatch(p *HandlePayload) bool {
	// check if there is an instance to that wants to handle this update
	if _, match := anyMatch(c.instances, p.Update); match {
		return true
	}

	// check if we should start a new instance for this update
	if p.Update.Message != nil && p.Update.Message.IsCommand() {
		for _, command := range c.startCommand {
			// Remove / from command
			if command == p.Update.Message.Command() {
				// start command received, create new instance
				state := make([]interface{}, 0)
				c.instances.Store(p.User.ID, conversationHandlerInstance{
					state: &state,
					lock:  &sync.Mutex{},
				})

				return true
			}
		}
	}
	return false
}

// handle determines which handler to run and what to pass it
func (c *conversationHandler) handle(p *HandlePayload) {
	instance, matched := anyMatch(c.instances, p.Update)
	if matched == false {
		log.Println("ERROR: No instance found")
		return
	}

	instance.lock.Lock()
	defer instance.lock.Unlock()

	// get handler to pass to now and check if the handler exists
	curr := len(*instance.state)

	// execute handler
	res, valid := c.handlers[curr](p, *instance.state)
	if valid == false {
		// Return without changing a thing so we stay in this handler for the user to try again
		return
	}

	// Append result from handler to the conversation state
	*instance.state = append(*instance.state, res)

	if len(c.handlers)-1 == curr {
		// Run the finalizer with the state retrieved from the conversation
		c.finalizer(p, *instance.state)

		// Remove conversation instance because it is done
		c.instances.Delete(p.User.ID)
		return
	}
}

// ConversationHandlerFunc is a function used as a handler in the conversation handler
type ConversationHandlerFunc func(payload *HandlePayload, state []interface{}) (interface{}, bool)

// ConversationFinalizerFunc is a function that gets passed the state of the conversation after it is finished
type ConversationFinalizerFunc func(payload *HandlePayload, state []interface{})

// conversationIntances is a sync map wrapped with type safe functions
type conversationInstances struct {
	// interMap should never be accessed directly to keep it type safe
	interMap sync.Map
}

func (c *conversationInstances) Store(id uint, i conversationHandlerInstance) {
	c.interMap.Store(id, i)
}

func (c *conversationInstances) Load(id uint) (conversationHandlerInstance, bool) {
	i, exists := c.interMap.Load(id)
	if exists {
		return i.(conversationHandlerInstance), true
	}
	return conversationHandlerInstance{}, false
}

func (c *conversationInstances) Delete(id uint) {
	c.interMap.Delete(id)
}

func anyMatch(instances *conversationInstances, update tgbotapi.Update) (conversationHandlerInstance, bool) {
	if update.Message != nil {
		return instances.Load(uint(update.Message.From.ID))
	} else if update.CallbackQuery != nil {
		return instances.Load(uint(update.CallbackQuery.From.ID))
	}
	return conversationHandlerInstance{}, false
}

type conversationHandlerInstance struct {
	// The result from all ran handlers
	state *[]interface{}
	// Lock to avoid race conditions when running handlers that access the state
	lock *sync.Mutex
}
