package opengeodb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/andrepxx/location-visualizer/geo"
)

const (
	BITS_PER_BYTE        = 8
	MAGIC_NUMBER         = 0x47656f44420a0004
	SIZE_DATABASE_ENTRY  = 14
	SIZE_DATABASE_HEADER = 10
	SIZE_MAGIC           = 8
	SIZE_TIMESTAMP       = 6
	SIZE_COORDINATE      = 4
)

/*
 * Data structure representing a single location.
 */
type locationStruct struct {
	timestampMs uint64
	latitudeE7  int32
	longitudeE7 int32
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
 * this GPS location was recorded.
 */
func (this *locationStruct) Timestamp() uint64 {
	timestamp := this.timestampMs
	return timestamp
}

/*
 * Data structure representing a geo database in OpenGeoDB format.
 */
type databaseStruct struct {
	fd *bytes.Reader
}

/*
 * Validate file header and get version number.
 */
func (this *databaseStruct) validateHeaderAndGetVersion() (byte, byte, error) {
	fd := this.fd
	size := fd.Size()
	majorVersion := byte(0)
	minorVersion := byte(0)
	errResult := error(nil)

	/*
	 * Validate file size.
	 */
	if size < SIZE_DATABASE_HEADER {
		errResult = fmt.Errorf("Failed to read database header: Header size is %d bytes, but file size is only %d bytes.", SIZE_DATABASE_HEADER, size)
	} else {
		buf := make([]byte, SIZE_DATABASE_HEADER)
		n, err := fd.ReadAt(buf, 0)

		/*
		 * Check if read was successful.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to read database header: %s", msg)
		} else if n != SIZE_DATABASE_HEADER {
			errResult = fmt.Errorf("Failed to read database header: Header size is %d bytes, but read operation only returned %d bytes.", SIZE_DATABASE_HEADER, n)
		} else {
			endian := binary.BigEndian
			bufMagic := buf[0:SIZE_MAGIC]
			magic := endian.Uint64(bufMagic)

			/*
			 * Check if magic number matches.
			 */
			if magic != MAGIC_NUMBER {
				errResult = fmt.Errorf("Failed to read database header: Magic number does not match. Expected 0x%016x, found 0x%016x.", MAGIC_NUMBER, magic)
			} else {
				majorVersion = buf[SIZE_MAGIC]
				minorVersion = buf[SIZE_MAGIC+1]
			}

		}

	}

	return majorVersion, minorVersion, errResult
}

/*
 * The location stored at the given index in this database.
 */
func (this *databaseStruct) LocationAt(idx int) (geo.Location, error) {
	numLocations := this.LocationCount()

	/*
	 * If entry exists, read it.
	 */
	if idx < 0 || idx >= numLocations {
		return nil, fmt.Errorf("Index %d out of range", idx)
	} else {
		idx64 := int64(idx)
		offset := SIZE_DATABASE_HEADER + (idx64 * SIZE_DATABASE_ENTRY)
		fd := this.fd
		entry := make([]byte, SIZE_DATABASE_ENTRY)
		n, err := fd.ReadAt(entry, offset)

		/*
		 * Check if read error occured.
		 */
		if err != nil {
			msg := err.Error()
			return nil, fmt.Errorf("Error reading entry number %d: %s", idx, msg)
		} else if n != SIZE_DATABASE_ENTRY {
			return nil, fmt.Errorf("Error reading entry number %d: Expected %d bytes, read %d.", idx, SIZE_DATABASE_ENTRY, n)
		} else {
			timestamp := uint64(0)
			base := int(0)

			/*
			 * Read timestamp.
			 */
			for i := 0; i < SIZE_TIMESTAMP; i++ {
				offs := base + i
				byt := entry[offs]
				byt64 := uint64(byt)
				timestamp <<= BITS_PER_BYTE
				timestamp |= byt64
			}

			base += SIZE_TIMESTAMP
			longitude := uint32(0)

			/*
			 * Read longitude.
			 */
			for i := 0; i < SIZE_COORDINATE; i++ {
				offs := base + i
				byt := entry[offs]
				byt32 := uint32(byt)
				longitude <<= BITS_PER_BYTE
				longitude |= byt32
			}

			longitudeSigned := int32(longitude)
			base += SIZE_COORDINATE
			latitude := uint32(0)

			/*
			 * Read latitude.
			 */
			for i := 0; i < SIZE_COORDINATE; i++ {
				offs := base + i
				byt := entry[offs]
				byt32 := uint32(byt)
				latitude <<= BITS_PER_BYTE
				latitude |= byt32
			}

			latitudeSigned := int32(latitude)
			base += SIZE_COORDINATE

			/*
			 * Check if we arrive at desired entry size.
			 */
			if base != SIZE_DATABASE_ENTRY {
				panic("Database entry size does not match!")
			}

			/*
			 * Create location
			 */
			loc := locationStruct{
				timestampMs: timestamp,
				latitudeE7:  longitudeSigned,
				longitudeE7: latitudeSigned,
			}

			return &loc, nil
		}

	}

}

/*
 * The number of locations stored in this database.
 */
func (this *databaseStruct) LocationCount() int {
	numEntries := int(math.MaxInt)
	fd := this.fd
	numBytes := fd.Size()

	/*
	 * There can only be entries when a header is present.
	 */
	if numBytes >= SIZE_DATABASE_HEADER {
		sizeEntries := numBytes - SIZE_DATABASE_HEADER
		numEntries64 := sizeEntries / SIZE_DATABASE_ENTRY

		/*
		 * Store number of entries, if it fits into integer.
		 */
		if numEntries64 <= math.MaxInt {
			numEntries = int(numEntries64)
		}

	}

	return numEntries
}

/*
 * Create OpenGeoDB database from byte slice.
 */
func FromBytes(data []byte) (geo.Database, error) {
	r := bytes.NewReader(data)

	/*
	 * Create data structure for database.
	 */
	db := databaseStruct{
		fd: r,
	}

	major, minor, err := db.validateHeaderAndGetVersion()

	/*
	 * Check if error occured.
	 */
	if err != nil {
		return nil, err
	} else {

		/*
		 * Decide on file version.
		 */
		switch {
		case major == 1 && minor == 0:
			return &db, nil
		default:
			return nil, fmt.Errorf("Unsupported version: v%d.%d", major, minor)
		}

	}

}
