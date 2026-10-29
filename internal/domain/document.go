package domain

import "time"

type Document struct {
	Slug        string
	Title       string
	Description string
	Cover       string
	Path        string
	Images      []string
	Publish     bool
	Date        time.Time
}
