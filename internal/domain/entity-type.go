package domain

type DocActionType string

const (
	UpsertDocActionType DocActionType = "UPSERT"
	DeleteDocActionType DocActionType = "DELETE"
)
