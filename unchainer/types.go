package unchainer

import (
	"net/url"
)

// InputData contains read data from input file
type InputData struct {
	Links []Link `json:"links"`
}

// Link contains URL to unchain
type Link struct {
	URL *url.URL
}

// UnmarshalJSON unmarshals JSON to Link struct parsing the link
func (j *Link) UnmarshalJSON(b []byte) error {
	// Strip off the surrounding quotes
	parsedURL, err := url.Parse(string(b[1 : len(b)-1]))
	if err == nil {
		j.URL = parsedURL
	}
	return err
}

// Result contains unchaining results and start link
type Result struct {
	Start string
	Chain []string
}
