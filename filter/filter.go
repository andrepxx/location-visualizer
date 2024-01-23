package filter

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/andrepxx/location-visualizer/geo/geodb"
)

const (
	MILLISECONDS_PER_SECOND     = 1000
	NANOSECONDS_PER_MILLISECOND = 1000000
	REX_SLOPPY_TIME             = "^\\s*(\\d{4})(-(\\d{2}))?(-(\\d{2}))?(((T|\\s)(\\d{2})(:(\\d{2}))(:(\\d{2}))?)?((Z)|((\\s+(GMT|UTC))?(([+-])(\\d{2})(:(\\d{2}))?)?)))?\\s*$"
)

/*
 * A filter for location data.
 */
type Filter interface {
	Evaluate(loc *geodb.Location) bool
}

/*
 * Filters location data by time stamp.
 */
type timeFilterStruct struct {
	min time.Time
	max time.Time
}

/*
 * Get string value with default value if empty.
 */
func getValue(values []string, idx int, d string) string {
	numValues := len(values)

	/*
	 * If index is out of range, return default value.
	 */
	if idx > numValues {
		return d
	} else {
		value := values[idx]

		/*
		 * If value is empty, return default value.
		 */
		if value == "" {
			return d
		} else {
			return value
		}

	}

}

/*
 * Convert "sloppy" time stamp to RFC3339 representation.
 */
func fromSloppyTime(in string) (string, error) {
	rex, _ := regexp.Compile(REX_SLOPPY_TIME)

	/*
	 * Check if regular expression compiles.
	 */
	if rex == nil {
		return "", fmt.Errorf("Failed to compile regular expression: %s", REX_SLOPPY_TIME)
	} else {
		upper := strings.ToUpper(in)
		groups := rex.FindStringSubmatch(upper)

		/*
		 * Extract values if regular expression matches.
		 */
		if groups == nil {
			return "", fmt.Errorf("Time stamp does not match regular expression: %s", REX_SLOPPY_TIME)
		} else {
			year := getValue(groups, 1, "0001")
			month := getValue(groups, 3, "01")
			day := getValue(groups, 5, "01")
			hour := getValue(groups, 9, "00")
			minute := getValue(groups, 11, "00")
			second := getValue(groups, 13, "00")
			z := getValue(groups, 15, "")
			result := ""

			/*
			 * Check if "Z" representation is used for UTC.
			 */
			if z != "" {
				result = fmt.Sprintf("%s-%s-%sT%s:%s:%s%s", year, month, day, hour, minute, second, z)
			} else {
				offsetSign := getValue(groups, 20, "+")
				offsetHours := getValue(groups, 21, "00")
				offsetMinutes := getValue(groups, 23, "00")
				result = fmt.Sprintf("%s-%s-%sT%s:%s:%s%s%s:%s", year, month, day, hour, minute, second, offsetSign, offsetHours, offsetMinutes)
			}

			return result, nil
		}

	}

}

/*
 * Evaluate whether a geographical location matches a filter criteria.
 */
func (this *timeFilterStruct) Evaluate(loc *geodb.Location) bool {

	/*
	 * Nil locations never match a filter.
	 */
	if loc == nil {
		return false
	} else {
		ts := loc.Timestamp
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
func Apply(flt Filter, in []geodb.Location, out []geodb.Location) int {
	numIn := len(in)
	matches := make([]bool, numIn)
	Evaluate(flt, in, matches)
	numOut := len(out)
	idx := 0

	/*
	 * Copy matches over.
	 */
	for i, match := range matches {
		hasSpace := idx < numOut

		/*
		 * Copy to output on match.
		 */
		if match && hasSpace {
			loc := in[i]
			out[idx] = loc
			idx++
		}

	}

	return idx
}

/*
 * Evaluate whether a set of geographical locations matches filter criteria.
 */
func Evaluate(flt Filter, locs []geodb.Location, result []bool) {

	/*
	 * A nil filter always matches everything.
	 */
	if flt == nil {

		/*
		 * Clear result.
		 */
		for i := range result {
			result[i] = true
		}

	} else {
		numLocs := len(locs)

		/*
		 * Produce results by evaluating filter.
		 */
		for i := range result {
			match := false

			/*
			 * Prevent out-of-bounds error.
			 */
			if i < numLocs {
				loc := &locs[i]
				match = flt.Evaluate(loc)
			}

			result[i] = match
		}

	}

}

/*
 * Creates a UTC time stamp from RFC3339 string representation.
 */
func ParseTime(timestamp string, sloppy bool, utc bool) (time.Time, error) {
	err := error(nil)

	/*
	 * Convert sloppy time to RFC3339 representation.
	 */
	if sloppy {
		timestamp, err = fromSloppyTime(timestamp)
	}

	/*
	 * Check if error occured so far.
	 */
	if err != nil {
		return time.Time{}, err
	} else {
		t, err := time.ParseInLocation(time.RFC3339, timestamp, time.UTC)

		/*
		 * Convert to UTC if requested.
		 */
		if utc {
			t = t.UTC()
		}

		return t, err
	}

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
