package booksource

import (
	"strings"
	"unicode"
)

// mainTitle is a title without its subtitle or alternative title, which
// follow a colon, a semicolon or a bracket, as in "Frankenstein; or, The
// Modern Prometheus".
func mainTitle(title string) string {
	if end := strings.IndexAny(title, ":;(["); end >= 0 {
		title = title[:end]
	}
	return strings.TrimSpace(title)
}

// normalized lowercases a title and drops its punctuation, so titles
// compare by their words alone.
func normalized(title string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(title), func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsNumber(character)
	}), " ")
}

// sameTitle reports whether a catalog's title is the book's, comparing
// their main titles' words, so "Frankenstein" matches "Frankenstein; or,
// The Modern Prometheus" but not "Frankenstein Unbound".
func sameTitle(bookTitle, catalogTitle string) bool {
	book := normalized(mainTitle(bookTitle))
	return book != "" && book == normalized(mainTitle(catalogTitle))
}

// firstNameFirst turns a catalog's "Austen, Jane" into "Jane Austen".
func firstNameFirst(name string) string {
	last, first, ok := strings.Cut(strings.TrimSpace(name), ",")
	if !ok || strings.Contains(first, ",") {
		return strings.TrimSpace(name)
	}
	return strings.TrimSpace(first) + " " + strings.TrimSpace(last)
}
