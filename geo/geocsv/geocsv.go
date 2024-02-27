package geocsv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/andrepxx/location-visualizer/geo"
)

/*
 * Data structure representing a CSV location.
 */
type locationStruct struct {
	timestamp   uint64
	latitudeE7  int32
	longitudeE7 int32
}

/*
 * Data structure representing the top-level CSV element.
 */
type databaseStruct struct {
	locations []locationStruct
}

/*
 * Returns the latitude of this location.
 */
func (this *locationStruct) Latitude() int32 {
	latitudeE7 := this.latitudeE7
	return latitudeE7
}

/*
 * Returns the longitude of this location.
 */
func (this *locationStruct) Longitude() int32 {
	longitudeE7 := this.longitudeE7
	return longitudeE7
}

/*
 * Returns the timestamp (in milliseconds since the Epoch) when
 * this location was recorded.
 */
func (this *locationStruct) Timestamp() uint64 {
	timestamp := this.timestamp
	return timestamp
}

/*
 * Parses a latitude.
 */
func (this *databaseStruct) parseLatitude(latitudeString string) (int32, error) {
	latitudeString = strings.TrimSpace(latitudeString)
	n := len(latitudeString)
	result := int32(0)
	errResult := error(nil)

	/*
	 * Make sure that string is long enough.
	 */
	if n < 10 {
		return 0, fmt.Errorf("Expected at least %d characters, but found only %d.", 10, n)
	} else {
		posDirection := n - 1
		direction := latitudeString[posDirection]
		sign := int32(0)

		/*
		 * Decide on direction.
		 */
		switch direction {
		case 'N', 'n':
			sign = 1
		case 'S', 's':
			sign = -1
		}

		posDot := n - 9
		dot := latitudeString[posDot]

		/*
		 * Check that the format is as expected.
		 */
		if sign == 0 {
			errResult = fmt.Errorf("Failed to parse latitude: Expected 'N', 'S', 'n' or 's' at offset %d.", posDirection)
		} else if dot != '.' {
			errResult = fmt.Errorf("Failed to parse latitude: Expected dot at offset %d.", posDot)
		} else {
			posDotInc := posDot + 1
			leftOfDot := latitudeString[0:posDot]
			rightOfDot := latitudeString[posDotInc:posDirection]
			mantissaString := leftOfDot + rightOfDot
			mantissa, err := strconv.ParseUint(mantissaString, 10, 31)

			/*
			 * Check if error occured.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Failed to parse latitude: Error parsing mantissa of fixed-point number: %s", msg)
			} else {
				mantissa32 := int32(mantissa)
				result = sign * mantissa32
			}

		}

	}

	return result, errResult
}

/*
 * Parses a longitude.
 */
func (this *databaseStruct) parseLongitude(longitudeString string) (int32, error) {
	longitudeString = strings.TrimSpace(longitudeString)
	n := len(longitudeString)
	result := int32(0)
	errResult := error(nil)

	/*
	 * Make sure that string is long enough.
	 */
	if n < 10 {
		return 0, fmt.Errorf("Expected at least %d characters, but found only %d.", 10, n)
	} else {
		posDirection := n - 1
		direction := longitudeString[posDirection]
		sign := int32(0)

		/*
		 * Decide on direction.
		 */
		switch direction {
		case 'E', 'e':
			sign = 1
		case 'W', 'w':
			sign = -1
		}

		posDot := n - 9
		dot := longitudeString[posDot]

		/*
		 * Check that the format is as expected.
		 */
		if sign == 0 {
			errResult = fmt.Errorf("Failed to parse longitude: Expected 'N', 'S', 'n' or 's' at offset %d.", posDirection)
		} else if dot != '.' {
			errResult = fmt.Errorf("Failed to parse longitude: Expected dot at offset %d.", posDot)
		} else {
			posDotInc := posDot + 1
			leftOfDot := longitudeString[0:posDot]
			rightOfDot := longitudeString[posDotInc:posDirection]
			mantissaString := leftOfDot + rightOfDot
			mantissa, err := strconv.ParseUint(mantissaString, 10, 31)

			/*
			 * Check if error occured.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Failed to parse longitude: Error parsing mantissa of fixed-point number: %s", msg)
			} else {
				mantissa32 := int32(mantissa)
				result = sign * mantissa32
			}

		}

	}

	return result, errResult
}

/*
 * Parses a timestamp and returns the number of milliseconds since the Epoch.
 */
func (this *databaseStruct) parseTimestamp(timestampString string) (uint64, error) {
	layout := time.RFC3339Nano
	location := time.UTC
	parsedTime, err := time.ParseInLocation(layout, timestampString, location)
	timestamp := uint64(0)

	/*
	 * ParseInLocation does not specify the result on error.
	 */
	if err == nil {
		unixMs := parsedTime.UnixMilli()
		timestamp = uint64(unixMs)
	}

	return timestamp, err
}

/*
 * The location stored at the given index in this database.
 */
func (this *databaseStruct) LocationAt(idx int) (geo.Location, error) {
	locs := this.locations
	numLocs := len(locs)

	/*
	 * Check if index is in valid range.
	 */
	if (idx < 0) || (idx >= numLocs) {
		lastIdx := numLocs - 1
		return nil, fmt.Errorf("Index must be in [%d, %d].", 0, lastIdx)
	} else {
		ptr := &locs[idx]
		return ptr, nil
	}

}

/*
 * The number of locations stored in this database.
 */
func (this *databaseStruct) LocationCount() int {
	locs := this.locations
	numLocs := len(locs)
	return numLocs
}

/*
 * Create CSV database from byte slice.
 */
func FromBytes(data []byte) (geo.Database, error) {
	db := &databaseStruct{}
	errResult := error(nil)
	fd := bytes.NewReader(data)
	r := csv.NewReader(fd)
	r.FieldsPerRecord = 3
	records, err := r.ReadAll()

	/*
	 * Check if an error occured during reading.
	 */
	if err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Error occured during reading: %s", msg)
	} else {
		numLocations := len(records)
		locs := make([]locationStruct, numLocations)

		/*
		 * Iterate over the records.
		 */
		for i, record := range records {
			timestampString := record[0]
			timestamp, errTimestamp := db.parseTimestamp(timestampString)
			latitudeString := record[1]
			latitude, errLatitude := db.parseLatitude(latitudeString)
			longitudeString := record[2]
			longitude, errLongitude := db.parseLongitude(longitudeString)

			/*
			 * Check for parse errors.
			 */
			if errTimestamp != nil {

				/*
				 * Store the first parse error.
				 */
				if errResult == nil {
					msg := errTimestamp.Error()
					errResult = fmt.Errorf("Error parsing timestamp of record %d: %s", i, msg)
				}

			} else if errLatitude != nil {

				/*
				 * Store the first parse error.
				 */
				if errResult == nil {
					msg := errLatitude.Error()
					errResult = fmt.Errorf("Error parsing latitude of record %d: %s", i, msg)
				}

			} else if errLongitude != nil {

				/*
				 * Store the first parse error.
				 */
				if errResult == nil {
					msg := errLongitude.Error()
					errResult = fmt.Errorf("Error parsing longitude of record %d: %s", i, msg)
				}

			} else {

				/*
				 * Create location.
				 */
				loc := locationStruct{
					timestamp:   timestamp,
					latitudeE7:  latitude,
					longitudeE7: longitude,
				}

				locs[i] = loc
			}

		}

		db.locations = locs
	}

	/*
	 * Do not return database when an error occured.
	 */
	if errResult != nil {
		db = nil
	}

	return db, errResult
}
