package filter

import (
	"github.com/andrepxx/location-visualizer/geo"
	"time"
)

const (
	MILLISECONDS_PER_SECOND     = 1000
	NANOSECONDS_PER_MILLISECOND = 1000000
)

/*
 * A filter for location data.
 */
type Filter interface {
	Evaluate(geo.Location) bool
}

/*
 * Filters location data by time stamp.
 */
type timeFilterStruct struct {
	min time.Time
	max time.Time
}

/*
 * Evaluate whether a geographical location matches a filter criteria.
 */
func (this *timeFilterStruct) Evaluate(loc geo.Location) bool {

	/*
	 * Nil locations never match a filter.
	 */
	if loc == nil {
		return false
	} else {
		ts := loc.Timestamp()
		tsSeconds := ts / MILLISECONDS_PER_SECOND
		tsSecondsUnix := int64(tsSeconds)
		tsNanos := (ts % MILLISECONDS_PER_SECOND) * NANOSECONDS_PER_MILLISECOND
		tsNanosUnix := int64(tsNanos)
		tsTime := time.Unix(tsSecondsUnix, tsNanosUnix)
		tsTimeUTC := tsTime.UTC()
		match := true
		min := this.min
		minZero := min.IsZero()

		/*
		 * Match against minimum time, if it is set.
		 */
		if !minZero {
			match = match && !tsTimeUTC.Before(min)
		}

		max := this.max
		maxZero := max.IsZero()

		/*
		 * Match against maximum time, if it is set.
		 */
		if !maxZero {
			match = match && !tsTimeUTC.After(max)
		}

		return match
	}

}

/*
 * Apply a filter to a set of geographical locations to narrow it down.
 */
func Apply(flt Filter, locs []geo.Location) []geo.Location {
	matches := Evaluate(flt, locs)
	numMatches := 0

	/*
	 * Count number of matches.
	 */
	for _, match := range matches {

		/*
		 * Increment match counter on match.
		 */
		if match {
			numMatches++
		}

	}

	filtered := make([]geo.Location, numMatches)
	idx := 0

	/*
	 * Copy matches over.
	 */
	for i, match := range matches {

		/*
		 * Increment match counter on match.
		 */
		if match {
			loc := locs[i]
			filtered[idx] = loc
			idx++
		}

	}

	return filtered
}

/*
 * Evaluate whether a set of geographical locations matches filter criteria.
 */
func Evaluate(flt Filter, locs []geo.Location) []bool {
	numLocs := len(locs)
	matches := make([]bool, numLocs)

	/*
	 * A nil filter never matches anything.
	 */
	if flt != nil {

		/*
		 * Iterate over the locations and evaluate filter.
		 */
		for i, loc := range locs {
			match := flt.Evaluate(loc)
			matches[i] = match
		}

	}

	return matches
}

/*
 * Creates a UTC time stamp from RFC3339 string representation.
 */
func ParseTime(timestamp string, utc bool) (time.Time, error) {
	t, err := time.ParseInLocation(time.RFC3339, timestamp, time.UTC)

	/*
	 * Convert to UTC if requested.
	 */
	if utc {
		t = t.UTC()
	}

	return t, err
}

/*
 * Creates a filter which matches data points in a given time interval.
 *
 * Time values may be zero to leave either or both of the bounds open.
 */
func Time(min time.Time, max time.Time) Filter {

	/*
	 * Create a new time filter.
	 */
	t := timeFilterStruct{
		min: min,
		max: max,
	}

	return &t
}
