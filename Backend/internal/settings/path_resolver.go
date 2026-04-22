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
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", ValidationError{Field: "basePath", Message: "Base path is required"}
	}

	if len(rawPath) > 1000 {
		return "", ValidationError{Field: "basePath", Message: "Base path must not exceed 1000 characters"}
	}

	if containsTraversalSegment(rawPath) {
		return "", ValidationError{Field: "basePath", Message: "Base path must not contain traversal segments"}
	}

	cleaned := r.cleanPath(rawPath)
	absCleaned, err := filepath.Abs(cleaned)
	if err != nil {
		return "", ValidationError{Field: "basePath", Message: "Base path is invalid"}
	}

	if len(r.allowedBases) == 0 {
		return absCleaned, nil
	}

	for _, base := range r.allowedBases {
		if pathWithinBase(absCleaned, base) {
			return absCleaned, nil
		}
	}

	return "", ValidationError{Field: "basePath", Message: "Base path must stay within an allowed storage root"}
}

func (r *PathResolver) cleanPath(raw string) string {
	volume := filepath.VolumeName(raw)
	trimmed := strings.TrimPrefix(raw, volume)
	isAbs := filepath.IsAbs(raw)

	replacer := strings.NewReplacer(
		"<", "_",
		">", "_",
		`"`, "_",
		"|", "_",
		"?", "_",
		"*", "_",
		":", "_",
	)

	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	var cleanParts []string
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			continue
		}
		cleanParts = append(cleanParts, replacer.Replace(part))
	}

	joined := filepath.Join(cleanParts...)
	if volume != "" && isAbs {
		if joined == "" {
			return volume + string(filepath.Separator)
		}
		return filepath.Join(volume+string(filepath.Separator), joined)
	}
	if isAbs {
		if joined == "" {
			return string(filepath.Separator)
		}
		return filepath.Join(string(filepath.Separator), joined)
	}
	if volume != "" {
		if joined == "" {
			return volume + string(filepath.Separator)
		}
		return filepath.Join(volume+string(filepath.Separator), joined)
	}
	return joined
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

func containsTraversalSegment(rawPath string) bool {
	parts := strings.FieldsFunc(rawPath, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	for _, part := range parts {
		if part == ".." {
			return true
		}
	}
	return false
}

func pathWithinBase(path string, base string) bool {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(absBase, path)
	if err != nil {
		return false
	}

	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
