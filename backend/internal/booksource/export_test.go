package booksource

import "time"

// SetRetryDelay shortens the wait before a lookup is retried, returning a
// function that restores it.
func SetRetryDelay(delay time.Duration) func() {
	previous := retryDelay
	retryDelay = delay
	return func() { retryDelay = previous }
}
