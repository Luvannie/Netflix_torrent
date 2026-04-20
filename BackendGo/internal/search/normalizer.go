package search

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Normalizer struct{}

func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

func (n *Normalizer) Normalize(results []ProviderResult) []NormalizedResult {
	seen := make(map[string]bool)
	var normalized []NormalizedResult

	for _, pr := range results {
		if pr.Error != nil {
			continue
		}
		for _, r := range pr.Results {
			key := n.deduplicationKey(r)
			if key == "" {
				continue
			}
			if seen[key] {
				continue
			}
			seen[key] = true

			r.Provider = pr.Provider
			r.Score = n.calculateScore(r.Seeders, r.PubDate)
			normalized = append(normalized, r)
		}
	}
	return normalized
}

func (n *Normalizer) deduplicationKey(r NormalizedResult) string {
	if r.Hash != "" {
		return "hash:" + r.Hash
	}
	key := normalizeString(r.Title)
	if r.Size > 0 {
		key += "|" + strconv.FormatInt(r.Size, 10)
	}
	return key
}

func (n *Normalizer) calculateScore(seeders int, pubDate *time.Time) int {
	score := seeders * 2

	if pubDate != nil {
		daysOld := time.Since(*pubDate).Hours() / 24
		if daysOld < 30 {
			score += 50
		} else if daysOld < 180 {
			score += 25
		} else if daysOld < 365 {
			score += 10
		}
	}

	return score
}

func (n *Normalizer) ExtractBTIHHash(magnetURI string) string {
	btihMatch := regexp.MustCompile(`urn:btih:([a-fA-F0-9]{40}|[a-zA-Z0-9]{32})`)
	matches := btihMatch.FindStringSubmatch(magnetURI)
	if len(matches) > 1 {
		return strings.ToLower(matches[1])
	}
	return ""
}

func normalizeString(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return s
}