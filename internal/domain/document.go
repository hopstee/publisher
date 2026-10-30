package domain

import "time"

type Document struct {
	Slug        string    `yaml:"Slug"`
	Title       string    `yaml:"Title"`
	Description string    `yaml:"Description"`
	Cover       string    `yaml:"Cover"`
	Publish     bool      `yaml:"Publish"`
	Draft       bool      `yaml:"Draft"`
	Date        time.Time `yaml:"Date"`

	Path   string
	Images []string
}
