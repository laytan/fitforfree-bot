package checker

import (
	"testing"

	"github.com/laytan/go-fff-notifications-bot/database"
	"github.com/laytan/go-fff-notifications-bot/fitforfree"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type getCheckTimeFramePayloads struct {
	notis    []database.Noti
	outStart uint
	outEnd   uint
}

func TestGetCheckTimeFrame(t *testing.T) {
	payloads := []getCheckTimeFramePayloads{
		{
			notis: []database.Noti{
				{
					Lesson: database.Lesson{
						Start:           1,
						DurationSeconds: 9,
					},
				},
			},
			outStart: 0,
			outEnd:   11,
		},
		{
			notis:    []database.Noti{},
			outStart: 0,
			outEnd:   0,
		},
		{
			notis: []database.Noti{
				{
					Lesson: database.Lesson{
						ID:              "0",
						Start:           10,
						DurationSeconds: 5,
					},
				},
				{
					Lesson: database.Lesson{
						ID:              "1",
						Start:           5,
						DurationSeconds: 5,
					},
				},
			},
			outStart: 4,
			outEnd:   16,
		},
		{
			notis: []database.Noti{
				{
					Lesson: database.Lesson{
						ID:              "0",
						Start:           10,
						DurationSeconds: 5,
					},
				},
				{
					Lesson: database.Lesson{
						ID:              "1",
						Start:           5,
						DurationSeconds: 20,
					},
				},
				{

					Lesson: database.Lesson{
						ID:              "2",
						Start:           5,
						DurationSeconds: 19,
					},
				},
			},
			outStart: 4,
			outEnd:   26,
		},
	}

	// Connect to db
	db, err := gorm.Open(sqlite.Open("../database/test.sqlite"), &gorm.Config{})
	if err != nil {
		t.Error(err)
	}

	if err := db.AutoMigrate(&database.Lesson{}, &database.Noti{}); err != nil {
		t.Error(err)
	}

	db.Exec("DELETE FROM lessons")
	db.Exec("DELETE FROM notis")

	for _, payload := range payloads {
		if len(payload.notis) > 0 {
			if err := db.Create(&payload.notis).Error; err != nil {
				t.Error(err)
			}
		}

		start, end, notis := getCheckTimeframe(db)
		if start != payload.outStart {
			t.Error("Start not the same")
		}

		if end != payload.outEnd {
			t.Error("End not the same")
		}

		if len(notis) != len(payload.notis) {
			t.Error("Notis not returned correctly")
		}

		db.Exec("DELETE FROM lessons")
		db.Exec("DELETE FROM notis")
	}
}

type filterUnavailablePayload struct {
	outLen    uint
	inLessons []fitforfree.Lesson
}

func TestFilterUnavailable(t *testing.T) {
	payloads := []filterUnavailablePayload{
		{
			outLen: 1,
			inLessons: []fitforfree.Lesson{
				{
					SpotsAvailable: 0,
				},
				{
					SpotsAvailable: 1,
				},
			},
		},
		{
			outLen: 3,
			inLessons: []fitforfree.Lesson{
				{
					SpotsAvailable: 1,
				},
				{
					SpotsAvailable: 0,
				},
				{
					SpotsAvailable: 100,
				},
				{
					SpotsAvailable: 255,
				},
				{
					SpotsAvailable: 0,
				},
			},
		},
		{
			outLen: 0,
			inLessons: []fitforfree.Lesson{
				{
					SpotsAvailable: 0,
				},
				{
					SpotsAvailable: 0,
				},
				{
					SpotsAvailable: 0,
				},
				{
					SpotsAvailable: 0,
				},
				{
					SpotsAvailable: 0,
				},
				{
					SpotsAvailable: 0,
				},
				{
					SpotsAvailable: 0,
				},
				{
					SpotsAvailable: 0,
				},
				{
					SpotsAvailable: 0,
				},
				{
					SpotsAvailable: 0,
				},
				{
					SpotsAvailable: 0,
				},
				{
					SpotsAvailable: 0,
				},
			},
		},
		{
			outLen: 3,
			inLessons: []fitforfree.Lesson{
				{
					SpotsAvailable: 1,
				},
				{
					SpotsAvailable: 1,
				},
				{
					SpotsAvailable: 1,
				},
			},
		},
	}

	for _, payload := range payloads {
		ls := filterUnavailable(payload.inLessons)

		if len(ls) != int(payload.outLen) {
			t.Error("Length does not match")
		}

		for _, lesson := range ls {
			if lesson.SpotsAvailable == 0 {
				t.Error("Let through a non available lesson")
			}
		}
	}
}
