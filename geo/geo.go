package geo

import (
	"encoding/json"
	"fmt"
	"github.com/andrepxx/sydney/coordinates"
	"github.com/andrepxx/sydney/projection"
	"math"
	"strconv"
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
	Optimize() Location
	Projected() coordinates.Cartesian
	Timestamp() uint64
}

/*
 * Data structure representing a GeoJSON location.
 */
type locationStruct struct {
	LatitudeE7  int32  `json:"latitudeE7"`
	LongitudeE7 int32  `json:"longitudeE7"`
	TimestampMs string `json:"timestampMs"`
}

/*
 * Data structure representing a location optimized for faster
 * access.
 */
type optimizedLocationStruct struct {
	latitude    float64
	longitude   float64
	timestampMs uint64
	x           float64
	y           float64
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
	geo := this.Coordinates()
	lng := geo.Longitude()
	lat := geo.Latitude()
	ts := this.Timestamp()
	cart := this.Projected()
	x := cart.X()
	y := cart.Y()

	/*
	 * The optimized location structure.
	 */
	loc := optimizedLocationStruct{
		latitude:    lat,
		longitude:   lng,
		timestampMs: ts,
		x:           x,
		y:           y,
	}

	return &loc
}

/*
 * Returns the Mercator projected coordinates of this location.
 */
func (this *locationStruct) Projected() coordinates.Cartesian {
	proj := projection.Mercator()
	geo := this.Coordinates()
	cart := proj.Forward(geo)
	return cart
}

/*
 * Returns the timestamp (in milliseconds since the Epoch) when
 * this GPS location was recorded.
 */
func (this *locationStruct) Timestamp() uint64 {
	timestampMs := this.TimestampMs
	timestamp, _ := strconv.ParseUint(timestampMs, 10, 64)
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
 * Returns the Mercator projected coordinates of this location.
 */
func (this *optimizedLocationStruct) Projected() coordinates.Cartesian {
	x := this.x
	y := this.y
	coords := coordinates.CreateCartesian(x, y)
	return coords
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
	geo := &geoJSONStruct{}
	err := json.Unmarshal(data, geo)

	/*
	 * Check if an error occured during unmarshalling.
	 */
	if err != nil {
		msg := err.Error()
		return nil, fmt.Errorf("Error occured during unmarshalling: %s", msg)
	} else {
		return geo, nil
	}

}
