package openlibrary

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type Sort string

// Sorts are the orders Open Library's search supports; SortRelevance is its
// default.
const (
	SortRelevance Sort = ""
	SortRating    Sort = "rating"
	SortNewest    Sort = "new"
	SortOldest    Sort = "old"
)

var (
	languagePattern = regexp.MustCompile(`^[a-z]{3}$`)
	// subjectPattern keeps a subject to the words of a genre name, so it
	// can't close its quotes and add clauses of its own.
	subjectPattern = regexp.MustCompile(`^[a-z0-9' -]{2,40}$`)
)

// SearchInput is a search query plus the filters that narrow it. Either
// Query or Subject is required, so a genre can be browsed without a query.
// Language is a MARC language code such as "eng", a zero PublishedFrom or
// PublishedTo leaves that end of the first-published range open, and Page
// counts from 1, with 0 meaning the first page.
type SearchInput struct {
	Query         string
	Subject       string
	Sort          Sort
	Language      string
	PublishedFrom int
	PublishedTo   int
	Page          int
}

func (input SearchInput) validate() error {
	if input.Query == "" && input.Subject == "" {
		return ErrEmptyQuery
	}
	if input.Query != "" && utf8.RuneCountInString(input.Query) < MinimumQueryLength {
		return ErrQueryTooShort
	}
	if input.Subject != "" && !subjectPattern.MatchString(input.Subject) {
		return fmt.Errorf("%w %q", ErrInvalidSubject, input.Subject)
	}
	if input.Page < 0 {
		return fmt.Errorf("%w %d", ErrInvalidPage, input.Page)
	}
	switch input.Sort {
	case SortRelevance, SortRating, SortNewest, SortOldest:
	default:
		return fmt.Errorf("%w %q", ErrInvalidSort, input.Sort)
	}
	if input.Language != "" && !languagePattern.MatchString(input.Language) {
		return fmt.Errorf("%w %q", ErrInvalidLanguage, input.Language)
	}
	if input.PublishedFrom < 0 || input.PublishedTo < 0 ||
		(input.PublishedFrom != 0 && input.PublishedTo != 0 && input.PublishedFrom > input.PublishedTo) {
		return fmt.Errorf("%w %d to %d", ErrInvalidYearRange, input.PublishedFrom, input.PublishedTo)
	}
	return nil
}

// searchQuery appends the filters to the user's query as Open Library's
// fielded search clauses, e.g. `le guin subject:"fantasy" language:eng first_publish_year:[1960 TO 1979]`.
func (input SearchInput) searchQuery() string {
	var clauses []string
	if input.Query != "" {
		clauses = append(clauses, input.Query)
	}
	if input.Subject != "" {
		clauses = append(clauses, fmt.Sprintf("subject:%q", input.Subject))
	}
	if input.Language != "" {
		clauses = append(clauses, "language:"+input.Language)
	}
	if input.PublishedFrom != 0 || input.PublishedTo != 0 {
		clauses = append(clauses, fmt.Sprintf("first_publish_year:[%s TO %s]", yearBound(input.PublishedFrom), yearBound(input.PublishedTo)))
	}
	return strings.Join(clauses, " ")
}

func yearBound(year int) string {
	if year == 0 {
		return "*"
	}
	return fmt.Sprint(year)
}
