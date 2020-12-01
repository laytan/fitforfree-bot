package handlers

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/laytan/go-fff-notifications-bot/bot"
	"github.com/laytan/go-fff-notifications-bot/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type mockSender struct {
	OnSend func(tgbotapi.Chattable)
}

func (m mockSender) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	m.OnSend(c)
	return tgbotapi.Message{}, nil
}

func getDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("../database/test.sqlite"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&database.User{}, &database.Noti{}, &database.Lesson{})

	clearDB(db)

	return db
}

func clearDB(db *gorm.DB) {
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM notis")
	db.Exec("DELETE FROM lessons")
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

func TestClearHandler(t *testing.T) {
	db := getDB()

	db.Create(&database.User{ID: 1})
	db.Create(&[]database.Noti{
		{
			UserID: 2,
		},
		{
			UserID: 1,
		},
	})

	ClearHandler(db)(&bot.HandlePayload{User: database.User{ID: 1}}, []string{})

	notis := []database.Noti{}
	db.Find(&notis)

	for _, noti := range notis {
		if noti.UserID == 1 {
			t.Error("Found noti with user id that was cleared")
		}
	}
}

type testRemoveHandlerPayload struct {
	User                 database.User
	Notis                []database.Noti
	Args                 []string
	ExpectedResultSubstr string
}

func TestRemoveHandler(t *testing.T) {
	db := getDB()
	handler := RemoveHandler(db)

	user := database.User{ID: 1}

	adminChatID := uint(111111)
	os.Setenv("ADMIN_CHAT_ID", fmt.Sprintf("%d", adminChatID))
	adminUser := database.User{ID: 1, ChatID: adminChatID}

	payloads := []testRemoveHandlerPayload{
		{
			User:                 user,
			Notis:                []database.Noti{},
			Args:                 []string{},
			ExpectedResultSubstr: "nummer van de notificatie",
		},
		{
			User:                 user,
			Notis:                []database.Noti{},
			Args:                 []string{"1"},
			ExpectedResultSubstr: "geen notificatie",
		},
		{
			User:                 user,
			Notis:                []database.Noti{},
			Args:                 []string{"test"},
			ExpectedResultSubstr: "niet goed ingevuld",
		},
		{
			User: user,
			Notis: []database.Noti{
				{
					Model:  gorm.Model{ID: 123123},
					UserID: 4,
				},
			},
			Args:                 []string{"123123"},
			ExpectedResultSubstr: "Je kunt deze notificatie niet verwijderen",
		},
		{
			User: adminUser,
			Notis: []database.Noti{
				{
					Model:  gorm.Model{ID: 123123},
					UserID: 4,
				},
			},
			Args:                 []string{"123123"},
			ExpectedResultSubstr: "verwijderd",
		},
		{
			User: user,
			Notis: []database.Noti{
				{
					Model:  gorm.Model{ID: 123123},
					UserID: user.ID,
				},
			},
			Args:                 []string{"123123"},
			ExpectedResultSubstr: "verwijderd",
		},
	}

	for _, payload := range payloads {
		sender := mockSender{
			OnSend: func(msg tgbotapi.Chattable) {
				message := msg.(tgbotapi.MessageConfig)
				if !strings.Contains(message.Text, payload.ExpectedResultSubstr) {
					t.Errorf("%s does not contain %s", message.Text, payload.ExpectedResultSubstr)
				}
			},
		}
		update := newMockCommandUpdate("/remove", strings.Join(payload.Args, " "))
		update.Message.Chat = &tgbotapi.Chat{
			ID: 1,
		}

		if len(payload.Notis) > 0 {
			if err := db.Create(&payload.Notis).Error; err != nil {
				t.Error(err)
			}
		}

		handlePayload := bot.HandlePayload{
			User:   payload.User,
			Update: update,
			Bot:    sender,
		}

		handler(&handlePayload, payload.Args)

		clearDB(db)
	}
}

type testListNotisHandlerPayload struct {
	Executor database.User
	Notis    []database.Noti
}

func TestListNotisHandler(t *testing.T) {
	db := getDB()
	handler := ListNotisHandler(db)

	adminChatID := uint(4)
	os.Setenv("ADMIN_CHAT_ID", fmt.Sprintf("%d", adminChatID))

	payloads := []testListNotisHandlerPayload{
		// Normal user getting message
		{
			Executor: database.User{ID: 1},
			Notis: []database.Noti{
				{
					User:   database.User{ID: 1},
					Model:  gorm.Model{ID: 1},
					Lesson: database.Lesson{ID: "1"},
				},
			},
		},
		// Admin should get all messages
		{
			Executor: database.User{ID: 1, ChatID: adminChatID},
			Notis: []database.Noti{
				{
					User:   database.User{ID: 1},
					Model:  gorm.Model{ID: 1},
					Lesson: database.Lesson{ID: "1"},
				},
				{
					User:   database.User{ID: 2},
					Model:  gorm.Model{ID: 2},
					Lesson: database.Lesson{ID: "2"},
				},
				{
					User:   database.User{ID: 3},
					Model:  gorm.Model{ID: 5},
					Lesson: database.Lesson{ID: "123"},
				},
			},
		},
		// Normal user should not get any messages here
		{
			Executor: database.User{ID: 1},
			Notis: []database.Noti{
				{
					User:   database.User{ID: 2},
					Model:  gorm.Model{ID: 1},
					Lesson: database.Lesson{ID: "1"},
				},
				{
					User:   database.User{ID: 3},
					Model:  gorm.Model{ID: 2},
					Lesson: database.Lesson{ID: "2"},
				},
				{
					User:   database.User{ID: 4},
					Model:  gorm.Model{ID: 5},
					Lesson: database.Lesson{ID: "123"},
				},
			},
		},
		// Normal user should only get it's messages
		{
			Executor: database.User{ID: 1},
			Notis: []database.Noti{
				{
					User:   database.User{ID: 2},
					Model:  gorm.Model{ID: 1},
					Lesson: database.Lesson{ID: "1"},
				},
				{
					User:   database.User{ID: 1},
					Model:  gorm.Model{ID: 2},
					Lesson: database.Lesson{ID: "2"},
				},
				{
					User:   database.User{ID: 4},
					Model:  gorm.Model{ID: 5},
					Lesson: database.Lesson{ID: "123"},
				},
			},
		},
		// empty admin
		{
			Executor: database.User{ID: 1, ChatID: adminChatID},
			Notis:    []database.Noti{},
		},
		// empty normal
		{
			Executor: database.User{ID: 1},
			Notis:    []database.Noti{},
		},
	}

	for _, payload := range payloads {
		sender := mockSender{
			OnSend: func(msg tgbotapi.Chattable) {
				message := msg.(tgbotapi.MessageConfig)
				for _, noti := range payload.Notis {
					inMessage := strings.Contains(message.Text, fmt.Sprintf("Nummer: %d", noti.ID))
					if payload.Executor.Admin() && !inMessage {
						t.Error("Admin should have all messages")
					}

					if !payload.Executor.Admin() {
						if payload.Executor.ID == noti.User.ID && !inMessage {
							t.Error("Not in message when it should")
						}

						if inMessage && payload.Executor.ID != noti.User.ID {
							t.Error("In message when it should not be")
						}
					}
				}
			},
		}
		update := newMockCommandUpdate("/notifications", "")
		update.Message.Chat = &tgbotapi.Chat{
			ID: 1,
		}

		if len(payload.Notis) > 0 {
			if err := db.Create(&payload.Notis).Error; err != nil {
				t.Error(err)
			}
		}

		handlePayload := bot.HandlePayload{
			User:   payload.Executor,
			Update: update,
			Bot:    sender,
		}

		handler(&handlePayload, []string{})

		clearDB(db)
	}
}

func TestStartNotiHandler(t *testing.T) {
	handlePayload := bot.HandlePayload{
		Bot: mockSender{
			OnSend: func(msg tgbotapi.Chattable) {
				message := msg.(tgbotapi.MessageConfig)
				if !strings.Contains(message.Text, "welke datum") {
					t.Error("Did not get which date message")
				}
			},
		},
	}

	_, continueConv := StartNotiHandler(&handlePayload, nil)
	if continueConv != true {
		t.Error("Should continue conv")
	}
}

func TestDateNotiHandler(t *testing.T) {
	var called uint
	handlePayload := bot.HandlePayload{
		Bot: mockSender{
			OnSend: func(msg tgbotapi.Chattable) {
				called++
				message := msg.(tgbotapi.MessageConfig)
				if called == 1 {
					if !strings.Contains(message.Text, "Groepsles of vrije les") {
						t.Error("Did not get which type")
					}
				}
				if called == 2 {
					if !strings.Contains(message.Text, "geldige datum in") {
						t.Error("Did not get invalid date msg")
					}
				}
			},
		},
	}
	handlePayload.Update = tgbotapi.Update{
		Message: &tgbotapi.Message{
			Text: "04-12-2020",
			Chat: &tgbotapi.Chat{
				ID: 1,
			},
		},
	}

	date, continueConv := DateNotiHandler(&handlePayload, nil)
	if !continueConv {
		t.Error("Should continue conv here")
	}
	timeObj, ok := date.(time.Time)
	if !ok {
		t.Error("Can't get date back")
	}

	if timeObj.Day() != 4 || timeObj.Month() != 12 || timeObj.Year() != 2020 {
		t.Error("Invalid date back")
	}

	// make payload invalid
	handlePayload.Update.Message.Text = "04-13-2020"
	_, continueConv = DateNotiHandler(&handlePayload, nil)
	if continueConv {
		t.Error("Should not continue conv")
	}

}

func TestTypeNotiHandlerDoesNotRespondToEmptyOrMessage(t *testing.T) {
	handlePayload := bot.HandlePayload{
		Bot: mockSender{
			OnSend: func(msg tgbotapi.Chattable) {
				message := msg.(tgbotapi.MessageConfig)

				if !strings.Contains(message.Text, "Groepsles of Vrij") {
					t.Error("Did not get which type")
				}
			},
		},
	}

	_, continueConv := TypeNotiHandler(&handlePayload, nil)
	if continueConv {
		t.Error("Should not continue conv")
	}

	handlePayload.Update.Message = &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 1}}

	_, continueConv = TypeNotiHandler(&handlePayload, nil)
	if continueConv {
		t.Error("Should not continue conv")
	}

	handlePayload.Update.Message = nil
	handlePayload.Update.CallbackQuery = &tgbotapi.CallbackQuery{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 1}}}

	_, continueConv = TypeNotiHandler(&handlePayload, nil)
	if continueConv {
		t.Error("Should not continue conv")
	}

	handlePayload.Update.CallbackQuery.Data = "blablabla"
	_, continueConv = TypeNotiHandler(&handlePayload, nil)
	if continueConv {
		t.Error("Should not continue conv")
	}
}
