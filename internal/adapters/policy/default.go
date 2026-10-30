package policy

import (
	"context"
	"log/slog"
	"path/filepath"
	"publisher/internal/domain"
	"strings"
)

type Policy struct {
	checkVerdicts chan domain.CheckVerdict
	verbose       bool
	strict        bool
}

const defaultCheckVerdictsChanBuffer = 10

func NewPolicy(v bool, s bool) *Policy {
	return &Policy{
		checkVerdicts: make(chan domain.CheckVerdict, defaultCheckVerdictsChanBuffer),
		verbose:       v,
		strict:        s,
	}
}

func (p *Policy) Check(ctx context.Context, results <-chan domain.ReadResult) <-chan domain.CheckVerdict {
	go p.validate(ctx, results)
	return p.checkVerdicts
}

func (p *Policy) validate(ctx context.Context, results <-chan domain.ReadResult) {
	defer p.close()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				slog.Error("context occured", slog.Any("error", ctx.Err()))
			}
			return
		case r, ok := <-results:
			if !ok {
				return
			}

			findings := collectFindings(r.Content)
			imageFindings := p.validateImages(r.Path, r.Images)
			findings.Errors = append(findings.Errors, imageFindings.Errors...)
			findings.Warnings = append(findings.Warnings, imageFindings.Warnings...)
			p.logFindings(r.Path, findings)

			canPublish := p.canPublish(findings)

			p.writeToChannel(ctx, domain.CheckVerdict{
				Findings:         findings,
				Content:          r.Content,
				Path:             r.Path,
				Images:           r.Images,
				Action:           r.Action,
				CanPublish:       canPublish,
				ImmidiatePublish: false, // use from config
				PublishAt:        r.Content.Date,
			})
		}
	}
}

func (p *Policy) writeToChannel(ctx context.Context, c domain.CheckVerdict) {
	select {
	case <-ctx.Done():
		return
	case p.checkVerdicts <- c:
	}
}

// TODO: check image size (max from config) - warning
// check tags - warning
// title length - warning
// check slug for unique - error
func collectFindings(document domain.Document) domain.ValidationResult {
	var findings domain.ValidationResult
	if document.Slug == "" {
		findings.Errors = append(findings.Errors, domain.Finding{Field: "slug", Message: "Slug is required"})
	}
	if document.Title == "" {
		findings.Errors = append(findings.Errors, domain.Finding{Field: "title", Message: "Title is empty"})
	}
	if document.Description == "" {
		findings.Warnings = append(findings.Warnings, domain.Finding{Field: "description", Message: "Description is empty"})
	}
	if document.Cover == "" {
		findings.Warnings = append(findings.Warnings, domain.Finding{Field: "cover", Message: "Cover is empty"})
	}
	return findings
}

func (p *Policy) canPublish(f domain.ValidationResult) bool {
	if len(f.Errors) > 0 {
		return false
	}
	if p.verbose && len(f.Warnings) > 0 {
		return false
	}
	return true
}

func (p *Policy) logFindings(filepath string, f domain.ValidationResult) {
	if !p.verbose {
		return
	}
	errAttrs := make([]any, 0, len(f.Errors))
	for _, f := range f.Errors {
		errAttrs = append(errAttrs, slog.Group("finding",
			slog.String("field", f.Field),
			slog.String("message", f.Message),
		))
	}
	slog.Error(
		"Document has errors",
		slog.String("document", filepath),
		slog.Group("findings", errAttrs...),
	)

	warnAttrs := make([]any, 0, len(f.Warnings))
	for _, f := range f.Warnings {
		warnAttrs = append(warnAttrs, slog.Group("finding",
			slog.String("field", f.Field),
			slog.String("message", f.Message),
		))
	}
	slog.Warn(
		"Document has warnings",
		slog.String("document", filepath),
		slog.Group("findings", warnAttrs...),
	)
}

func (p *Policy) validateImages(filePath string, images []string) domain.ValidationResult {
	var findings domain.ValidationResult
	for _, image := range images {
		if isExternalImage(image) {
			continue
		}
		if isInsideFileDir(filePath, image) {
			continue
		}
		findings.Errors = append(findings.Errors, domain.Finding{Field: image, Message: "Unknown placement"})
	}
	return findings
}

func isExternalImage(image string) bool {
	image = strings.ToLower(image)
	return strings.HasPrefix(image, "http://") || strings.HasPrefix(image, "https://")
}

// this is a simlpe check for mac os
// TODO: check symlinks and file existing
func isInsideFileDir(filePath, imagePath string) bool {
	if filepath.IsAbs(imagePath) {
		return false
	}

	fileDir := filepath.Dir(filePath)
	resolvePath := filepath.Clean(filepath.Join(fileDir, imagePath))

	rel, err := filepath.Rel(fileDir, resolvePath)
	if err != nil {
		return false
	}

	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (p *Policy) close() {
	close(p.checkVerdicts)
}
