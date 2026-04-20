package pagination

type QueryParser interface {
	FormValue(name string) string
}

type Page[T any] struct {
	Content          []T   `json:"content"`
	Number           int   `json:"number"`
	Size             int   `json:"size"`
	TotalElements    int64 `json:"totalElements"`
	TotalPages       int   `json:"totalPages"`
	First            bool  `json:"first"`
	Last             bool  `json:"last"`
	NumberOfElements int   `json:"numberOfElements"`
	Empty            bool  `json:"empty"`
}

func Parse(r QueryParser) (page int, size int) {
	page = 0
	size = 20

	if p := r.FormValue("page"); p != "" {
		if parsed, ok := parseInt(p); ok && parsed >= 0 {
			page = parsed
		}
	}

	if s := r.FormValue("size"); s != "" {
		if parsed, ok := parseInt(s); ok {
			if parsed < 1 {
				size = 20
			} else if parsed > 100 {
				size = 100
			} else {
				size = parsed
			}
		}
	}

	return page, size
}

func parseInt(s string) (int, bool) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

func New[T any](content []T, page int, size int, total int64) Page[T] {
	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}

	return Page[T]{
		Content:          content,
		Number:           page,
		Size:             size,
		TotalElements:    total,
		TotalPages:       totalPages,
		First:            page == 0,
		Last:             page >= totalPages-1 && totalPages > 0,
		NumberOfElements: len(content),
		Empty:            len(content) == 0,
	}
}

func LimitOffset(page int, size int) (limit int, offset int) {
	if page < 0 {
		page = 0
	}
	limit = size
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset = page * limit
	return limit, offset
}