package checker

import (
	"log"

	"github.com/laytan/go-fff-notifications-bot/database"
	"github.com/laytan/go-fff-notifications-bot/fitforfree"
	"gorm.io/gorm"
)

func AvailabilityCheck(db *gorm.DB, venues []string, bearerToken string, availableChan chan database.Noti) {
	// Get timeframe to get lessons for
	start, end, notis := getCheckTimeframe(db)
	if len(notis) == 0 {
		return
	}

	// Get lessons from fitforfree to check
	lessons := fitforfree.GetLessons(start, end, venues, bearerToken)
	lessons = filterUnavailable(lessons)

	// Get notis that are now available
	availables := filterNotNeeded(lessons, notis)
	if len(availables) > 0 {
		primaryKeys := []uint{}
		for _, a := range availables {
			primaryKeys = append(primaryKeys, a.ID)
			// Get the user
			if err := db.Model(&a).Association("User").Find(&a.User); err != nil {
				log.Printf("ERROR: No user for noti, which should not happen: %+v", err)
				break
			}
			availableChan <- a
		}

		// Delete notis because they are handled
		if err := db.Delete(&database.Noti{}, primaryKeys).Error; err != nil {
			log.Printf("ERROR: Error deleting handled notis: %+v", err)
		}
	}
}

// Gets the lowest start timestamp and the highest end timestamp of all notis in the db
func getCheckTimeframe(db *gorm.DB) (uint, uint, []database.Noti) {
	notis := make([]database.Noti, 0)
	db.Joins("Lesson").Find(&notis)

	if len(notis) == 0 {
		return 0, 0, notis
	}

	start := notis[0].Lesson.Start
	end := notis[0].Lesson.Start + notis[0].Lesson.DurationSeconds
	for _, noti := range notis[1:] {
		if start > noti.Lesson.Start {
			start = noti.Lesson.Start
		}

		if end < noti.Lesson.Start+noti.Lesson.DurationSeconds {
			end = noti.Lesson.Start + noti.Lesson.DurationSeconds
		}
	}

	return start, end, notis
}

// Filters out all lessons that are unavailable
func filterUnavailable(lessons []fitforfree.Lesson) []fitforfree.Lesson {
	// amt of lessons to return
	var amt uint
	for _, lesson := range lessons {
		if lesson.SpotsAvailable > 0 {
			// keep
			lessons[amt] = lesson
			amt++
		}

	}
	return lessons[:amt]
}

// Filters out all lessons we don't have notis for
func filterNotNeeded(lessons []fitforfree.Lesson, notis []database.Noti) []database.Noti {
	var n uint

	for _, lesson := range lessons {
		for _, noti := range notis {
			if noti.Lesson.ID == lesson.ID {
				notis[n] = noti
				n++

				// Noti has found it's lesson so we can break
				break
			}
		}
	}

	return notis[:n]
}
