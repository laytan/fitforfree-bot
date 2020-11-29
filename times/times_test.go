package times

import (
	"testing"
)

func TestFromInput(t *testing.T) {
	time, err := FromInput("11:11", TimeLayout)
	if err != nil {
		t.Error(err)
	}

	if time.Hour() != 11 {
		t.Error("Hour should be 11")
	}

	if time.Minute() != 11 {
		t.Error("Minute should be 11")
	}

	time, err = FromInput("16-03-2001", DateLayout)
	if err != nil {
		t.Error(err)
	}

	if time.Day() != 16 {
		t.Error("Day should be 16")
	}

	if time.Month() != 3 {
		t.Error("Month should be 3")
	}

	if time.Year() != 2001 {
		t.Error("Year should be 2001")
	}

	time, err = FromInput("11:11 16-03-2001", FullLayout)
	if err != nil {
		t.Error(err)
	}

	if time.Hour() != 11 {
		t.Error("Hour should be 11")
	}

	if time.Minute() != 11 {
		t.Error("Minute should be 11")
	}

	if time.Day() != 16 {
		t.Error("Day should be 16")
	}

	if time.Month() != 3 {
		t.Error("Month should be 3")
	}

	if time.Year() != 2001 {
		t.Error("Year should be 2001")
	}
}

func TestFormatTimestamp(t *testing.T) {
	timestamp := uint(1606683332)
	if FormatTimestamp(timestamp, TimeLayout) != "21:55" {
		t.Error("Time should be 21:55")
	}

	if FormatTimestamp(timestamp, DateLayout) != "29-11-2020" {
		t.Error("Date should be 29-11-2020")
	}

	if FormatTimestamp(timestamp, FullLayout) != "21:55 29-11-2020" {
		t.Error("Date should be 21:55 29-11-2020")
	}
}
