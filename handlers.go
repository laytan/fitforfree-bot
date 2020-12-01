package main

import (
	"errors"
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

// HelpHandler responds with the a message
func HelpHandler(p *bot.HandlePayload, _ []string) {
	p.Respond(
		`
		Te gebruiken commandos:
		- /noti: Start een gesprek om een nieuwe notificatie toe te voegen
		- /notifications: Verkrijg een lijst met alle ingestelde notificaties
		- /clear: Verwijder al je notificaties
		- /remove {nummer}: Verwijder de notificatie met het gegeven nummer 
		`,
	)
}

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

	withoutAvailable := fitforfree.Filter(filteredTypes, func(l fitforfree.Lesson) bool {
		return l.SpotsAvailable == 0
	})

	msg := ""
	for i, lesson := range withoutAvailable {
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

// ListNotisHandler forwards the update to either ListNotisAdminHandler or ListNotisNormalHandler based on the update's user
func ListNotisHandler(db *gorm.DB) func(*bot.HandlePayload, []string) {
	return func(p *bot.HandlePayload, _ []string) {
		if p.User.Admin() {
			ListNotisAdminHandler(db, p)
			return
		}

		ListNotisNormalHandler(db, p)
	}
}

// ListNotisAdminHandler lists all the notis in the database
// NOTE: Should only be used when the payload has an admin user
func ListNotisAdminHandler(db *gorm.DB, p *bot.HandlePayload) {
	if !p.User.Admin() {
		panic("Called ListNotisAdminHandler without admin user")
	}

	msg := ""

	notis := make([]database.Noti, 0)
	if err := db.Joins("User").Joins("Lesson").Find(&notis).Error; err != nil {
		log.Printf("ERROR: Error retrieving all notis from the database for ListNotisAdminHandler, err: %+v", err)
		p.Respond("Er ging iets fout, probeer het opnieuw")
		return
	}

	if len(notis) == 0 {
		p.Respond("Geen notificaties gevonden")
		return
	}

	for _, noti := range notis {
		msg += formatNoti(noti, true)
	}

	p.Respond(msg)
}

// ListNotisNormalHandler lists all the user's notis
func ListNotisNormalHandler(db *gorm.DB, p *bot.HandlePayload) {
	msg := ""

	notis := make([]database.Noti, 0)
	if err := db.Joins("Lesson").Find(&notis).Error; err != nil {
		log.Printf("ERROR: Error retrieving users noti's, user: %+v, err: %+v", p.User, err)
		p.Respond("Er ging iets fout, probeer het opnieuw")
		return
	}

	if len(notis) == 0 {
		p.Respond("Geen notificaties gevonden.")
		return
	}

	for _, noti := range notis {
		msg += formatNoti(noti, false)
	}

	p.Respond(msg)
}

// RemoveHandler removes the noti specified if the user is allowed to
func RemoveHandler(db *gorm.DB) func(*bot.HandlePayload, []string) {
	return func(p *bot.HandlePayload, args []string) {
		if len(args) != 1 {
			p.Respond("Stuur het nummer van de notificatie die verwijdert moet worden mee, zoals: /remove 1")
			return
		}

		id, err := strconv.Atoi(args[1])
		if err != nil {
			p.Respond("Nummer is niet goed ingevuld")
			return
		}

		noti := database.Noti{}
		err = db.First(&noti, id).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				p.Respond("Er bestaat geen notificatie met dat nummer")
				return
			}
			p.Respond("Er ging iets fout bij het ophalen van de notificatie, probeer het opnieuw.")
			log.Printf("ERROR: Error retrieving noti in RemoveHandler, err: %+v", err)
			return
		}

		if noti.UserID != p.User.ID && !p.User.Admin() {
			p.Respond("Je kunt deze notificatie niet verwijderen omdat deze door iemand anders is gemaakt")
			return
		}

		if err := db.Delete(&noti).Error; err != nil {
			p.Respond("Er ging iets fout bij het verwijderen van de notificatie, probeer het opnieuw.")
			log.Printf("ERROR: Error when removing noti, err: %+v", err)
			return
		}

		p.Respond("Notificatie verwijderd")
	}
}

// ClearHandler removes all noti's from a user
func ClearHandler(db *gorm.DB) func(*bot.HandlePayload, []string) {
	return func(p *bot.HandlePayload, _ []string) {
		db.Where("user_id = ?", p.User.ID).Delete(&database.Noti{})
		p.Respond("Notificaties verwijderd")
	}
}

func formatLesson(lesson fitforfree.Lesson, id uint) string {
	return fmt.Sprintf(`
		Nummer: %d
		Activiteit: %s
		Start: %s
		Eind: %s`,
		id,
		lesson.Activity.Name,
		times.FormatTimestamp(lesson.StartTimestamp, times.TimeLayout),
		times.FormatTimestamp(lesson.StartTimestamp+lesson.DurationSeconds, times.TimeLayout),
	)
}

// formatNoti formats a notification for display
func formatNoti(noti database.Noti, withName bool) string {
	var msg string
	if withName {
		msg = fmt.Sprintf("Naam: %s", noti.User.Name)
	} else {
		msg = ""
	}

	msg += fmt.Sprintf(`
		Nummer: %d
		Datum: %s
		Start: %s
		Eind: %s
		Gemaakt: %s
	`,
		noti.ID,
		times.FormatTimestamp(uint(noti.Lesson.Start), times.DateLayout),
		times.FormatTimestamp(uint(noti.Lesson.Start), times.TimeLayout),
		times.FormatTimestamp(uint(noti.Lesson.Start+noti.Lesson.DurationSeconds), times.TimeLayout),
		times.FormatTimestamp(uint(noti.CreatedAt.Unix()), times.FullLayout))

	return msg
}
