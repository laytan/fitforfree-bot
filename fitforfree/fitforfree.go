package fitforfree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

// TODO: Make all exported functions return errors

type LessonResponse struct {
	Status Status
	Data   LessonResponseData
}

type Status struct {
	Code    uint
	Message string
}

type LessonResponseData struct {
	Lessons []Lesson
}

type Lesson struct {
	ID                     string
	VenueName              string
	VenueID                string
	StartTimestamp         uint
	PreCheckinTimestamp    uint
	PostCheckinTimestamp   uint
	DurationSeconds        uint
	Instructor             string
	Activity               Activity
	Status                 string
	Booked                 bool
	ClassType              string
	SpotsAvailable         uint8
	Capacity               uint8
	AvailabilityPercentage uint8 `json:"availability_percentage"`
	RoomName               string
}

func Filter(vs []Lesson, f func(Lesson) bool) []Lesson {
	vsf := make([]Lesson, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

type Activity struct {
	ID          string
	Name        string
	Category    string
	Description string
	ImageURL    string
}

type Venue struct {
	ID     string
	Name   string
	F500ID uint
}

type LoginResponse struct {
	Status Status
	Data   User
}

type User struct {
	Gender               string
	FirstName            string
	SurName              string
	NamePrefix           string
	Cellphone            string
	Phone                string
	Email                string
	MembershipNumber     string
	CardNumber           string
	MemberID             string
	MemberUUID           string
	CanBookGroupLessons  uint
	Expired              uint
	VenueID              string
	Picture              string
	PreferredVenueID     string
	SessionID            string
	F500Membership       interface{} `json:"f500_membership"`
	HasGroupLessons      bool        `json:"has_group_lessons"`
	HasYanga             bool        `json:"has_yanga"`
	ShowYangaBanner      bool        `json:"show_yanga_banner"`
	IsCheckedIn          bool        `json:"is_checked_in"`
	UserProfile          interface{} `json:"user_profile"`
	Proficiency          string
	WorkoutDuration      uint        `json:"workout_duration"`
	WantsClasses         string      `json:"wants_classes"`
	TrainingAvailability []string    `json:"training_availability"`
	WeightedMuscleGroups interface{} `json:"weighted_muscle_groups"`
	Weight               uint
	PreferredClub        interface{} `json:"preferred_club"`
	WeeklyTrainingGoal   uint        `json:"weekly_training_goal"`
}

const BASE_URL = "https://electrolyte.fitforfree.nl/"

func doGetRequest(url string, bearerToken string) ([]byte, uint) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", BASE_URL, url), nil)
	if err != nil {
		log.Panicf("ERROR: Error while creating request, err: %+v", err)
	}

	if len(bearerToken) > 0 {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	}

	res, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: Error while executing request, err: %+v", err)
		return []byte{}, 400
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return []byte{}, uint(res.StatusCode)
	}

	// Turn response into bytes
	buffer := new(bytes.Buffer)
	buffer.ReadFrom(res.Body)
	bytes := buffer.Bytes()

	return bytes, 200
}

func doPostRequest(url string, data interface{}, bearerToken string) ([]byte, uint) {
	client := &http.Client{}

	json, err := json.Marshal(data)
	if err != nil {
		log.Panicf("ERROR: Can not format data: %+v as json", data)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", BASE_URL, url), bytes.NewBuffer(json))
	if err != nil {
		log.Panicf("ERROR: Error while creating request, err: %+v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if len(bearerToken) > 0 {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	}

	res, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: Error while executing request, err: %+v", err)
		return []byte{}, 400
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return []byte{}, uint(res.StatusCode)
	}

	// Turn response into bytes
	buffer := new(bytes.Buffer)
	buffer.ReadFrom(res.Body)
	bytes := buffer.Bytes()

	return bytes, 200
}

func Login(memberID string, postalCode string) (*User, bool) {
	bytesRes, statusCode := doPostRequest("v0/login", map[string]interface{}{"memberid": memberID, "postcode": postalCode, "terms_accepted": true}, "")
	if statusCode != 200 {
		if statusCode == 401 {
			return new(User), false
		}
		log.Printf("ERROR: Status code not 200 but %d", statusCode)
		return new(User), false
	}

	res := new(LoginResponse)
	if err := json.Unmarshal(bytesRes, &res); err != nil {
		log.Printf("ERROR: Can't parse json response into user: %s, err: %+v", string(bytesRes), err)
		return new(User), false
	}

	return &res.Data, true
}

func GetAllVenues(bearerToken string) []Venue {
	resBytes, statusCode := doGetRequest("v1/venues", bearerToken)
	if statusCode != 200 {
		log.Printf("ERROR: Status code GetAllVenues is %d instead of 200", statusCode)
		return make([]Venue, 0)
	}

	venues := make([]Venue, 0)
	if err := json.Unmarshal(resBytes, &venues); err != nil {
		log.Printf("ERROR: Unable to unmarshal venues JSON: %s", string(resBytes))
	}
	return venues
}

func GetVenueByName(name string, bearerToken string) (*Venue, bool) {
	venues := GetAllVenues(bearerToken)
	for _, venue := range venues {
		if venue.Name == name {
			return &venue, true
		}
	}
	return new(Venue), false
}

// GetLessons gets available free fitness lessons between 2 timestamps
func GetLessons(start uint, end uint, venues []string, bearerToken string) []Lesson {
	lessons := make([]Lesson, 0)

	if start > end {
		log.Panic("ERROR: Start time is after end time in GetLessons")
		return lessons
	}

	venuesQuery := url.QueryEscape(fmt.Sprintf("%q", venues))

	bytesRes, statusCode := doGetRequest(fmt.Sprintf("v0/lessons/?venues=%s&from=%d&to=%d&language=%s", venuesQuery, start, end, "nl_NL"), bearerToken)

	if statusCode != 200 {
		log.Printf("ERROR: Status code while getting lessons is: %d when 200 is expected", statusCode)
		return lessons
	}

	// Turn bytes into map string to interface
	resJSON := new(LessonResponse)
	if err := json.Unmarshal(bytesRes, &resJSON); err != nil {
		log.Printf("ERROR: Json could not be parsed while getting lessons, err: %+v", err)
		return lessons
	}

	return resJSON.Data.Lessons
}
