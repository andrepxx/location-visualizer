package geojson

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

/*
 * Mathematical constants.
 */
const (
	DEGREES_TO_RADIANS = math.Pi / 180.0
)

/*
 * A location in GeoJSON representation.
 */
type Location interface {
	Latitude() int32
	Longitude() int32
	Timestamp() uint64
}

/*
 * Data structure representing a GeoJSON location.
 */
type locationStruct struct {
	LatitudeE7   int32  `json:"latitudeE7"`
	LongitudeE7  int32  `json:"longitudeE7"`
	TimestampMs  string `json:"timestampMs"`
	TimestampISO string `json:"timestamp"`
}

/*
 * Top-level GeoJSON element.
 */
type Database interface {
	LocationAt(idx int) (Location, error)
	LocationCount() int
}

/*
 * Data structure representing the top-level GeoJSON element.
 */
type databaseStruct struct {
	Locations []locationStruct `json:"locations"`
}

/*
 * Returns the latitude of this location.
 */
func (this *locationStruct) Latitude() int32 {
	latitudeE7 := this.LatitudeE7
	return latitudeE7
}

/*
 * Returns the longitude of this location.
 */
func (this *locationStruct) Longitude() int32 {
	longitudeE7 := this.LongitudeE7
	return longitudeE7
}

/*
 * Returns the timestamp (in milliseconds since the Epoch) when
 * this GPS location was recorded.
 */
func (this *locationStruct) Timestamp() uint64 {
	timestampMs := this.TimestampMs
	timestampISO := this.TimestampISO
	timestamp := uint64(0)

	/*
	 * The timestamp format was changed around the end of 2021 / beginning
	 * of 2022.
	 *
	 * We try to support both so that we can parse old and new exports.
	 */
	if timestampMs != "" {
		timestamp, _ = strconv.ParseUint(timestampMs, 10, 64)
	} else if timestampISO != "" {
		layout := time.RFC3339Nano
		location := time.UTC
		parsedTime, err := time.ParseInLocation(layout, timestampISO, location)

		/*
		 * ParseInLocation does not specify the result on error.
		 */
		if err != nil {
			timestamp = 0
		} else {
			unixMs := parsedTime.UnixMilli()
			timestamp = uint64(unixMs)
		}

	}

	return timestamp
}

/*
 * The location stored at the given index in this database.
 */
func (this *databaseStruct) LocationAt(idx int) (Location, error) {
	locs := this.Locations
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
	locs := this.Locations
	numLocs := len(locs)
	return numLocs
}

/*
 * Create GeoJSON database from byte slice.
 */
func FromBytes(data []byte) (Database, error) {
	db := &databaseStruct{}
	err := json.Unmarshal(data, db)

	/*
	 * Check if an error occured during unmarshalling.
	 */
	if err != nil {
		msg := err.Error()
		return nil, fmt.Errorf("Error occured during unmarshalling: %s", msg)
	} else {
		return db, nil
	}

}
