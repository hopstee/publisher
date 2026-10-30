package domain

type ReadResult struct {
	Content Document
	Path    string
	Images  []string
	Action  DocActionType
}
