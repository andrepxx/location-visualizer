package gpx

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/andrepxx/location-visualizer/geo"
	"github.com/andrepxx/location-visualizer/math"
)

/*
 * Data structure representing a track point in XML.
 */
type xmlTrackPointStruct struct {
	XMLName   xml.Name `xml:"trkpt"`
	Latitude  string   `xml:"lat,attr"`
	Longitude string   `xml:"lon,attr"`
	Timestamp string   `xml:"time"`
}

/*
 * Data structure representing a track segment in XML.
 */
type xmlTrackSegmentStruct struct {
	XMLName xml.Name              `xml:"trkseg"`
	Points  []xmlTrackPointStruct `xml:"trkpt"`
}

/*
 * Data structure representing a track in XML.
 */
type xmlTrackStruct struct {
	XMLName  xml.Name                `xml:"trk"`
	Segments []xmlTrackSegmentStruct `xml:"trkseg"`
}

/*
 * Data structure representing the XML root element.
 */
type xmlRootStruct struct {
	XMLName xml.Name         `xml:"gpx"`
	Tracks  []xmlTrackStruct `xml:"trk"`
}

/*
 * Data structure representing a single location.
 */
type locationStruct struct {
	LatitudeE7  int32
	LongitudeE7 int32
	TimestampMs uint64
}

/*
 * Data structure representing a location database imported from GPX.
 */
type databaseStruct struct {
	Locations []locationStruct
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
	timestamp := this.TimestampMs
	return timestamp
}

/*
 * The location stored at the given index in this database.
 */
func (this *databaseStruct) LocationAt(idx int) (geo.Location, error) {
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
 * Create GPX database from byte slice.
 */
func FromBytes(data []byte) (geo.Database, error) {
	root := xmlRootStruct{}
	err := xml.Unmarshal(data, &root)

	/*
	 * Check if an error occured during unmarshalling.
	 */
	if err != nil {
		msg := err.Error()
		return nil, fmt.Errorf("Error occured during unmarshalling: %s", msg)
	} else {
		locs := []locationStruct{}
		tracks := root.Tracks

		/*
		 * Iterate over tracks.
		 */
		for _, track := range tracks {
			segments := track.Segments

			/*
			 * Iterate over segments.
			 */
			for _, segment := range segments {
				points := segment.Points
				numPoints := len(points)
				currentLocs := make([]locationStruct, numPoints)

				/*
				 * Iterate over points.
				 */
				for i, point := range points {
					latitudeString := point.Latitude
					latitudeE7, _ := math.ParseFixed32(latitudeString, 7)
					longitudeString := point.Longitude
					longitudeE7, _ := math.ParseFixed32(longitudeString, 7)
					timestampString := point.Timestamp
					timestamp := uint64(0)
					layout := time.RFC3339Nano
					location := time.UTC
					parsedTime, err := time.ParseInLocation(layout, timestampString, location)

					/*
					 * ParseInLocation does not specify the result on error.
					 */
					if err != nil {
						timestamp = 0
					} else {
						unixMs := parsedTime.UnixMilli()
						timestamp = uint64(unixMs)
					}

					/*
					 * Create location structure.
					 */
					currentLocs[i] = locationStruct{
						LatitudeE7:  latitudeE7,
						LongitudeE7: longitudeE7,
						TimestampMs: timestamp,
					}

				}

				locs = append(locs, currentLocs...)
			}

		}

		/*
		 * Create new database.
		 */
		db := databaseStruct{
			Locations: locs,
		}

		return &db, nil
	}

}
