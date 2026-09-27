package models

import "math"

// ScalePage moves page of fromCount to the same fraction of toCount, keeping
// a started book between its first and last page. Page 0, not started,
// stays 0.
func ScalePage(page, fromCount, toCount int) int {
	if page < 1 || fromCount < 1 || toCount < 1 {
		return page
	}
	scaled := int(math.Round(float64(page) / float64(fromCount) * float64(toCount)))
	return min(max(scaled, 1), toCount)
}
