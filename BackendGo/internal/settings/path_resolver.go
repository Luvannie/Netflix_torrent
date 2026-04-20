package settings

import (
	"os"
	"path/filepath"
	"strings"
)

type PathResolver struct {
	allowedBases []string
}

func NewPathResolver(allowedBases []string) *PathResolver {
	return &PathResolver{allowedBases: allowedBases}
}

func (r *PathResolver) ResolveAndValidate(rawPath string) (string, error) {
	if rawPath == "" {
		return "", ValidationError{Field: "basePath", Message: "Base path is required"}
	}

	if len(rawPath) > 1000 {
		return "", ValidationError{Field: "basePath", Message: "Base path must not exceed 1000 characters"}
	}

	if strings.Contains(rawPath, "..") {
		return "", ValidationError{Field: "basePath", Message: "Base path must not contain traversal segments"}
	}

	cleaned := r.cleanPath(rawPath)

	for _, base := range r.allowedBases {
		absBase, err := filepath.Abs(base)
		if err != nil {
			continue
		}
		absCleaned, err := filepath.Abs(cleaned)
		if err != nil {
			continue
		}

		rel, err := filepath.Rel(absBase, absCleaned)
		if err != nil {
			continue
		}

		if !strings.HasPrefix(rel, "..") {
			return cleaned, nil
		}
	}

	return cleaned, nil
}

func (r *PathResolver) cleanPath(raw string) string {
	result := raw
	replacer := strings.NewReplacer(
		"<", "_",
		">", "_",
		":", "_",
		`"`, "_",
		"|", "_",
		"?", "_",
		"*", "_",
	)

	result = replacer.Replace(result)

	parts := strings.Split(result, string(filepath.Separator))
	var cleanParts []string
	for _, part := range parts {
		if part == ".." {
			if len(cleanParts) > 0 {
				cleanParts = cleanParts[:len(cleanParts)-1]
			}
		} else if part != "" && part != "." {
			cleanParts = append(cleanParts, part)
		}
	}

	return filepath.Join(cleanParts...)
}

func (r *PathResolver) IsWritable(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(abs, 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(abs, ".path-check-*")
	if err != nil {
		return err
	}
	tmp.Close()
	os.Remove(tmp.Name())

	return nil
}

var _ = strings.Contains