package search

import "time"

type SearchJobStatus string

const (
	StatusRequested  SearchJobStatus = "REQUESTED"
	StatusSearching  SearchJobStatus = "SEARCHING"
	StatusSearchReady SearchJobStatus = "SEARCH_READY"
	StatusFailed    SearchJobStatus = "FAILED"
	StatusCancelled SearchJobStatus = "CANCELLED"
)

type SearchJob struct {
	ID           int64          `json:"id"`
	Query        string         `json:"query"`
	Status       SearchJobStatus `json:"status"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	ErrorMessage string         `json:"errorMessage,omitempty"`
	Results      []SearchResult `json:"results,omitempty"`
}

type SearchResult struct {
	ID           int64      `json:"id"`
	SearchJobID  int64      `json:"searchJobId"`
	Guid         string     `json:"guid"`
	Title        string     `json:"title"`
	Link         string     `json:"link"`
	Permalink    string     `json:"permalink"`
	Size         int64      `json:"size"`
	PubDate      *time.Time `json:"pubDate,omitempty"`
	Seeders      int        `json:"seeders"`
	Leechers     int        `json:"leechers"`
	Indexer      string     `json:"indexer"`
	Provider     string     `json:"provider"`
	Hash         string     `json:"hash"`
	Score        int        `json:"score"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type CreateSearchJobRequest struct {
	Query string `json:"query"`
}

type NormalizedResult struct {
	Title     string
	Guid      string
	Link      string
	Permalink string
	Size      int64
	PubDate   *time.Time
	Seeders   int
	Leechers  int
	Indexer   string
	Provider  string
	Hash      string
	Score     int
}

type ProviderResult struct {
	Provider string
	Results  []NormalizedResult
	Error   error
}

func (s SearchJobStatus) String() string {
	return string(s)
}

func (s SearchJobStatus) IsValidTransition(to SearchJobStatus) bool {
	switch s {
	case StatusRequested:
		return to == StatusSearching || to == StatusCancelled
	case StatusSearching:
		return to == StatusSearchReady || to == StatusFailed || to == StatusCancelled
	case StatusSearchReady:
		return to == StatusCancelled
	default:
		return false
	}
}