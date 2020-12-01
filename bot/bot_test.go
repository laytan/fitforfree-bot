package bot

import (
	"fmt"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/laytan/go-fff-notifications-bot/database"
)

type mockSender struct {
	OnSend func(tgbotapi.Chattable)
}

func (m mockSender) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	m.OnSend(c)
	return tgbotapi.Message{}, nil
}

func newMockCommandUpdate(command string, args string) tgbotapi.Update {
	return tgbotapi.Update{
		Message: &tgbotapi.Message{
			Text: fmt.Sprintf("%s %s", command, args),
			Entities: &[]tgbotapi.MessageEntity{
				{
					Offset: 0,
					Type:   "bot_command",
					Length: len(command),
				},
			},
		},
	}
}

func newMockUpdate(msg string) tgbotapi.Update {
	return tgbotapi.Update{
		Message: &tgbotapi.Message{
			Text: msg,
		},
	}
}

func TestCommandHandler(t *testing.T) {
	failTimer := time.NewTimer(time.Second)
	done := make(chan bool)

	sender := mockSender{}
	middlewares := make([]Middleware, 0)
	handlers := []Handler{
		&CommandHandler{
			Command: []string{"test"},
			Handler: func(p *HandlePayload, args []string) {
				done <- true
			},
		},
	}

	update := newMockCommandUpdate("/test", "")

	handle(update, sender, middlewares, middlewares, handlers)

	select {
	case <-failTimer.C:
		t.Error("Did not handle test command in 1 second")
	case <-done:
		return
	}
}

func TestConversationHandler(t *testing.T) {
	failTimer := time.NewTimer(time.Second * 1)
	done := make(chan bool)

	sender := mockSender{}

	// Conversations require a user
	middlewares := []Middleware{
		{
			IsSync: true,
			Handler: func(p *HandlePayload) {
				p.User = database.User{
					ID: 2,
				}
			},
		},
	}

	handlers := []Handler{
		NewConversationHandler(
			[]string{"test"},
			[]ConversationHandlerFunc{
				func(_ *HandlePayload, _ []interface{}) (interface{}, bool) {
					return "Handler 1", true
				},
				func(_ *HandlePayload, _ []interface{}) (interface{}, bool) {
					return database.User{ID: 2}, true
				},
			},
			func(_ *HandlePayload, state []interface{}) {
				if state[0] != "Handler 1" {
					t.Error("Handler 1 state not correct")
				}

				userFromState, ok := state[1].(database.User)
				if !ok {
					t.Error("Did not receive user struct from state/conversation")
				}

				if userFromState.ID != 2 {
					t.Error("Did not get User ID 2 from state/conversation")
				}

				done <- true
			},
		),
	}

	update := newMockCommandUpdate("/test", "")
	update.Message.From = &tgbotapi.User{ID: 2}
	update2 := newMockUpdate("bla")
	update2.Message.From = &tgbotapi.User{ID: 2}

	handle(update, sender, middlewares, make([]Middleware, 0), handlers)

	timer := time.NewTimer(time.Millisecond * 50)
	<-timer.C

	handle(update2, sender, middlewares, make([]Middleware, 0), handlers)

	select {
	case <-failTimer.C:
		t.Error("Did not handle conversation in 1 second")
	case <-done:
		return
	}
}

func TestMultipleConversations(t *testing.T) {
	wg := sync.WaitGroup{}
	sender := mockSender{}

	// Conversations require a user
	middlewares := []Middleware{
		{
			IsSync: true,
			Handler: func(p *HandlePayload) {
				p.User = database.User{ID: uint(p.Update.Message.From.ID)}
			},
		},
	}

	handlers := []Handler{
		NewConversationHandler(
			[]string{"test"},
			[]ConversationHandlerFunc{
				func(payload *HandlePayload, _ []interface{}) (interface{}, bool) {
					return payload.User.ID, true
				},
				func(payload *HandlePayload, _ []interface{}) (interface{}, bool) {
					return payload.User.ID, true
				},
			},
			func(_ *HandlePayload, state []interface{}) {
				userIDFromHandler, ok := state[0].(uint)
				if !ok {
					t.Error("Did not receive uint from state 0")
				}

				userIDFromHandler2, ok := state[0].(uint)
				if !ok {
					t.Error("Did not receive uint from state 1")
				}

				if userIDFromHandler != userIDFromHandler2 {
					t.Error("User id's between handlers changed")
				}

				wg.Done()
			},
		),
	}

	asyncMiddleware := make([]Middleware, 0)

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(index int) {
			update := newMockCommandUpdate("/test", "")
			update.Message.From = &tgbotapi.User{ID: index}
			update2 := newMockUpdate("bla")
			update2.Message.From = &tgbotapi.User{ID: index}

			handle(update, sender, middlewares, asyncMiddleware, handlers)

			timer := time.NewTimer(time.Duration(rand.Intn(50)+50) * time.Millisecond)
			<-timer.C

			handle(update2, sender, middlewares, asyncMiddleware, handlers)
		}(i)
	}

	wg.Wait()
}

type AllHandler struct {
	Handler func(payload *HandlePayload)
}

func (a *AllHandler) isMatch(p *HandlePayload) bool {
	return true
}

func (a *AllHandler) handle(p *HandlePayload) {
	a.Handler(p)
}

func TestNonMessageOrCallbackQueryDoesNotGetHandled(t *testing.T) {
	update := tgbotapi.Update{UpdateID: 1}
	handlers := []Handler{
		&AllHandler{
			Handler: func(_ *HandlePayload) { t.Error("Should not get here") },
		},
	}
	handle(update, mockSender{}, make([]Middleware, 0), make([]Middleware, 0), handlers)
}

func TestAsyncMiddleware(t *testing.T) {
	timer := time.NewTimer(time.Second * 1)
	done := make(chan bool)
	update := newMockCommandUpdate("/test", "bla bla")
	asyncMiddlewares := []Middleware{
		{
			IsSync: false,
			Handler: func(p *HandlePayload) {
				done <- true
			},
		},
	}

	handle(update, mockSender{}, make([]Middleware, 0), asyncMiddlewares, make([]Handler, 0))

	select {
	case <-timer.C:
		t.Error("Middleware not ran")
	case <-done:
		return
	}
}

func TestCommandHandlerIsMatch(t *testing.T) {
	payload := &HandlePayload{
		Update: newMockCommandUpdate("/test", ""),
	}

	handlerShouldMatch := &CommandHandler{
		Command: []string{"test"},
		Handler: func(payload *HandlePayload, arguments []string) {},
	}

	if !handlerShouldMatch.isMatch(payload) {
		t.Error("Command Handler should have matched with the payload")
	}

	handlerShouldNotMatch := &CommandHandler{
		Command: []string{"bka"},
		Handler: func(payload *HandlePayload, arguments []string) {},
	}

	if handlerShouldNotMatch.isMatch(payload) {
		t.Error("Command Handler should not have matched with the payload")
	}

	// Should not match empty update
	if handlerShouldMatch.isMatch(&HandlePayload{Update: tgbotapi.Update{}}) {
		t.Error("Command Handler should not match empty update")
	}
}

func TestPayloadRespond(t *testing.T) {
	messageSent := false
	sender := mockSender{
		OnSend: func(t tgbotapi.Chattable) { messageSent = true },
	}

	update := newMockCommandUpdate("/tst", "")
	update.Message.Chat = &tgbotapi.Chat{ID: 12}

	payload := HandlePayload{
		Bot:    sender,
		Update: update,
	}

	payload.Respond("test")

	if !messageSent {
		t.Error("Message should have been sent")
	}

	messageSent = false

	update = tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{
					ID: 12,
				},
			},
		},
	}

	payload = HandlePayload{
		Bot:    sender,
		Update: update,
	}

	payload.Respond("test")

	if !messageSent {
		t.Error("Message should be sent")
	}

	messageSent = false

	payload = HandlePayload{
		Bot:    sender,
		Update: tgbotapi.Update{},
	}

	payload.Respond("test")

	if messageSent {
		t.Error("Message should not be sent on empty update")
	}
}

func TestConversationHandlerHandleShouldNotCrashWhenThereIsNoInstanceAllOfTheSudden(t *testing.T) {
	handler := NewConversationHandler([]string{"test"}, make([]ConversationHandlerFunc, 0), func(payload *HandlePayload, state []interface{}) {})
	payload := HandlePayload{}
	handler.handle(&payload)
}

func TestConversationHandlerInvalidMessage(t *testing.T) {
	called := 0
	finalizerRan := false
	handlers := []ConversationHandlerFunc{
		func(payload *HandlePayload, state []interface{}) (interface{}, bool) {
			called++
			if called == 1 {
				return nil, false
			}
			return nil, true
		},
	}

	handler := NewConversationHandler([]string{"test"}, handlers,
		func(payload *HandlePayload, state []interface{}) {
			if called != 2 {
				t.Error("Handler should have been called twice")
			}
			finalizerRan = true
		},
	)

	state := make([]interface{}, 0)
	handler.instances.Store(1, conversationHandlerInstance{
		state: &state,
		lock:  &sync.Mutex{},
	})

	update := newMockUpdate("b")
	update.Message.From = &tgbotapi.User{
		ID: 1,
	}

	handler.handle(&HandlePayload{Update: update})
	handler.handle(&HandlePayload{Update: update})

	if !finalizerRan {
		t.Error("Finalizer should have ran")
	}
}

func TestConversationHandlerIsMatchReturnsFalseOnEmptyUpdate(t *testing.T) {
	handler := NewConversationHandler([]string{"test"}, make([]ConversationHandlerFunc, 0), func(payload *HandlePayload, state []interface{}) {})
	if handler.isMatch(&HandlePayload{
		Update: tgbotapi.Update{},
	}) {
		t.Error("IsMatch should handle empty update")
	}
}

func TestConversationCallbackQuery(t *testing.T) {
	finalizerRan := false
	handler := NewConversationHandler(
		[]string{"test"},
		[]ConversationHandlerFunc{
			func(p *HandlePayload, _ []interface{}) (interface{}, bool) {
				if p.Update.CallbackQuery == nil {
					t.Error("Should have callbackQuery here")
				}

				if p.Update.CallbackQuery.Data != "test" {
					t.Error("Should have test in data here")
				}

				return "test", true
			},
		},
		func(_ *HandlePayload, state []interface{}) {
			if state[0] != "test" {
				t.Error("Should have test in state from handler")
			}
			finalizerRan = true
		},
	)

	update := tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			Data: "test",
			From: &tgbotapi.User{ID: 1},
		},
	}

	state := make([]interface{}, 0)
	handler.instances.Store(1, conversationHandlerInstance{
		state: &state,
		lock:  &sync.Mutex{},
	})

	handler.handle(&HandlePayload{Update: update})

	if !finalizerRan {
		t.Error("Finalizer should have ran here")
	}
}

type SplitMiddlewareTestPayload struct {
	Middlewares   []Middleware
	ExpectedSync  []Middleware
	ExpectedAsync []Middleware
}

func TestSplitMiddleware(t *testing.T) {
	asyncMiddleware := Middleware{
		IsSync: false,
	}
	syncMiddleware := Middleware{
		IsSync: true,
	}

	payloads := []SplitMiddlewareTestPayload{
		{
			Middlewares:   []Middleware{asyncMiddleware},
			ExpectedSync:  make([]Middleware, 0),
			ExpectedAsync: []Middleware{asyncMiddleware},
		},
		{
			Middlewares:   []Middleware{asyncMiddleware, syncMiddleware},
			ExpectedSync:  []Middleware{syncMiddleware},
			ExpectedAsync: []Middleware{asyncMiddleware},
		},
		{
			Middlewares:   []Middleware{asyncMiddleware, asyncMiddleware, syncMiddleware, syncMiddleware, asyncMiddleware, syncMiddleware, asyncMiddleware},
			ExpectedSync:  []Middleware{syncMiddleware, syncMiddleware, syncMiddleware},
			ExpectedAsync: []Middleware{asyncMiddleware, asyncMiddleware, asyncMiddleware, asyncMiddleware},
		},
		{
			Middlewares:   make([]Middleware, 0),
			ExpectedSync:  make([]Middleware, 0),
			ExpectedAsync: make([]Middleware, 0),
		},
	}

	for _, p := range payloads {
		sync, async := splitMiddleware(p.Middlewares)

		if len(sync) != len(p.ExpectedSync) {
			t.Error("Not same length")
		}

		for _, es := range p.ExpectedSync {
			found := false
			for _, s := range sync {
				if reflect.DeepEqual(es, s) {
					found = true
					break
				}
			}

			if !found {
				t.Error("Expected not in actual slice")
			}
		}

		if len(async) != len(p.ExpectedAsync) {
			t.Error("Not same length")
		}

		for _, ea := range p.ExpectedAsync {
			found := false
			for _, a := range async {
				if reflect.DeepEqual(ea, a) {
					found = true
					break
				}
			}

			if !found {
				t.Error("Expected not in actual slice")
			}
		}
	}
}
