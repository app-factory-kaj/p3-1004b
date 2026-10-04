// Package store is the in-memory book catalog. catalog-service keeps no
// database — the design (design.json) fixes it as an in-memory store, so a
// restart drops every write, including the seeded books.
package store

import (
	"sort"
	"strconv"
	"strings"
	"sync"

	"catalog-service/internal/gen"
)

// ErrNotFound is returned when a bookId has no matching row.
type ErrNotFound struct{ BookID string }

func (e ErrNotFound) Error() string { return "book not found: " + e.BookID }

// Store is a concurrency-safe in-memory catalog of books.
type Store struct {
	mu     sync.RWMutex
	books  map[string]gen.Book
	nextID int
}

// New returns a Store seeded with a small handful of realistic books so the
// catalog is not empty on first boot.
func New() *Store {
	s := &Store{books: make(map[string]gen.Book)}
	for _, b := range seedBooks() {
		s.books[b.ID] = b
	}
	s.nextID = len(s.books) + 1
	return s
}

func seedBooks() []gen.Book {
	return []gen.Book{
		{
			ID:          "book-1",
			Title:       "The Pragmatic Programmer",
			Author:      "David Thomas, Andrew Hunt",
			Isbn:        "978-0135957059",
			Description: "A guide to becoming a better, more effective programmer.",
			Price:       39.99,
			Tags:        []string{"programming", "software-engineering"},
		},
		{
			ID:          "book-2",
			Title:       "Dune",
			Author:      "Frank Herbert",
			Isbn:        "978-0441013593",
			Description: "A sweeping science-fiction epic set on the desert planet Arrakis.",
			Price:       18.50,
			Tags:        []string{"fiction", "sci-fi"},
		},
		{
			ID:          "book-3",
			Title:       "Sapiens: A Brief History of Humankind",
			Author:      "Yuval Noah Harari",
			Isbn:        "978-0062316097",
			Description: "A sweeping look at how Homo sapiens came to dominate the world.",
			Price:       22.00,
			Tags:        []string{"non-fiction", "history"},
		},
		{
			ID:          "book-4",
			Title:       "The Hobbit",
			Author:      "J.R.R. Tolkien",
			Isbn:        "978-0547928227",
			Description: "Bilbo Baggins is swept into an epic quest to reclaim a dwarf kingdom.",
			Price:       14.99,
			Tags:        []string{"fiction", "fantasy"},
		},
		{
			ID:          "book-5",
			Title:       "Thinking, Fast and Slow",
			Author:      "Daniel Kahneman",
			Isbn:        "978-0374533557",
			Description: "An exploration of the two systems that drive the way we think.",
			Price:       19.99,
			Tags:        []string{"non-fiction", "psychology"},
		},
	}
}

// List returns books matching q (free text over title/author/description)
// and tag, paginated by limit/offset, plus the total count of matches before
// pagination.
func (s *Store) List(q, tag string, limit, offset int) (books []gen.Book, count int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matched := make([]gen.Book, 0, len(s.books))
	for _, b := range s.books {
		if !matchesQuery(b, q) {
			continue
		}
		if !matchesTag(b, tag) {
			continue
		}
		matched = append(matched, b)
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].ID < matched[j].ID })

	count = len(matched)
	if offset >= count {
		return []gen.Book{}, count
	}
	end := offset + limit
	if end > count {
		end = count
	}
	return matched[offset:end], count
}

func matchesQuery(b gen.Book, q string) bool {
	if q == "" {
		return true
	}
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(b.Title), q) ||
		strings.Contains(strings.ToLower(b.Author), q) ||
		strings.Contains(strings.ToLower(b.Description), q)
}

func matchesTag(b gen.Book, tag string) bool {
	if tag == "" {
		return true
	}
	for _, t := range b.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Get returns one book by id.
func (s *Store) Get(bookID string) (gen.Book, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.books[bookID]
	if !ok {
		return gen.Book{}, ErrNotFound{BookID: bookID}
	}
	return b, nil
}

// Create adds a new book and returns it with its generated id.
func (s *Store) Create(in gen.BookInput) gen.Book {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	b := gen.Book{
		ID:          "book-" + strconv.Itoa(s.nextID),
		Title:       in.Title,
		Author:      in.Author,
		Isbn:        in.Isbn,
		Description: in.Description,
		Price:       in.Price,
		Tags:        normalizeTags(in.Tags),
	}
	s.books[b.ID] = b
	return b
}

// Update replaces an existing book's fields. Returns ErrNotFound when bookID
// does not exist.
func (s *Store) Update(bookID string, in gen.BookInput) (gen.Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.books[bookID]; !ok {
		return gen.Book{}, ErrNotFound{BookID: bookID}
	}
	b := gen.Book{
		ID:          bookID,
		Title:       in.Title,
		Author:      in.Author,
		Isbn:        in.Isbn,
		Description: in.Description,
		Price:       in.Price,
		Tags:        normalizeTags(in.Tags),
	}
	s.books[bookID] = b
	return b, nil
}

// Delete removes a book. Returns ErrNotFound when bookID does not exist.
func (s *Store) Delete(bookID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.books[bookID]; !ok {
		return ErrNotFound{BookID: bookID}
	}
	delete(s.books, bookID)
	return nil
}

// normalizeTags turns a nil slice into an empty one so a JSON response always
// carries `"tags": []` rather than `"tags": null` for a book created with none.
func normalizeTags(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}
