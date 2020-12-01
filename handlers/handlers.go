package handlers

import (
	"errors"
	"fmt"
	"log"
	"strconv"

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
	if err := db.Joins("Lesson").Where("user_id = ?", p.User.ID).Find(&notis).Error; err != nil {
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

		id, err := strconv.Atoi(args[0])
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
