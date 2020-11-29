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
	"gorm.io/gorm"
)

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

func StartNotiHandler(p *bot.HandlePayload, _ []interface{}) (interface{}, bool) {
	p.Respond("Hier gaan we, welke datum wil je sporten? (d-m-yyyy)")
	return nil, true
}

func DateNotiHandler(p *bot.HandlePayload, _ []interface{}) (interface{}, bool) {
	date, err := time.Parse("2-1-2006", p.Update.Message.Text)
	if err != nil {
		p.Respond("Vul een geldige datum in, bijvoorbeeld 30-1-2020.")
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

func TypeNotiHandler(p *bot.HandlePayload, s []interface{}) (interface{}, bool) {
	if p.Update.CallbackQuery == nil || !(p.Update.CallbackQuery.Data == "group_lesson|mixed_lesson" || p.Update.CallbackQuery.Data == "free_practise") {
		p.Respond("Kies aub Groepsles of Vrij.")
		return nil, false
	}
	classType := p.Update.CallbackQuery.Data

	selectedDate, ok := s[1].(time.Time)
	if !ok {
		p.Respond("Geen goede datum ingevuld")
		// TODO: End conversation or send back to date picking
	}

	lessons := fitforfree.GetLessons(uint(selectedDate.Unix()), uint(selectedDate.Add(time.Duration(time.Hour*24)).Unix()), []string{os.Getenv("VENUE")}, os.Getenv("FIT_FOR_FREE_TOKEN"))
	filteredTypes := fitforfree.Filter(lessons, func(lesson fitforfree.Lesson) bool {
		// TODO: Make classType type with check func built in
		// TODO: Filter out non-empty lessons
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
		// TODO: format timestamps
		msg += fmt.Sprintf(`
		Nummer: %d
		Instucteur: %s
		Activiteit: %s
		Start: %d
		Eind: %d`, i, lesson.Instructor, lesson.Activity.Name, lesson.StartTimestamp, lesson.StartTimestamp+lesson.DurationSeconds)
	}

	p.Respond(fmt.Sprintf("Welk les nummer wil je in de gaten houden? Hier zijn ze allemaal: %s", msg))
	return filteredTypes, true
}

func ClassNotiHandler(p *bot.HandlePayload, s []interface{}) (interface{}, bool) {
	num, err := strconv.Atoi(p.Update.Message.Text)
	if err != nil {
		p.Respond("Ongeldig nummer, probeer opnieuw.")
		return nil, false
	}

	lessons, ok := s[2].([]fitforfree.Lesson)
	if !ok {
		p.Respond("Er ging iets fout, probeer opnieuw.")
		log.Println("ERROR: Can not convert state back to lessons slice")
		return nil, false
	}

	// Uint so - doesn't work
	if uint(num) >= uint(len(lessons)) {
		p.Respond("Geen les met dat nummer gevonden, probeer opnieuw.")
		return nil, false
	}

	return lessons[num], true
}

// NotiHandler first type casts the conversation's state, then validates the times entered and finally inserts a new noti into the db
func NotiHandler(db *gorm.DB) bot.ConversationFinalizerFunc {
	return func(p *bot.HandlePayload, s []interface{}) {
		lesson, ok := s[3].(fitforfree.Lesson)
		if !ok {
			p.Respond("Er ging iets fout bij het aanmaken van de notificatie.")
			log.Printf("ERROR: Can't assert type fitforfree.lesson on given lesson: %+v", s[3])
			return
		}

		startTimestamp := lesson.StartTimestamp
		endTimeStamp := lesson.StartTimestamp + lesson.DurationSeconds

		if startTimestamp < uint(time.Now().Unix()) {
			p.Respond("Je kan alleen tijden in de toekomst toevoegen, probeer opnieuw")
			return
		}

		noti := database.Noti{
			UserID:    p.User.ID,
			Start:     uint64(startTimestamp),
			End:       uint64(endTimeStamp),
			ClassType: lesson.ClassType,
		}

		if err := db.Create(&noti).Error; err != nil {
			p.Respond("Er ging iets fout bij het aanmaken van de notificatie.")
			log.Printf("ERROR: error on noti creation: %+v", err)
			return
		}

		p.Respond("Notificatie aangezet.")
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
	if err := db.Preload("User").Find(&notis).Error; err != nil {
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
	if err := db.Model(&p.User).Association("Notis").Find(&notis); err != nil {
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

// formatNoti formats a notification for display
func formatNoti(noti database.Noti, withName bool) string {
	startDate := time.Unix(int64(noti.Start), 0)
	end := time.Unix(int64(noti.End), 0).Format("15:04")

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
	`, noti.ID, startDate.Format("2-1-2006"), startDate.Format("15:04"), end, noti.CreatedAt.Format("2-1-2006 15:04"))

	return msg
}

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

func ClearHandler(db *gorm.DB) func(*bot.HandlePayload, []string) {
	return func(p *bot.HandlePayload, _ []string) {
		db.Where("user_id = ?", p.User.ID).Delete(&database.Noti{})
		p.Respond("Notificaties verwijderd")
	}
}
