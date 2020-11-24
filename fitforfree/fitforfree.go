package fitforfree

import (
	"errors"
	"fmt"
	"net/http"
)

// GetLessons gets available free fitness lessons between 2 timestamps
func GetLessons(start uint, end uint) error {
	if start > end {
		return errors.New("Start must be before end")
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("https://electrolyte.fitforfree.nl/v0/lessons?venues=%s&from=%d&to=%d&language=%s", "%5B%22f6f802c8-e678-47b9-91e2-36392a557581%22%5D", start, end, "nl_NL"), nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", ""))
	_, err = client.Do(req)
	if err != nil {
		panic(err)
	}

	panic("Not implemented")

	return nil
}
