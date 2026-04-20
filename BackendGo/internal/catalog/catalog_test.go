package catalog

import (
	"encoding/json"
	"testing"
)

func TestMovieSummaryJSONFields(t *testing.T) {
	summary := MovieSummary{
		ID:           1,
		TmdbID:       550,
		Title:        "Fight Club",
		PosterPath:   "/poster.jpg",
		ReleaseDate:  "1999-10-15",
		VoteAverage:  8.4,
		Popularity:   100.5,
		CatalogAddedAt: "2024-01-01T00:00:00Z",
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	expected := []string{"id", "tmdbId", "title", "posterPath", "releaseDate", "voteAverage", "popularity", "catalogAddedAt"}
	for _, field := range expected {
		if _, ok := decoded[field]; !ok {
			t.Errorf("missing field %q", field)
		}
	}
}

func TestTMDBClientConfig(t *testing.T) {
	client := NewTMDBClient("test-api-key", "", "https://api.themoviedb.org/3", "https://image.tmdb.org/t/p")

	if client.APIKey != "test-api-key" {
		t.Errorf("APIKey = %q", client.APIKey)
	}
	if client.BaseURL != "https://api.themoviedb.org/3" {
		t.Errorf("BaseURL = %q", client.BaseURL)
	}
}

func TestTMDBErrorImplementsError(t *testing.T) {
	err := &TMDBError{StatusCode: 404, Message: "not found"}
	if err.Error() != "not found" {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestPersonSearchResultStructure(t *testing.T) {
	jsonData := `{"id": 525, "name": "Christopher Nolan", "known_for": [{"id": 155, "title": "The Dark Knight"}]}`

	var result PersonSearchResult
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if result.ID != 525 {
		t.Errorf("ID = %d", result.ID)
	}
	if result.Name != "Christopher Nolan" {
		t.Errorf("Name = %q", result.Name)
	}
}

func TestTMDBSearchResponseStructure(t *testing.T) {
	jsonData := `{"results": [{"id": 550, "title": "Fight Club"}], "total_pages": 1}`

	var resp TMDBSearchResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if len(resp.Results) != 1 {
		t.Errorf("Results count = %d", len(resp.Results))
	}
	if resp.TotalPages != 1 {
		t.Errorf("TotalPages = %d", resp.TotalPages)
	}
}

func TestGenreStructure(t *testing.T) {
	jsonData := `{"id": 28, "name": "Action"}`

	var genre Genre
	if err := json.Unmarshal([]byte(jsonData), &genre); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if genre.ID != 28 {
		t.Errorf("ID = %d", genre.ID)
	}
	if genre.Name != "Action" {
		t.Errorf("Name = %q", genre.Name)
	}
}

func TestImageURLHelper(t *testing.T) {
	client := NewTMDBClient("", "", "", "https://image.tmdb.org/t/p")

	url := client.ImageBaseURL + "/w500" + "/poster.jpg"
	expected := "https://image.tmdb.org/t/p/w500/poster.jpg"
	if url != expected {
		t.Errorf("imageURL = %q, want %q", url, expected)
	}
}

func TestTMDBMovieDetailStructure(t *testing.T) {
	jsonData := `{
		"id": 550,
		"title": "Fight Club",
		"overview": "A ticking-Loss",
		"poster_path": "/poster.jpg",
		"backdrop_path": "/backdrop.jpg",
		"release_date": "1999-10-15",
		"vote_average": 8.4,
		"vote_count": 5000,
		"popularity": 100.5,
		"original_language": "en",
		"original_title": "Fight Club"
	}`

	var movie TMDBMovieDetail
	if err := json.Unmarshal([]byte(jsonData), &movie); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if movie.ID != 550 {
		t.Errorf("ID = %d", movie.ID)
	}
	if movie.Title != "Fight Club" {
		t.Errorf("Title = %q", movie.Title)
	}
}