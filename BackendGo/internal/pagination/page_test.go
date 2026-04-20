package pagination

import (
	"testing"
)

type mockQueryParser struct {
	values map[string]string
}

func (m mockQueryParser) FormValue(key string) string {
	return m.values[key]
}

func TestParseDefaults(t *testing.T) {
	parser := mockQueryParser{values: map[string]string{}}
	page, size := Parse(parser)
	if page != 0 {
		t.Fatalf("page = %d, want 0", page)
	}
	if size != 20 {
		t.Fatalf("size = %d, want 20", size)
	}
}

func TestParseCustomPageAndSize(t *testing.T) {
	parser := mockQueryParser{values: map[string]string{"page": "2", "size": "50"}}
	page, size := Parse(parser)
	if page != 2 {
		t.Fatalf("page = %d, want 2", page)
	}
	if size != 50 {
		t.Fatalf("size = %d, want 50", size)
	}
}

func TestParseNegativePageBecomesZero(t *testing.T) {
	parser := mockQueryParser{values: map[string]string{"page": "-1"}}
	page, _ := Parse(parser)
	if page != 0 {
		t.Fatalf("page = %d, want 0", page)
	}
}

func TestParseSizeAbove100Becomes100(t *testing.T) {
	parser := mockQueryParser{values: map[string]string{"size": "150"}}
	_, size := Parse(parser)
	if size != 100 {
		t.Fatalf("size = %d, want 100", size)
	}
}

func TestParseSizeBelow1Becomes20(t *testing.T) {
	parser := mockQueryParser{values: map[string]string{"size": "0"}}
	_, size := Parse(parser)
	if size != 20 {
		t.Fatalf("size = %d, want 20", size)
	}
}

func TestNewPageContentShape(t *testing.T) {
	content := []string{"a", "b", "c"}
	page := New(content, 0, 20, 100)

	if len(page.Content) != 3 {
		t.Fatalf("Content length = %d", len(page.Content))
	}
	if page.Number != 0 {
		t.Fatalf("Number = %d", page.Number)
	}
	if page.Size != 20 {
		t.Fatalf("Size = %d", page.Size)
	}
	if page.TotalElements != 100 {
		t.Fatalf("TotalElements = %d", page.TotalElements)
	}
	if page.TotalPages != 5 {
		t.Fatalf("TotalPages = %d", page.TotalPages)
	}
	if !page.First {
		t.Fatalf("First should be true")
	}
	if page.Last {
		t.Fatalf("Last should be false")
	}
	if page.Empty {
		t.Fatalf("Empty should be false")
	}
}

func TestNewEmptyPage(t *testing.T) {
	content := []string{}
	page := New(content, 0, 20, 0)

	if !page.Empty {
		t.Fatalf("Empty should be true")
	}
	if page.TotalPages != 0 {
		t.Fatalf("TotalPages = %d", page.TotalPages)
	}
}

func TestLimitOffset(t *testing.T) {
	limit, offset := LimitOffset(0, 20)
	if limit != 20 || offset != 0 {
		t.Fatalf("LimitOffset(0, 20) = (%d, %d)", limit, offset)
	}

	limit, offset = LimitOffset(2, 20)
	if limit != 20 || offset != 40 {
		t.Fatalf("LimitOffset(2, 20) = (%d, %d)", limit, offset)
	}

	limit, offset = LimitOffset(1, 50)
	if limit != 50 || offset != 50 {
		t.Fatalf("LimitOffset(1, 50) = (%d, %d)", limit, offset)
	}
}

func TestLimitOffsetNegativePage(t *testing.T) {
	_, offset := LimitOffset(-5, 20)
	if offset != 0 {
		t.Fatalf("offset = %d, want 0", offset)
	}
}

func TestLimitOffsetSizeAbove100(t *testing.T) {
	limit, _ := LimitOffset(0, 200)
	if limit != 100 {
		t.Fatalf("limit = %d, want 100", limit)
	}
}