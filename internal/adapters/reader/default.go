package reader

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"publisher/internal/domain"
	"regexp"

	"github.com/adrg/frontmatter"
	"go.yaml.in/yaml/v3"
)

const defaultReadDocumentsChanBuffer = 10
const defaultFrontmatterFormat = "yaml"

type Reader struct {
	readDocuments     chan domain.ReadResult
	frontmatterFormat string
}

func NewReader(format string) *Reader {
	if format == "" {
		format = defaultFrontmatterFormat
	}
	return &Reader{
		readDocuments:     make(chan domain.ReadResult, defaultReadDocumentsChanBuffer),
		frontmatterFormat: format,
	}
}

func (r *Reader) ReadAll(ctx context.Context, events <-chan domain.WatcherEvent) <-chan domain.ReadResult {
	go r.listenEvents(ctx, events)
	return r.readDocuments
}

func (r *Reader) Read(e domain.WatcherEvent) (domain.ReadResult, error) {
	return r.handleEvent(e)
}

func (r *Reader) listenEvents(ctx context.Context, events <-chan domain.WatcherEvent) {
	defer r.close()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				slog.Error("context occured", slog.Any("error", ctx.Err()))
			}
			return
		case e, ok := <-events:
			if !ok {
				return
			}
			res, err := r.handleEvent(e)
			if err != nil {
				continue
			}
			r.sendToChannel(ctx, res)
		}
	}
}

func (r *Reader) handleEvent(e domain.WatcherEvent) (domain.ReadResult, error) {
	b, err := readFile(e.Path)
	if err != nil {
		slog.Error("failed to read file", slog.Any("error", err))
		return domain.ReadResult{}, err
	}
	doc, err := r.parseFrontmatter(bytes.NewReader(b))
	if err != nil {
		slog.Error("failed to parse file", slog.Any("error", err))
		return domain.ReadResult{}, err
	}

	return domain.ReadResult{
		Content: doc,
		Path:    e.Path,
		Images:  extractImgLinks(b),
		Action:  e.Type,
	}, nil
}

func (r *Reader) sendToChannel(ctx context.Context, result domain.ReadResult) {
	select {
	case <-ctx.Done():
		return
	case r.readDocuments <- result:
	}
}

func readFile(filePath string) ([]byte, error) {
	// for how files small, and we can read them to the memory
	return os.ReadFile(filePath)
}

func (r *Reader) parseFrontmatter(b io.Reader) (domain.Document, error) {
	var doc domain.Document
	_, err := frontmatter.Parse(b, &doc, r.resolveFormat())
	if err != nil {
		return doc, err
	}
	return doc, nil
}

var (
	imgLinkRegexp = regexp.MustCompile(`!\[\[([^\]|]+)(?:\|[^\]]+)?\]\]|!\[[^\]]*\]\(([^)]+)\)`)
)

// TODO: add extracting alt text
func extractImgLinks(b []byte) []string {
	matches := imgLinkRegexp.FindAllSubmatch(b, -1)
	images := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		var imagePath string

		if match[1] != nil {
			imagePath = string(match[1])
		} else if match[2] != nil {
			imagePath = string(match[2])
		}

		if imagePath != "" && !seen[imagePath] {
			seen[imagePath] = true
			images = append(images, imagePath)
		}
	}

	return images
}

func (r *Reader) resolveFormat() *frontmatter.Format {
	// TODO: later add more formats
	switch r.frontmatterFormat {
	default:
		return frontmatter.NewFormat("---", "---", yaml.Unmarshal)
	}
}

func (r *Reader) close() {
	close(r.readDocuments)
}
