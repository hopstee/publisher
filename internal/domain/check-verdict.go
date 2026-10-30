package domain

import "time"

type Finding struct {
	Field   string
	Message string
}

type ValidationResult struct {
	Errors   []Finding
	Warnings []Finding
}

type CheckVerdict struct {
	Findings ValidationResult
	Content  Document
	Path     string
	Images   []string
	Action   DocActionType

	CanPublish       bool
	ImmidiatePublish bool
	PublishAt        time.Time
}
