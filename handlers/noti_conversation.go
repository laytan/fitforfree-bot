package handlers

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/laytan/go-fff-notifications-bot/bot"
	"github.com/laytan/go-fff-notifications-bot/database"
	"github.com/laytan/go-fff-notifications-bot/fitforfree"
	"github.com/laytan/go-fff-notifications-bot/times"
	"gorm.io/gorm"
)

// StartNotiHandler asks for the date of the new notification
func StartNotiHandler(p *bot.HandlePayload, _ []interface{}) (interface{}, bool) {
	p.Respond("Hier gaan we, welke datum wil je sporten? (d-m-yyyy)")
	return nil, true
}

// DateNotiHandler validates the date entered and asks for the type of lesson for the notification
func DateNotiHandler(p *bot.HandlePayload, _ []interface{}) (interface{}, bool) {
	date, err := times.FromInput(p.Update.Message.Text, times.DateLayout)
	if err != nil {
		p.Respond(fmt.Sprintf("Vul een geldige datum in, bijvoorbeeld %s.", times.DateLayout))
		return nil, false
	}

	msg := tgbotapi.NewMessage(p.Update.Message.Chat.ID, "Groepsles of vrije les?")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Groepsles", "group_lesson|mixed_lesson"),
			tgbotapi.NewInlineKeyboardButtonData("Vrij", "free_practise"),
		),
	)
	p.Bot.Send(msg)

	return date, true
}

// TypeNotiHandler validates the type entered and shows all lessons a notification can be added to asking for the number of the lesson they want to track
func TypeNotiHandler(p *bot.HandlePayload, s []interface{}) (interface{}, bool) {
	if p.Update.CallbackQuery == nil || !(p.Update.CallbackQuery.Data == "group_lesson|mixed_lesson" || p.Update.CallbackQuery.Data == "free_practise") {
		p.Respond("Kies aub Groepsles of Vrij.")
		return nil, false
	}
	classType := p.Update.CallbackQuery.Data

	selectedDate := s[1].(time.Time)

	lessons := fitforfree.GetLessons(uint(selectedDate.Unix()), uint(selectedDate.Add(time.Duration(time.Hour*24)).Unix()), []string{os.Getenv("VENUE")}, os.Getenv("FIT_FOR_FREE_TOKEN"))
	filteredTypes := fitforfree.Filter(lessons, func(lesson fitforfree.Lesson) bool {
		if strings.Contains(classType, "|") {
			types := strings.Split(classType, "|")
			for _, t := range types {
				if t == lesson.ClassType {
					return true
				}
			}
		} else if classType == lesson.ClassType {
			return true
		}
		return false
	})

	msg := ""
	for i, lesson := range filteredTypes {
		msg += formatLesson(lesson, uint(i))
	}

	p.Respond(fmt.Sprintf("Welk les nummer wil je in de gaten houden? Hier zijn ze allemaal: %s", msg))
	return filteredTypes, true
}

// ClassNotiHandler gets the lesson for the entered and validates it
func ClassNotiHandler(p *bot.HandlePayload, s []interface{}) (interface{}, bool) {
	num, err := strconv.Atoi(p.Update.Message.Text)
	if err != nil {
		p.Respond("Ongeldig nummer, probeer opnieuw.")
		return nil, false
	}

	lessons := s[2].([]fitforfree.Lesson)

	// Uint so minus doesn't work
	if uint(num) >= uint(len(lessons)) {
		p.Respond("Geen les met dat nummer gevonden, probeer opnieuw.")
		return nil, false
	}

	return uint(num), true
}

// NotiHandler adds a new noti based on the conversations state
func NotiHandler(db *gorm.DB) bot.ConversationFinalizerFunc {
	return func(p *bot.HandlePayload, s []interface{}) {
		num := s[3].(uint)
		lesson := s[2].([]fitforfree.Lesson)[num]

		if lesson.StartTimestamp < uint(time.Now().Unix()) {
			p.Respond("Je kan alleen tijden in de toekomst toevoegen, probeer opnieuw")
			return
		}

		if err := database.CreateNoti(db, p.User, lesson); err != nil {
			p.Respond("Er ging iets fout bij het toevoegen van de noti.")
			log.Printf("ERROR: Error creating noti, error: %+v", err)
			return
		}

		p.Respond(
			fmt.Sprintf(
				`%s
				%s`,
				"Notificatie aangezet voor les:",
				formatLesson(lesson, num),
			),
		)
	}
}
