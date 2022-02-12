package geo

import (
	"encoding/json"
	"fmt"
	"github.com/andrepxx/sydney/coordinates"
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
	Coordinates() coordinates.Geographic
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
 * Data structure representing a location optimized for faster
 * access.
 */
type optimizedLocationStruct struct {
	timestampMs uint64
	latitude    float64
	longitude   float64
}

/*
 * Top-level GeoJSON element.
 */
type GeoJSON interface {
	LocationAt(idx int) (Location, error)
	LocationCount() int
}

/*
 * Data structure representing the top-level GeoJSON element.
 */
type geoJSONStruct struct {
	Locations []locationStruct `json:"locations"`
}

/*
 * Returns the geographic coordinates of this location.
 * By convention, these are in radians.
 */
func (this *locationStruct) Coordinates() coordinates.Geographic {
	longitudeE7 := this.LongitudeE7
	longitude := (DEGREES_TO_RADIANS * 1e-7) * float64(longitudeE7)
	latitudeE7 := this.LatitudeE7
	latitude := (DEGREES_TO_RADIANS * 1e-7) * float64(latitudeE7)
	coords := coordinates.CreateGeographic(longitude, latitude)
	return coords
}

/*
 * Returns a location optimized for faster access.
 */
func (this *locationStruct) Optimize() Location {
	coords := this.Coordinates()
	lng := coords.Longitude()
	lat := coords.Latitude()
	ts := this.Timestamp()

	/*
	 * The optimized location structure.
	 */
	loc := optimizedLocationStruct{
		timestampMs: ts,
		latitude:    lat,
		longitude:   lng,
	}

	return &loc
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
 * Returns the geographic coordinates of this location.
 * By convention, these are in radians.
 */
func (this *optimizedLocationStruct) Coordinates() coordinates.Geographic {
	longitude := this.longitude
	latitude := this.latitude
	coords := coordinates.CreateGeographic(longitude, latitude)
	return coords
}

/*
 * Returns a location optimized for faster access.
 */
func (this *optimizedLocationStruct) Optimize() Location {
	return this
}

/*
 * Returns the timestamp (in milliseconds since the Epoch) when
 * this GPS location was recorded.
 */
func (this *optimizedLocationStruct) Timestamp() uint64 {
	timestampMs := this.timestampMs
	return timestampMs
}

/*
 * The location stored at the given index in this GeoJSON structure.
 */
func (this *geoJSONStruct) LocationAt(idx int) (Location, error) {
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
 * The number of locations stored in this GeoJSON structure.
 */
func (this *geoJSONStruct) LocationCount() int {
	locs := this.Locations
	numLocs := len(locs)
	return numLocs
}

/*
 * Create GeoJSON structure from byte slice.
 */
func FromBytes(data []byte) (GeoJSON, error) {
	geoj := &geoJSONStruct{}
	err := json.Unmarshal(data, geoj)

	/*
	 * Check if an error occured during unmarshalling.
	 */
	if err != nil {
		msg := err.Error()
		return nil, fmt.Errorf("Error occured during unmarshalling: %s", msg)
	} else {
		return geoj, nil
	}

}
