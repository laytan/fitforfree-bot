package main

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/laytan/go-fff-notifications-bot/bot"
	"github.com/laytan/go-fff-notifications-bot/database"
)

func HelpHandler(p *bot.HandlePayload, _ []string) {
	msg := tgbotapi.NewMessage(p.Update.Message.Chat.ID, fmt.Sprintf("Geen stress %s", p.User.Name))
	p.Bot.Send(msg)
}

func StartNotiHandler(p *bot.HandlePayload) (interface{}, bool) {
	p.Respond("Hier gaan we, welke datum wil je sporten?")
	return nil, true
}

func DateNotiHandler(p *bot.HandlePayload) (interface{}, bool) {
	date, err := time.Parse("2-1-2006", p.Update.Message.Text)
	if err != nil {
		p.Respond("Vul een geldige datum in, bijvoorbeeld 30-1-2020.")
		return nil, false
	}

	p.Respond("Oke, hoelaat wil je beginnen?")
	return date, true
}

func StartTimeNotiHandler(p *bot.HandlePayload) (interface{}, bool) {
	time, err := time.Parse("15:04", p.Update.Message.Text)
	if err != nil {
		p.Respond("Vul een geldige tijd in, bijvoorbeeld 23:02.")
		return nil, false
	}

	p.Respond("Oke, tot hoelaat?")
	return time, true
}

func EndTimeNotiHandler(p *bot.HandlePayload) (interface{}, bool) {
	time, err := time.Parse("15:04", p.Update.Message.Text)
	if err != nil {
		p.Respond("Vul een geldige tijd in, bijvoorbeeld 23:02.")
		return nil, false
	}

	p.Respond("Oke, welk type les wil je volgen (vrij, groep of beide)?")
	return time, true
}

func TypeNotiHandler(p *bot.HandlePayload) (interface{}, bool) {
	// TODO: Extract possible types
	if (p.Update.Message.Text == "vrij") || (p.Update.Message.Text == "groep") || (p.Update.Message.Text == "beide") {
		p.Respond("Geweldig")
		return p.Update.Message.Text, true
	}
	p.Respond("Vul vrij, groep of beide in.")
	return nil, false
}

func NotiHandler(db *database.Database) bot.ConversationFinalizerFunc {
	return func(p *bot.HandlePayload, s []bot.ConversationState) {
		date, ok := s[1].Value.(time.Time)
		if !ok {
			p.Respond("Er ging iets fout bij het aanmaken van de notificatie.")
			log.Printf("ERROR: Can't assert type time.Time on given date: %+v", s[1])
			return
		}

		start, ok := s[2].Value.(time.Time)
		if !ok {
			p.Respond("Er ging iets fout bij het aanmaken van de notificatie.")
			log.Printf("ERROR: Can't assert type time.Time on given start time: %+v", s[2])
			return
		}

		end, ok := s[3].Value.(time.Time)
		if !ok {
			p.Respond("Er ging iets fout bij het aanmaken van de notificatie.")
			log.Printf("ERROR: Can't assert type time.Time on given end time: %+v", s[3])
			return
		}

		classType, ok := s[4].Value.(string)
		if !ok {
			p.Respond("Er ging iets fout bij het aanmaken van de notificatie.")
			log.Printf("ERROR: Can't assert type time.Time on given type: %+v", s[4])
			return
		}

		startTimestamp := date.Add(time.Hour*time.Duration(start.Hour()) + time.Minute*time.Duration(start.Minute())).Unix()
		endTimeStamp := date.Add(time.Hour*time.Duration(end.Hour()) + time.Minute*time.Duration(end.Minute())).Unix()

		noti := database.Noti{
			UserID:    p.User.ID,
			Start:     uint64(startTimestamp),
			End:       uint64(endTimeStamp),
			ClassType: classType,
		}

		db.Mutex.Lock()
		defer db.Mutex.Unlock()

		if err := db.Conn.Create(&noti).Error; err != nil {
			p.Respond("Er ging iets fout bij het aanmaken van de notificatie.")
			log.Printf("ERROR: error on noti creation: %+v", err)
			return
		}

		p.Respond("Notificatie aangezet.")
	}
}

func ListNotisHandler(db *database.Database) func(*bot.HandlePayload, []string) {
	return func(p *bot.HandlePayload, args []string) {
		msg := ""

		db.Mutex.Lock()
		defer db.Mutex.Unlock()

		notis := make([]database.Noti, 0)
		db.Conn.Model(&p.User).Association("Notis").Find(&notis)

		if len(notis) == 0 {
			p.Respond("Geen notificaties gevonden.")
			return
		}

		for _, noti := range notis {
			startDate := time.Unix(int64(noti.Start), 0)
			end := time.Unix(int64(noti.End), 0).Format("15:04")
			msg += fmt.Sprintf(`
			Nummer: %d
			Datum: %s
			Start: %s
			Eind: %s
			Gemaakt: %s
			`, noti.ID, startDate.Format("2-1-2006"), startDate.Format("15:04"), end, noti.CreatedAt.Format("2-1-2006 15:04"))
		}

		p.Respond(msg)
	}
}
