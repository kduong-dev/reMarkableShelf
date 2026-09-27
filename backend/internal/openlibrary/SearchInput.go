package openlibrary

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/ansel1/merry"
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

var languagePattern = regexp.MustCompile(`^[a-z]{3}$`)

// SearchInput is a search query plus the filters that narrow it. Language is
// a MARC language code such as "eng", and a zero PublishedFrom or
// PublishedTo leaves that end of the first-published range open.
type SearchInput struct {
	Query         string
	Sort          Sort
	Language      string
	PublishedFrom int
	PublishedTo   int
}

func (input SearchInput) validate() error {
	if input.Query == "" {
		return ErrEmptyQuery
	}
	if utf8.RuneCountInString(input.Query) < MinimumQueryLength {
		return merry.Here(ErrQueryTooShort)
	}
	switch input.Sort {
	case SortRelevance, SortRating, SortNewest, SortOldest:
	default:
		return merry.Here(ErrInvalidSort).WithMessagef("unknown sort %q", input.Sort)
	}
	if input.Language != "" && !languagePattern.MatchString(input.Language) {
		return merry.Here(ErrInvalidLanguage).WithMessagef("invalid language %q", input.Language)
	}
	if input.PublishedFrom < 0 || input.PublishedTo < 0 ||
		(input.PublishedFrom != 0 && input.PublishedTo != 0 && input.PublishedFrom > input.PublishedTo) {
		return merry.Here(ErrInvalidYearRange).WithMessagef("invalid year range %d to %d", input.PublishedFrom, input.PublishedTo)
	}
	return nil
}

// searchQuery appends the filters to the user's query as Open Library's
// fielded search clauses, e.g. `le guin language:eng first_publish_year:[1960 TO 1979]`.
func (input SearchInput) searchQuery() string {
	clauses := []string{input.Query}
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
