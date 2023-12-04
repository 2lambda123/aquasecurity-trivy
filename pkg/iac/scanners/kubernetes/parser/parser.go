package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/aquasecurity/defsec/pkg/debug"
	"github.com/aquasecurity/defsec/pkg/scanners/options"
	"github.com/aquasecurity/trivy/pkg/iac/detection"
)

var _ options.ConfigurableParser = (*Parser)(nil)

type Parser struct {
	debug        debug.Logger
	skipRequired bool
}

func (p *Parser) SetDebugWriter(writer io.Writer) {
	p.debug = debug.New(writer, "kubernetes", "parser")
}

func (p *Parser) SetSkipRequiredCheck(b bool) {
	p.skipRequired = b
}

// New creates a new K8s parser
func New(po ...options.ParserOption) *Parser {
	p := &Parser{}
	for _, option := range po {
		option(p)
	}
	return p
}

func (p *Parser) ParseFS(ctx context.Context, target fs.FS, path string) (map[string][]any, error) {
	files := make(map[string][]any)
	if err := fs.WalkDir(target, filepath.ToSlash(path), func(path string, entry fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !p.required(target, path) {
			return nil
		}
		parsed, err := p.ParseFile(ctx, target, path)
		if err != nil {
			p.debug.Log("Parse error in '%s': %s", path, err)
			return nil
		}
		files[path] = parsed
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

// ParseFile parses Kubernetes manifest from the provided filesystem path.
func (p *Parser) ParseFile(_ context.Context, fsys fs.FS, path string) ([]any, error) {
	f, err := fsys.Open(filepath.ToSlash(path))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return p.Parse(f, path)
}

func (p *Parser) required(fsys fs.FS, path string) bool {
	if p.skipRequired {
		return true
	}
	f, err := fsys.Open(filepath.ToSlash(path))
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	if data, err := io.ReadAll(f); err == nil {
		return detection.IsType(path, bytes.NewReader(data), detection.FileTypeKubernetes)
	}
	return false
}

func (p *Parser) Parse(r io.Reader, path string) ([]any, error) {

	contents, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if len(contents) == 0 {
		return nil, nil
	}

	if strings.TrimSpace(string(contents))[0] == '{' {
		var target any
		if err := json.Unmarshal(contents, &target); err != nil {
			return nil, err
		}
		return []any{target}, nil
	}

	var results []any

	re := regexp.MustCompile(`(?m:^---\r?\n)`)
	pos := 0
	for _, partial := range re.Split(string(contents), -1) {
		var result Manifest
		result.Path = path
		if err := yaml.Unmarshal([]byte(partial), &result); err != nil {
			return nil, fmt.Errorf("unmarshal yaml: %w", err)
		}
		if result.Content != nil {
			result.Content.Offset = pos
			results = append(results, result.ToRego())
		}
		pos += len(strings.Split(partial, "\n"))
	}

	return results, nil
}
