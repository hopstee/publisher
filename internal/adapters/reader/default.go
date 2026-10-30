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

const DEFAULT_READ_DOCUMENTS_CHAN_BUFFER = 10
const DEFAULT_FRONTMATTER_FORMAT = "yaml"

type Reader struct {
	readDocuments     chan domain.ReadResult
	frontmatterFormat string
}

func NewReader(format string) Reader {
	if format == "" {
		format = DEFAULT_FRONTMATTER_FORMAT
	}
	return Reader{
		readDocuments:     make(chan domain.ReadResult, DEFAULT_READ_DOCUMENTS_CHAN_BUFFER),
		frontmatterFormat: format,
	}
}

func (r Reader) Read(ctx context.Context, events <-chan domain.WatcherEvent) <-chan domain.ReadResult {
	go r.listenEvents(ctx, events)
	return r.readDocuments
}

func (r Reader) listenEvents(ctx context.Context, events <-chan domain.WatcherEvent) {
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				slog.Error("context occured", slog.Any("error", ctx.Err()))
			}
			return
		case e := <-events:
			b, err := readFile(e.Path)
			if err != nil {
				slog.Error("failed to read file", slog.Any("error", err))
				continue
			}
			doc, err := r.parseFrontmatter(bytes.NewReader(b))
			if err != nil {
				slog.Error("failed to parse file", slog.Any("error", err))
				continue
			}

			r.readDocuments <- domain.ReadResult{
				Content: doc,
				Path:    e.Path,
				Images:  extractImgLinks(b),
				Action:  e.Type,
			}
		}
	}
}

func readFile(filePath string) ([]byte, error) {
	// for how files small, and we can read them to the memory
	return os.ReadFile(filePath)
}

func (r Reader) parseFrontmatter(b io.Reader) (domain.Document, error) {
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

func (r Reader) resolveFormat() *frontmatter.Format {
	// TODO: later add more formats
	switch r.frontmatterFormat {
	default:
		return frontmatter.NewFormat("---", "---", yaml.Unmarshal)
	}
}
