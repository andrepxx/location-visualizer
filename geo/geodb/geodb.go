package geodb

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
)

/*
 * Constants for the geographical database.
 */
const (
	MAGIC_NUMBER         = 0x47656f44420a0004
	SIZE_DATABASE_ENTRY  = 14
	SIZE_DATABASE_HEADER = 10
	SIZE_TIMESTAMP       = 6
	VERSION_MAJOR        = 1
	VERSION_MINOR        = 0
)

/*
 * States for JSON serializer.
 */
const (
	JSON_STREAM_HEADER = iota
	JSON_STREAM_ENTRIES
	JSON_STREAM_TRAILER
	JSON_STREAM_EOF
	JSON_STREAM_ERROR
)

/*
 * Indentation direction.
 */
const (
	JSON_INDENT_IN   = 1
	JSON_INDENT_NONE = 0
	JSON_INDENT_OUT  = -1
)

/*
 * A geographic location stored in the geo database.
 */
type Location struct {
	Timestamp   uint64
	LatitudeE7  int32
	LongitudeE7 int32
}

/*
 * A database storing geographic data.
 */
type Database interface {
	Append(loc *Location) error
	Close()
	LocationCount() uint32
	ReadLocations(offset uint32, target []Location) (uint32, error)
	SerializeBinary() io.ReadSeekCloser
	SerializeCSV() io.ReadCloser
	SerializeJSON(pretty bool) io.ReadCloser
	Sort() error
}

/*
 * Interface that a storage backing a database will have to implement.
 * Typically, this storage will be a file.
 */
type Storage interface {
	ReadAt(buf []byte, offset int64) (int, error)
	Seek(offset int64, whence int) (int64, error)
	WriteAt(buf []byte, offset int64) (int, error)
}

/*
 * The header of a location database.
 */
type databaseHeaderStruct struct {
	Magic        uint64
	VersionMajor uint8
	VersionMinor uint8
}

/*
 * Each database entry consists of a 48 bit time stamp storing milliseconds
 * since the Epoch, as well as longitude and latitude values in degrees, stored
 * as fixed-point values with a fixed exponent of 10^(-7).
 */
type databaseEntryStruct struct {
	TimestampMSB uint16
	TimestampLSB uint32
	LatitudeE7   int32
	LongitudeE7  int32
}

/*
 * Database accessor.
 */
type databaseStruct struct {
	mutex         sync.RWMutex
	fd            Storage
	locationCount uint32
}

/*
 * Data structure for serializing the database into binary format.
 */
type databaseBinarySerializerStruct struct {
	mutex  sync.Mutex
	db     *databaseStruct
	offset uint64
}

/*
 * Data structure for serializing the database into CSV format.
 */
type databaseCsvSerializerStruct struct {
	mutex      sync.Mutex
	csvWriter  *csv.Writer
	db         *databaseStruct
	entryId    uint32
	lineBuffer *strings.Builder
	lineOffset int
}

/*
 * Data structure for serializing the database into GeoJSON format.
 */
type databaseJsonSerializerStruct struct {
	mutex   sync.Mutex
	buffer  *strings.Builder
	db      *databaseStruct
	entryId uint32
	indent  uint16
	pretty  bool
	state   int
}

/*
 * Data structure for sorting the database.
 */
type databaseSorterStruct struct {
	db *databaseStruct
}

/*
 * Internal sorting function.
 *
 * Converts panic into proper error handling.
 *
 * Assumes that the database is locked for writing.
 */
func (this *databaseStruct) sort() (err error) {

	/*
	 * Sorting may panic, for example on I/O errors.
	 */
	defer func() {
		r := recover()

		/*
		 * Check if panic occured during sorting.
		 */
		if r != nil {
			msg, ok := r.(string)

			/*
			 * Check if string was passed to panic.
			 */
			if ok {
				err = fmt.Errorf("Error during sorting: %s", msg)
			} else {
				err = fmt.Errorf("Unknown error during sorting.")
				stack := debug.Stack()
				fmt.Printf("%s\n", string(stack))
			}

		}

	}()

	/*
	 * Create database accessor for the sort algorithm.
	 */
	sorter := databaseSorterStruct{
		db: this,
	}

	sort.Stable(&sorter)
	return
}

/*
 * Appends the location pointed to by loc to the database.
 *
 * When loc == nil, this is a no-op.
 *
 * When loc != nil, this temporarily locks the database for write access.
 */
func (this *databaseStruct) Append(loc *Location) error {
	errResult := error(nil)

	/*
	 * Check if we got a location.
	 */
	if loc == nil {
		errResult = fmt.Errorf("%s", "Location must not be nil!")
	} else {
		this.mutex.Lock()
		fd := this.fd
		locationCount := this.locationCount

		/*
		 * Check if there is an open file descriptor and space left to
		 * store another location.
		 */
		if fd == nil {
			errResult = fmt.Errorf("%s", "Database is closed.")
		} else if locationCount >= math.MaxUint32 {
			errResult = fmt.Errorf("Reached maximum number of stored locations: %d", math.MaxUint32)
		} else {
			timestamp := loc.Timestamp
			timestampMSB := uint16((timestamp & 0xffff00000000) >> 32)
			timestampLSB := uint32(timestamp & 0xffffffff)
			latitudeE7 := loc.LatitudeE7
			longitudeE7 := loc.LongitudeE7

			/*
			 * Create database entry.
			 */
			entry := databaseEntryStruct{
				TimestampMSB: timestampMSB,
				TimestampLSB: timestampLSB,
				LatitudeE7:   latitudeE7,
				LongitudeE7:  longitudeE7,
			}

			buf := bytes.Buffer{}
			buf.Grow(SIZE_DATABASE_ENTRY)
			endianness := binary.BigEndian
			err := binary.Write(&buf, endianness, entry)
			sizeWrittenBuf := buf.Len()

			/*
			 * Check if database header could be serialized.
			 */
			if err != nil {
				reason := err.Error()
				errResult = fmt.Errorf("Failed to serialize database entry: %s", reason)
			} else if sizeWrittenBuf != SIZE_DATABASE_ENTRY {
				errResult = fmt.Errorf("Unexpected size of database entry: Expected %d, got %d.", SIZE_DATABASE_ENTRY, sizeWrittenBuf)
			} else {
				content := buf.Next(sizeWrittenBuf)
				locationCount64 := int64(locationCount)
				offset := SIZE_DATABASE_HEADER + (SIZE_DATABASE_ENTRY * locationCount64)
				sizeWrittenFd, err := fd.WriteAt(content, offset)

				/*
				 * Check if buffer could be written to file.
				 */
				if err != nil {
					reason := err.Error()
					errResult = fmt.Errorf("Failed to write database entry: %s", reason)
				} else if sizeWrittenFd != sizeWrittenBuf {
					errResult = fmt.Errorf("Unexpected write size when writing database entry: Expected %d, got %d.", sizeWrittenBuf, sizeWrittenFd)
				} else {
					this.locationCount = locationCount + 1
				}

			}

		}

		this.mutex.Unlock()
	}

	return errResult
}

/*
 * Closes this database, releasing the associated file descriptor.
 *
 * NOTE: This does NOT close the file descriptor itself!
 *
 * If the database is already closed, this is a no-op.
 *
 * This temporarily locks the database for write access.
 */
func (this *databaseStruct) Close() {
	this.mutex.Lock()
	this.fd = nil
	this.locationCount = 0
	this.mutex.Unlock()
}

/*
 * Returns the number of locations stored in the database.
 *
 * On a closed database, this returns zero.
 *
 * This temporarily locks the database for read access.
 */
func (this *databaseStruct) LocationCount() uint32 {
	this.mutex.RLock()
	result := this.locationCount
	this.mutex.RUnlock()
	return result
}

/*
 * Reads locations from the database into target, starting at the provided
 * offset.
 *
 * Will fill the target buffer unless there are not enough locations left.
 *
 * Returns the number of locations read and whether read errors occured.
 *
 * When len(target) > 0, this temporarily locks the database for read access.
 */
func (this *databaseStruct) ReadLocations(offset uint32, target []Location) (uint32, error) {
	numReadErrors := uint64(0)
	firstReadErrorOffset := uint64(0)
	numDeserializationErrors := uint64(0)
	firstDeserializationErrorOffset := uint64(0)
	numLocationsTarget := len(target)
	numLocationsRead := uint32(0)

	/*
	 * Check if we have to read locations.
	 */
	if numLocationsTarget > 0 {
		this.mutex.RLock()
		locationCount := this.locationCount

		/*
		 * Check if we are in bounds.
		 */
		if offset < locationCount {
			numLocationsToRead := uint32(numLocationsTarget)

			/*
			 * Prevent overflow.
			 */
			if numLocationsToRead > math.MaxUint32 {
				numLocationsToRead = math.MaxUint32
			}

			numLocationsInFile := locationCount - offset

			/*
			 * We can only read as many locations as are in the file.
			 */
			if numLocationsToRead > numLocationsInFile {
				numLocationsToRead = numLocationsInFile
			}

			buf := make([]byte, SIZE_DATABASE_ENTRY)
			entry := databaseEntryStruct{}
			fd := this.fd
			endianness := binary.BigEndian

			/*
			 * Read locations from file.
			 */
			for idx := uint32(0); idx < numLocationsToRead; idx++ {
				offsetTotal := offset + idx
				offsetTotal64 := uint64(offsetTotal)
				offsetBytes := SIZE_DATABASE_HEADER + (SIZE_DATABASE_ENTRY * offsetTotal64)
				offsetBytesSigned := int64(offsetBytes)
				numBytesRead, err := fd.ReadAt(buf, offsetBytesSigned)

				/*
				 * If we read less bytes than expected, zero
				 * out part of the buffer.
				 */
				if numBytesRead < SIZE_DATABASE_ENTRY {
					zero := buf[numBytesRead:SIZE_DATABASE_ENTRY]

					/*
					 * Zero the unused part of the buffer.
					 */
					for i := range zero {
						zero[i] = 0
					}

				}

				/*
				 * Check for read error.
				 */
				if err != nil {

					/*
					 * If this is the first read error,
					 * store offset.
					 */
					if numReadErrors == 0 {
						firstReadErrorOffset = offsetBytes
					}

					/*
					 * Count read errors.
					 */
					if numReadErrors < math.MaxUint64 {
						numReadErrors++
					}

				}

				rd := bytes.NewReader(buf)
				err = binary.Read(rd, endianness, &entry)

				/*
				 * Check if database entry could be deserialized.
				 */
				if err != nil {
					target[idx] = Location{}

					/*
					 * If this is the first deserialization
					 * error, store offset.
					 */
					if numDeserializationErrors == 0 {
						firstDeserializationErrorOffset = offsetBytes
					}

					/*
					 * Count deserialization errors.
					 */
					if numDeserializationErrors < math.MaxUint64 {
						numDeserializationErrors++
					}

				} else {
					timestampMSB := entry.TimestampMSB
					timestampMSB64 := uint64(timestampMSB)
					timestampLSB := entry.TimestampLSB
					timestampLSB64 := uint64(timestampLSB)
					timestamp := (timestampMSB64 << 32) | timestampLSB64
					latitudeE7 := entry.LatitudeE7
					longitudeE7 := entry.LongitudeE7

					/*
					 * Fill in location structure.
					 */
					target[idx] = Location{
						Timestamp:   timestamp,
						LatitudeE7:  latitudeE7,
						LongitudeE7: longitudeE7,
					}

				}

			}

			numLocationsRead = numLocationsToRead
		}

		this.mutex.RUnlock()
	}

	errResult := error(nil)

	/*
	 * Check for read errors and deserialization errors.
	 */
	switch {
	case (numReadErrors != 0) && (numDeserializationErrors == 0):
		errResult = fmt.Errorf("Encountered %d read errors, first at offset %d (0x%016x).", numReadErrors, firstReadErrorOffset, firstReadErrorOffset)
	case (numReadErrors == 0) && (numDeserializationErrors != 0):
		errResult = fmt.Errorf("Encountered %d deserialization errors, first at offset %d (0x%016x).", numDeserializationErrors, firstDeserializationErrorOffset, firstDeserializationErrorOffset)
	case (numReadErrors != 0) && (numDeserializationErrors != 0):
		errResult = fmt.Errorf("Encountered %d read errors, first at offset %d (0x%016x), and %d deserialization errors, first at offset %d (0x%016x).", numReadErrors, firstReadErrorOffset, firstReadErrorOffset, numDeserializationErrors, firstDeserializationErrorOffset, firstDeserializationErrorOffset)
	}

	return numLocationsRead, errResult
}

/*
 * Locks the database for read access and provides a ReadSeekCloser
 * granting random access to the database in binary format.
 *
 * Closing the returned ReadSeekCloser yields the lock on the database.
 */
func (this *databaseStruct) SerializeBinary() io.ReadSeekCloser {
	this.mutex.RLock()

	/*
	 * Create database binary serializer.
	 */
	s := databaseBinarySerializerStruct{
		db: this,
	}

	return &s
}

/*
 * Locks the database for read access and provides a ReadCloser granting
 * sequential access to the database in CSV format.
 *
 * CSV data will be generated on-the-fly while reading from the provided
 * ReadCloser.
 *
 * Closing the returned ReadCloser yields the lock on the database.
 */
func (this *databaseStruct) SerializeCSV() io.ReadCloser {
	this.mutex.RLock()
	buf := &strings.Builder{}
	w := csv.NewWriter(buf)

	/*
	 * Create database CSV serializer.
	 */
	s := databaseCsvSerializerStruct{
		csvWriter:  w,
		db:         this,
		lineBuffer: buf,
	}

	return &s
}

/*
 * Locks the database for read access and provides a ReadCloser granting
 * sequential access to the database in JSON format.
 *
 * JSON data will be generated on-the-fly while reading from the provided
 * ReadCloser.
 *
 * - When pretty == true, data will be pretty-printed for human consumption.
 * - When pretty == false, data will be compact for machine consumption.
 *
 * Closing the returned ReadCloser yields the lock on the database.
 */
func (this *databaseStruct) SerializeJSON(pretty bool) io.ReadCloser {
	this.mutex.RLock()
	buf := &strings.Builder{}

	/*
	 * Create database JSON serializer.
	 */
	s := databaseJsonSerializerStruct{
		buffer: buf,
		db:     this,
		pretty: pretty,
		state:  JSON_STREAM_HEADER,
	}

	return &s
}

/*
 * Sorts entries in the database by (ascending) time stamp using a stable
 * sorting algorithm.
 *
 * If the database is closed, this is a no-op.
 *
 * This temporarily locks the database for write access.
 */
func (this *databaseStruct) Sort() error {
	result := error(nil)
	this.mutex.Lock()
	fd := this.fd

	/*
	 * Only sort database if it is still open.
	 */
	if fd != nil {
		result = this.sort()
	}

	this.mutex.Unlock()
	return result
}

/*
 * Implements the Read function from io.ReadSeekCloser.
 */
func (this *databaseBinarySerializerStruct) Read(buf []byte) (int, error) {
	result := int(0)
	errResult := error(nil)
	this.mutex.Lock()
	db := this.db

	/*
	 * Check if serializer is still open.
	 */
	if db == nil {
		errResult = fmt.Errorf("%s", "Database serializer is already closed.")
	} else {
		fd := db.fd

		/*
		 * Check if file descriptor is still open.
		 */
		if fd == nil {
			errResult = fmt.Errorf("%s", "Database is already closed.")
		} else {
			locationCount := db.locationCount
			locationCount64 := uint64(locationCount)
			size := SIZE_DATABASE_HEADER + (SIZE_DATABASE_ENTRY * locationCount64)
			offset := this.offset
			bytesInFile := size - offset
			bufSize := len(buf)
			bytesToRead := uint64(bufSize)

			/*
			 * Limit bytes to read to file size.
			 */
			if bytesToRead > bytesInFile {
				bytesToRead = bytesInFile
			}

			bufTarget := buf[0:bytesToRead]
			offsetSigned := int64(offset)

			/*
			 * Prevent overflow.
			 */
			if offsetSigned < 0 {
				errResult = fmt.Errorf("%s", "Overflow.")
			} else {
				bytesRead, err := fd.ReadAt(bufTarget, offsetSigned)
				bytesRead64 := uint64(bytesRead)

				/*
				 * Prevent out of bounds errors and implausible results.
				 */
				if bytesRead < 0 {
					bytesRead = 0
					bytesRead64 = uint64(bytesRead)
				} else if bytesRead64 > bytesToRead {
					bytesRead = int(bytesToRead)
					bytesRead64 = bytesToRead
				}

				/*
				 * Handle I/O errors.
				 */
				if err == io.EOF {

					/*
					 * Check if we read as many bytes as expected.
					 */
					if bytesRead64 < bytesToRead {
						errResult = io.ErrUnexpectedEOF
					}

				} else if err != nil {
					msg := err.Error()
					errResult = fmt.Errorf("I/O error: %s", msg)
					bytesRead = 0
					bytesRead64 = 0
				}

				bufToZero := buf[bytesRead:bufSize]

				/*
				 * Zero out remaining part of the buffer.
				 */
				for i := range bufToZero {
					bufToZero[i] = 0
				}

				offset += bytesRead64
				result = bytesRead
			}

			this.offset = offset
		}

	}

	this.mutex.Unlock()
	return result, errResult
}

/*
 * Implements the Seek function from io.ReadSeekCloser.
 */
func (this *databaseBinarySerializerStruct) Seek(offset int64, whence int) (int64, error) {
	result := int64(0)
	errResult := error(nil)
	this.mutex.Lock()
	db := this.db

	/*
	 * Check if serializer is still open.
	 */
	if db == nil {
		errResult = fmt.Errorf("%s", "Database serializer is already closed.")
	} else {
		fd := db.fd

		/*
		 * Check if file descriptor is still open.
		 */
		if fd == nil {
			errResult = fmt.Errorf("%s", "Database is already closed.")
		} else {
			locationCount := db.locationCount
			locationCount64 := uint64(locationCount)
			size := SIZE_DATABASE_HEADER + (SIZE_DATABASE_ENTRY * locationCount64)
			offset64 := uint64(offset)
			offsetCurrent := this.offset

			/*
			 * Decide relative to what to seek.
			 */
			switch whence {
			case io.SeekStart:

				/*
				 * Check if absolute offset is negative.
				 */
				if offset < 0 {
					errResult = fmt.Errorf("%s", "Cannot seek to negative absolute offset.")
				} else {
					offsetCurrent = offset64
					result = int64(offsetCurrent)
				}

			case io.SeekCurrent:
				offsetNew := offsetCurrent + offset64

				/*
				 * Prevent numeric overflow.
				 */
				if ((offset > 0) && (offsetNew <= offsetCurrent)) || ((offset < 0) && (offsetNew >= offsetCurrent)) {
					errResult = fmt.Errorf("%s", "Overflow or negative target offset.")
				} else {
					offsetCurrent = offsetNew
					result = int64(offsetCurrent)
				}

			case io.SeekEnd:
				offsetNew := size + offset64

				/*
				 * Prevent numeric overflow.
				 */
				if ((offset > 0) && (offsetNew <= size)) || ((offset < 0) && (offsetNew >= size)) {
					errResult = fmt.Errorf("%s", "Overflow or negative target offset.")
				} else {
					offsetCurrent = offsetNew
					result = int64(offsetCurrent)
				}

			default:
				errResult = fmt.Errorf("Seek: Invalid value for 'whence': %d", whence)
			}

			this.offset = offsetCurrent
		}

	}

	this.mutex.Unlock()
	return result, errResult
}

/*
 * Implements the Close function from io.ReadSeekCloser.
 *
 * This will yield the read lock on the underlying database.
 */
func (this *databaseBinarySerializerStruct) Close() error {
	result := error(nil)
	this.mutex.Lock()
	db := this.db

	/*
	 * Check if serializer is already closed.
	 */
	if db == nil {
		result = fmt.Errorf("%s", "Database serializer is already closed.")
	} else {
		db.mutex.RUnlock()
		this.db = nil
	}

	this.mutex.Unlock()
	return result
}

/*
 * Format timestamp as string value.
 */
func (this *databaseCsvSerializerStruct) formatTimestamp(timestamp uint64) string {
	timestampSigned := int64(timestamp)
	t := time.UnixMilli(timestampSigned)
	utcTime := t.UTC()
	result := utcTime.Format(time.RFC3339Nano)
	return result
}

/*
 * Format latitude as string value.
 */
func (this *databaseCsvSerializerStruct) formatLatitude(latitudeE7 int32) string {
	result := "<INVALID>"
	buf := fmt.Sprintf("%+08d", latitudeE7)
	bufSize := len(buf)

	/*
	 * Check that buffer has sufficient size.
	 */
	if bufSize >= 8 {
		sign := buf[0]
		direction := '?'

		/*
		 * Check sign of number.
		 */
		switch sign {
		case byte('+'):
			direction = 'N'
		case byte('-'):
			direction = 'S'
		}

		posDecimalPoint := bufSize - 7
		leftOfPoint := buf[1:posDecimalPoint]
		rightOfPoint := buf[posDecimalPoint:bufSize]
		outputSize := bufSize + 1
		builder := strings.Builder{}
		builder.Grow(outputSize)
		builder.WriteString(leftOfPoint)
		builder.WriteRune('.')
		builder.WriteString(rightOfPoint)
		builder.WriteRune(direction)
		result = builder.String()
	}

	return result
}

/*
 * Format longitude as string value.
 */
func (this *databaseCsvSerializerStruct) formatLongitude(longitudeE7 int32) string {
	result := "<INVALID>"
	buf := fmt.Sprintf("%+08d", longitudeE7)
	bufSize := len(buf)

	/*
	 * Check that buffer has sufficient size.
	 */
	if bufSize >= 8 {
		sign := buf[0]
		direction := '?'

		/*
		 * Check sign of number.
		 */
		switch sign {
		case byte('+'):
			direction = 'E'
		case byte('-'):
			direction = 'W'
		}

		posDecimalPoint := bufSize - 7
		leftOfPoint := buf[1:posDecimalPoint]
		rightOfPoint := buf[posDecimalPoint:bufSize]
		outputSize := bufSize + 1
		builder := strings.Builder{}
		builder.Grow(outputSize)
		builder.WriteString(leftOfPoint)
		builder.WriteRune('.')
		builder.WriteString(rightOfPoint)
		builder.WriteRune(direction)
		result = builder.String()
	}

	return result
}

/*
 * Implements the Read function from io.ReadCloser.
 */
func (this *databaseCsvSerializerStruct) Read(buf []byte) (int, error) {
	numBytesToRead := len(buf)
	readBytes := int(0)
	errResult := error(nil)

	/*
	 * Check if we have to read bytes.
	 */
	if numBytesToRead > 0 {
		this.mutex.Lock()
		db := this.db

		/*
		 * Check if serializer is already closed.
		 */
		if db == nil {
			errResult = fmt.Errorf("%s", "Database serializer is already closed.")
		} else {
			numEntries := db.locationCount
			entryId := this.entryId
			csvWriter := this.csvWriter
			lineBuffer := this.lineBuffer
			line := lineBuffer.String()
			lineLength := len(line)
			lineOffset := this.lineOffset
			bufRead := make([]byte, SIZE_DATABASE_ENTRY)

			/*
			 * Continue until we reach the end of the file or
			 * filled the read buffer.
			 */
			for ((entryId < numEntries) || ((entryId == numEntries) && (lineLength > 0))) && (readBytes < numBytesToRead) && (errResult == nil) {
				lineFromOffset := line[lineOffset:]
				bufOffset := buf[readBytes:]
				n := copy(bufOffset, lineFromOffset)
				lineOffset += n
				readBytes += n

				/*
				 * If no bytes were copied, we have to update our buffers.
				 */
				if n == 0 {

					/*
					 * If there are no more entries, we have to clear our buffers.
					 *
					 * Otherwise, we will generate a new line.
					 */
					if entryId >= numEntries {
						lineBuffer.Reset()
						line = lineBuffer.String()
						lineLength = len(line)
						lineOffset = 0
					} else {
						entry := databaseEntryStruct{}
						fd := db.fd
						endianness := binary.BigEndian
						offset := uint64(entryId)
						offsetBytes := SIZE_DATABASE_HEADER + (SIZE_DATABASE_ENTRY * offset)
						offsetBytesSigned := int64(offsetBytes)
						numBytesRead, err := fd.ReadAt(bufRead, offsetBytesSigned)

						/*
						 * If we read less bytes than expected,
						 * zero out part of the buffer.
						 */
						if numBytesRead < SIZE_DATABASE_ENTRY {
							zero := bufRead[numBytesRead:SIZE_DATABASE_ENTRY]

							/*
							 * Zero the unused part of the buffer.
							 */
							for i := range zero {
								zero[i] = 0
							}

						}

						/*
						 * Check for read error.
						 */
						if err != nil {
							errResult = fmt.Errorf("Error reading from offset: 0x%016x", offsetBytes)
						} else {
							rd := bytes.NewReader(bufRead)
							err = binary.Read(rd, endianness, &entry)

							/*
							 * Check if database entry could be deserialized.
							 */
							if err != nil {
								errResult = fmt.Errorf("Error deserializing entry at offset: 0x%016x", offsetBytes)
							} else {
								timestampMSB := entry.TimestampMSB
								timestampMSB64 := uint64(timestampMSB)
								timestampLSB := entry.TimestampLSB
								timestampLSB64 := uint64(timestampLSB)
								timestamp := (timestampMSB64 << 32) | timestampLSB64
								latitudeE7 := entry.LatitudeE7
								longitudeE7 := entry.LongitudeE7
								timestampString := this.formatTimestamp(timestamp)
								latitudeString := this.formatLatitude(latitudeE7)
								longitudeString := this.formatLongitude(longitudeE7)

								/*
								 * Create record.
								 */
								record := []string{
									timestampString,
									latitudeString,
									longitudeString,
								}

								lineBuffer.Reset()
								csvWriter.Write(record)
								csvWriter.Flush()
								line = lineBuffer.String()
								lineLength = len(line)
								lineOffset = 0
							}

						}

						entryId++
					}

				}

			}

			/*
			 * Check for end of file condition.
			 */
			if (entryId > numEntries) || ((entryId == numEntries) && (lineLength == 0)) {
				errResult = io.EOF
			}

			this.entryId = entryId
			this.lineOffset = lineOffset
		}

		this.mutex.Unlock()
	}

	return readBytes, errResult
}

/*
 * Implements the Close function from io.ReadCloser.
 *
 * This will yield the read lock on the underlying database.
 */
func (this *databaseCsvSerializerStruct) Close() error {
	result := error(nil)
	this.mutex.Lock()
	db := this.db

	/*
	 * Check if serializer is already closed.
	 */
	if db == nil {
		result = fmt.Errorf("%s", "Database serializer is already closed.")
	} else {
		db.mutex.RUnlock()
		this.db = nil
	}

	this.mutex.Unlock()
	return result
}

/*
 * Begin a JSON list.
 */
func (this *databaseJsonSerializerStruct) beginList() {
	buffer := this.buffer
	buffer.WriteRune('[')
	this.startLine(JSON_INDENT_IN)
}

/*
 * Begin a JSON object.
 */
func (this *databaseJsonSerializerStruct) beginObject() {
	buffer := this.buffer
	buffer.WriteRune('{')
	this.startLine(JSON_INDENT_IN)
}

/*
 * Change the indentation depth.
 */
func (this *databaseJsonSerializerStruct) changeIndent(direction int) {
	indent := this.indent

	/*
	 * Decide on the indentation direction.
	 */
	switch direction {
	case JSON_INDENT_IN:

		/*
		 * Increase indent, preventing overflow.
		 */
		if indent < math.MaxUint16 {
			indent++
		}

	case JSON_INDENT_OUT:

		/*
		 * Decrease indent, preventing underflow.
		 */
		if indent > 0 {
			indent--
		}

	default:
		// Do nothing.
	}

	this.indent = indent
}

/*
 * End a JSON list.
 */
func (this *databaseJsonSerializerStruct) endList() {
	this.startLine(JSON_INDENT_OUT)
	buffer := this.buffer
	buffer.WriteRune(']')
}

/*
 * End a JSON object.
 */
func (this *databaseJsonSerializerStruct) endObject() {
	this.startLine(JSON_INDENT_OUT)
	buffer := this.buffer
	buffer.WriteRune('}')
}

/*
 * Format timestamp as string value.
 */
func (this *databaseJsonSerializerStruct) formatTimestamp(timestamp uint64) string {
	timestampSigned := int64(timestamp)
	t := time.UnixMilli(timestampSigned)
	utcTime := t.UTC()
	result := utcTime.Format(time.RFC3339Nano)
	return result
}

/*
 * Generate more JSON data.
 */
func (this *databaseJsonSerializerStruct) generateJSON() error {
	state := this.state
	errResult := error(nil)

	switch state {
	case JSON_STREAM_HEADER:
		this.beginObject()
		this.generateJSONForObjectKey("locations")
		this.beginList()
		state = JSON_STREAM_ENTRIES
	case JSON_STREAM_ENTRIES:
		err := this.generateJSONForNextEntry()

		/*
		 * Check for errors during serialization.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Error generating entry: %s", msg)
			state = JSON_STREAM_ERROR
		} else {
			moreAvailable := this.hasMoreEntries()

			/*
			 * If there are more entries to be serialized, write
			 * separator, otherwise transition to serializing the
			 * trailer.
			 */
			if moreAvailable {
				this.nextItem()
			} else {
				state = JSON_STREAM_TRAILER
			}

		}

	case JSON_STREAM_TRAILER:
		this.endList()
		this.endObject()
		state = JSON_STREAM_EOF
	case JSON_STREAM_EOF:
		errResult = io.EOF
	default:
		errResult = fmt.Errorf("%s", "Error during JSON serialization.")
	}

	this.state = state
	return errResult
}

/*
 * Generate JSON data for a key-value-pair.
 */
func (this *databaseJsonSerializerStruct) generateJSONForKeyValuePair(key string, value string, valueAsStringLiteral bool) {
	buffer := this.buffer
	this.generateJSONForObjectKey(key)
	valueLiteral := value

	/*
	 * Optionally, encode value as string literal.
	 */
	if valueAsStringLiteral {
		valueLiteral = this.toStringLiteral(value)
	}

	buffer.WriteString(valueLiteral)
}

/*
 * Generate JSON data for object key.
 */
func (this *databaseJsonSerializerStruct) generateJSONForObjectKey(key string) {
	pretty := this.pretty
	buffer := this.buffer
	keyLiteral := this.toStringLiteral(key)
	buffer.WriteString(keyLiteral)
	buffer.WriteRune(':')

	/*
	 * When pretty-printing, emit space after object key.
	 */
	if pretty {
		buffer.WriteRune(' ')
	}

}

/*
 * Generate JSON data for next entry in geographical database.
 */
func (this *databaseJsonSerializerStruct) generateJSONForNextEntry() error {
	errResult := error(nil)
	moreAvailable := this.hasMoreEntries()

	/*
	 * Check if more entries are available.
	 */
	if moreAvailable {
		db := this.db
		entryId := this.entryId
		entry := databaseEntryStruct{}
		fd := db.fd
		endianness := binary.BigEndian
		offset := uint64(entryId)
		offsetBytes := SIZE_DATABASE_HEADER + (SIZE_DATABASE_ENTRY * offset)
		offsetBytesSigned := int64(offsetBytes)
		bufRead := make([]byte, SIZE_DATABASE_ENTRY)
		numBytesRead, err := fd.ReadAt(bufRead, offsetBytesSigned)

		/*
		 * If we read less bytes than expected, zero out part of the
		 * buffer.
		 */
		if numBytesRead < SIZE_DATABASE_ENTRY {
			zero := bufRead[numBytesRead:SIZE_DATABASE_ENTRY]

			/*
			 * Zero the unused part of the buffer.
			 */
			for i := range zero {
				zero[i] = 0
			}

		}

		/*
		 * Check for read error.
		 */
		if err != nil {
			errResult = fmt.Errorf("Error reading from offset: 0x%016x", offsetBytes)
		} else {
			rd := bytes.NewReader(bufRead)
			err = binary.Read(rd, endianness, &entry)

			/*
			 * Check if database entry could be deserialized.
			 */
			if err != nil {
				errResult = fmt.Errorf("Error deserializing entry at offset: 0x%016x", offsetBytes)
			} else {
				timestampMSB := entry.TimestampMSB
				timestampMSB64 := uint64(timestampMSB)
				timestampLSB := entry.TimestampLSB
				timestampLSB64 := uint64(timestampLSB)
				timestamp := (timestampMSB64 << 32) | timestampLSB64
				latitudeE7 := entry.LatitudeE7
				longitudeE7 := entry.LongitudeE7
				timestampString := this.formatTimestamp(timestamp)
				timestampMsString := fmt.Sprintf("%d", timestamp)
				latitudeE7String := fmt.Sprintf("%d", latitudeE7)
				longitudeE7String := fmt.Sprintf("%d", longitudeE7)
				this.beginObject()
				this.generateJSONForKeyValuePair("timestamp", timestampString, true)
				this.nextItem()
				this.generateJSONForKeyValuePair("timestampMs", timestampMsString, true)
				this.nextItem()
				this.generateJSONForKeyValuePair("latitudeE7", latitudeE7String, false)
				this.nextItem()
				this.generateJSONForKeyValuePair("longitudeE7", longitudeE7String, false)
				this.endObject()
			}
		}

		entryId++
		this.entryId = entryId
	}

	return errResult
}

/*
 * Returns whether there are more entries in the database to be serialized.
 */
func (this *databaseJsonSerializerStruct) hasMoreEntries() bool {
	db := this.db
	entryId := this.entryId
	locationCount := db.locationCount
	result := entryId < locationCount
	return result
}

/*
 * Returns whether this byte is an ASCII control character.
 */
func (this *databaseJsonSerializerStruct) isControlCharacter(value rune) bool {
	result := (value < 0x20) || (value == 0x7f)
	return result
}

/*
 * Starts a new item, either in a list or an object.
 */
func (this *databaseJsonSerializerStruct) nextItem() {
	buffer := this.buffer
	buffer.WriteRune(',')
	pretty := this.pretty

	/*
	 * For pretty-printing, start new line for each item.
	 */
	if pretty {
		this.startLine(JSON_INDENT_NONE)
	}

}

/*
 * Begins a new line, including indentation.
 */
func (this *databaseJsonSerializerStruct) startLine(indentationDirection int) {
	pretty := this.pretty

	/*
	 * Only do this when pretty-printing JSON.
	 */
	if pretty {
		this.changeIndent(indentationDirection)
		indent := this.indent
		indentByte := uint8(indent)

		/*
		 * Limit indentation depth.
		 */
		if indent > math.MaxUint8 {
			indentByte = math.MaxUint8
		}

		buffer := this.buffer
		buffer.WriteRune('\n')

		/*
		 * Write indentation.
		 */
		for i := uint8(0); i < indentByte; i++ {
			buffer.WriteRune('\t')
		}

	}

}

/*
 * Convert a string value into a JSON string literal.
 */
func (this *databaseJsonSerializerStruct) toStringLiteral(value string) string {
	buf := strings.Builder{}
	buf.WriteRune('"')

	/*
	 * Iterate over the input string.
	 */
	for _, c := range value {

		/*
		 * Perform action depending on character.
		 */
		switch c {
		case '\\':
			buf.WriteString("\\\\")
		case '"':
			buf.WriteString("\\\"")
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		default:
			isControl := this.isControlCharacter(c)

			/*
			 * Escape control character.
			 */
			if isControl {
				uc := uint16(c)
				fmt.Fprintf(&buf, "\\u%04x", uc)
			} else {
				buf.WriteRune(c)
			}

		}

	}

	buf.WriteRune('"')
	result := buf.String()
	return result
}

/*
 * Implements the Read function from io.ReadCloser.
 */
func (this *databaseJsonSerializerStruct) Read(buf []byte) (int, error) {
	numBytesRead := 0
	errResult := error(nil)
	this.mutex.Lock()
	db := this.db

	/*
	 * Check if serializer is already closed.
	 */
	if db == nil {
		errResult = fmt.Errorf("%s", "Database serializer is already closed.")
	} else {
		buffer := this.buffer
		numBytesAvailable := buffer.Len()
		numBytesToRead := len(buf)
		err := error(nil)

		/*
		 * Generate JSON until enough data is available or error occurs.
		 */
		for (numBytesAvailable < numBytesToRead) && (err == nil) {
			err = this.generateJSON()
			numBytesAvailable = buffer.Len()
		}

		/*
		 * Check if error occured.
		 */
		if err != nil {
			errResult = err
		}

		bufferContent := buffer.String()
		bufferBytes := []byte(bufferContent)
		buffer.Reset()
		numBytesAvailable = len(bufferBytes)
		numBytesRead = numBytesToRead

		/*
		 * If there are fewer bytes available, then this is the limit.
		 */
		if numBytesAvailable < numBytesRead {
			numBytesRead = numBytesAvailable
		}

		bufferToCopy := bufferBytes[0:numBytesRead]
		copy(buf, bufferToCopy)

		/*
		 * If there are leftover bytes, we need to keep them.
		 */
		if numBytesAvailable > numBytesRead {
			bufferToKeep := bufferBytes[numBytesRead:numBytesAvailable]
			buffer.Write(bufferToKeep)
		}

		this.mutex.Unlock()
	}

	return numBytesRead, errResult
}

/*
 * Implements the Close function from io.ReadCloser.
 *
 * This will yield the read lock on the underlying database.
 */
func (this *databaseJsonSerializerStruct) Close() error {
	result := error(nil)
	this.mutex.Lock()
	db := this.db

	/*
	 * Check if serializer is already closed.
	 */
	if db == nil {
		result = fmt.Errorf("%s", "Database serializer is already closed.")
	} else {
		db.mutex.RUnlock()
		this.db = nil
	}

	this.mutex.Unlock()
	return result
}

/*
 * Returns the number of elements in the database sorted by this sorter.
 *
 * This method is required for implementation of sort.Interface.
 *
 * The underlying database is locked for write access during sorting.
 */
func (this *databaseSorterStruct) Len() int {
	result := int(0)
	db := this.db

	/*
	 * Make sure we reference a database.
	 */
	if db != nil {
		locationCount := db.locationCount
		locationCount64 := uint64(locationCount)

		/*
		 * Prevent overflow.
		 */
		if locationCount64 > math.MaxInt {
			panic("Database is too large to be sorted on this architecture.")
		} else {
			result = int(locationCount)
		}

	}

	return result
}

/*
 * Decide whether the element with index i must sort before the element with
 * the index j.
 *
 * This method is required for implementation of sort.Interface.
 *
 * The underlying database is locked for write access during sorting.
 */
func (this *databaseSorterStruct) Less(i int, j int) bool {
	result := false
	db := this.db

	/*
	 * Make sure we reference a database.
	 */
	if db != nil {
		locationCount32 := db.locationCount
		locationCount64 := uint64(locationCount32)
		locationCount := int(locationCount32)

		/*
		 * Prevent overflow.
		 */
		if locationCount64 > math.MaxInt {
			locationCount = math.MaxInt
		}

		/*
		 * Prevent out-of-bounds access.
		 */
		if ((i < 0) || (i >= locationCount)) || ((j < 0) || (j >= locationCount)) {
			msg := fmt.Sprintf("Index pair (%d, %d) out of bounds. (There are %d elements.)", i, j, locationCount)
			panic(msg)
		} else {
			i64 := int64(i)
			j64 := int64(j)
			offsetI := SIZE_DATABASE_HEADER + (SIZE_DATABASE_ENTRY * i64)
			offsetJ := SIZE_DATABASE_HEADER + (SIZE_DATABASE_ENTRY * j64)
			arrTimestampI := [SIZE_TIMESTAMP]byte{}
			arrTimestampJ := [SIZE_TIMESTAMP]byte{}
			bufTimestampI := arrTimestampI[:]
			bufTimestampJ := arrTimestampJ[:]
			fd := db.fd
			numBytesI, errI := fd.ReadAt(bufTimestampI, offsetI)
			numBytesJ, errJ := fd.ReadAt(bufTimestampJ, offsetJ)

			/*
			 * Make sure that we read both values.
			 */
			if ((errI == nil) && (numBytesI == SIZE_TIMESTAMP)) && ((errJ == nil) && (numBytesJ == SIZE_TIMESTAMP)) {
				eq := true

				/*
				 * Iterate over the timestamp.
				 */
				for idx := range arrTimestampI {

					/*
					 * Keep comparing as long as bytes are equal.
					 */
					if eq {
						curI := arrTimestampI[idx]
						curJ := arrTimestampJ[idx]

						/*
						 * Check if there is a mismatch.
						 */
						if curI != curJ {
							eq = false
							result = curI < curJ
						}

					}

				}

			}

		}

	}

	return result
}

/*
 * Swap the elements with indices i and j.
 *
 * This method is required for implementation of sort.Interface.
 *
 * The underlying database is locked for write access during sorting.
 */
func (this *databaseSorterStruct) Swap(i int, j int) {
	db := this.db

	/*
	 * Make sure we reference a database.
	 */
	if db != nil {
		locationCount32 := db.locationCount
		locationCount64 := uint64(locationCount32)
		locationCount := int(locationCount32)

		/*
		 * Prevent overflow.
		 */
		if locationCount64 > math.MaxInt {
			locationCount = math.MaxInt
		}

		/*
		 * Prevent out-of-bounds access.
		 */
		if ((i >= 0) && (i < locationCount)) && ((j >= 0) && (j < locationCount)) {
			i64 := int64(i)
			j64 := int64(j)
			offsetI := SIZE_DATABASE_HEADER + (SIZE_DATABASE_ENTRY * i64)
			offsetJ := SIZE_DATABASE_HEADER + (SIZE_DATABASE_ENTRY * j64)
			arrEntryI := [SIZE_DATABASE_ENTRY]byte{}
			arrEntryJ := [SIZE_DATABASE_ENTRY]byte{}
			bufEntryI := arrEntryI[:]
			bufEntryJ := arrEntryJ[:]
			fd := db.fd
			numBytesI, errI := fd.ReadAt(bufEntryI, offsetI)
			numBytesJ, errJ := fd.ReadAt(bufEntryJ, offsetJ)

			/*
			 * Make sure that we read both values.
			 */
			if ((errI == nil) && (numBytesI == SIZE_DATABASE_ENTRY)) && ((errJ == nil) && (numBytesJ == SIZE_DATABASE_ENTRY)) {
				bufEntryI, bufEntryJ = bufEntryJ, bufEntryI
				numBytesI, errI = fd.WriteAt(bufEntryI, offsetI)
				numBytesJ, errJ = fd.WriteAt(bufEntryJ, offsetJ)

				/*
				 * Make sure that we wrote both values.
				 */
				if ((errI != nil) || (numBytesI != SIZE_DATABASE_ENTRY)) || ((errJ != nil) || (numBytesJ != SIZE_DATABASE_ENTRY)) {
					msg := fmt.Sprintf("Error writing to offsets 0x%016x and 0x%016x! The geo database might have become corrupted.")
					panic(msg)
				}

			}

		}

	}

}

/*
 * Prepare storage for accessing geographic data, either by writing a new
 * header to an empty file or verifying the header of an already pre-filled
 * file.
 */
func prepareStorage(fd Storage) (int64, error) {
	fileSize := int64(0)
	errResult := error(nil)

	/*
	 * Verify that file descriptor is not nil.
	 */
	if fd == nil {
		errResult = fmt.Errorf("%s", "File descriptor must not be nil.")
	} else {
		posStored, err := fd.Seek(0, io.SeekCurrent)

		/*
		 * Check if file pointer was stored.
		 */
		if err != nil {
			reason := err.Error()
			errResult = fmt.Errorf("Failed to retrieve file pointer: %s", reason)
		} else {
			fileSize, err = fd.Seek(0, io.SeekEnd)

			/*
			 * Check if file size was obtained.
			 */
			if err != nil {
				reason := err.Error()
				errResult = fmt.Errorf("Failed to retrieve file size: %s", reason)
			} else {

				/*
				 * Check file size.
				 */
				if (fileSize != 0) && (fileSize < SIZE_DATABASE_HEADER) {
					errResult = fmt.Errorf("Illegal file size: Expected either zero or at least %d, but was %d.", SIZE_DATABASE_HEADER, fileSize)
				} else {
					posStart, err := fd.Seek(0, io.SeekStart)

					/*
					 * Check if seek to start was successful.
					 */
					if err != nil {
						reason := err.Error()
						errResult = fmt.Errorf("Failed to seek to beginning of file: %s", reason)
					} else if posStart != 0 {
						errResult = fmt.Errorf("Unexpected offset after seeking to beginning of file: Expected %d, found %d.", 0, posStart)
					} else {
						endianness := binary.BigEndian

						/*
						 * Either create or read file header.
						 */
						if fileSize == 0 {

							/*
							 * Create database header.
							 */
							hdr := databaseHeaderStruct{
								Magic:        MAGIC_NUMBER,
								VersionMajor: VERSION_MAJOR,
								VersionMinor: VERSION_MINOR,
							}

							buf := bytes.Buffer{}
							buf.Grow(SIZE_DATABASE_HEADER)
							err = binary.Write(&buf, endianness, &hdr)
							sizeWrittenBuf := buf.Len()

							/*
							 * Check if database header could be serialized.
							 */
							if err != nil {
								reason := err.Error()
								errResult = fmt.Errorf("Failed to serialize database header: %s", reason)
							} else if sizeWrittenBuf != SIZE_DATABASE_HEADER {
								errResult = fmt.Errorf("Unexpected size of database header: Expected %d, got %d.", SIZE_DATABASE_HEADER, sizeWrittenBuf)
							} else {
								content := buf.Next(sizeWrittenBuf)
								sizeWrittenFd, err := fd.WriteAt(content, 0)

								/*
								 * Check if buffer could be written to file.
								 */
								if err != nil {
									reason := err.Error()
									errResult = fmt.Errorf("Failed to write database header: %s", reason)
								} else if sizeWrittenFd != SIZE_DATABASE_HEADER {
									errResult = fmt.Errorf("Unexpected write size when writing database header: Expected %d, got %d.", SIZE_DATABASE_HEADER, sizeWrittenFd)
								}

							}

						} else {
							buf := make([]byte, SIZE_DATABASE_HEADER)
							sizeRead, err := fd.ReadAt(buf, 0)

							/*
							 * Check if read operation was successful.
							 */
							if err != nil {
								reason := err.Error()
								errResult = fmt.Errorf("Failed to read database header: %s", reason)
							} else if sizeRead != SIZE_DATABASE_HEADER {
								errResult = fmt.Errorf("Unexpected size of database header: Expected %d, got %d.", SIZE_DATABASE_HEADER, sizeRead)
							} else {
								rd := bytes.NewReader(buf)
								hdr := databaseHeaderStruct{}
								err := binary.Read(rd, endianness, &hdr)
								hdrMagic := hdr.Magic
								hdrVersionMajor := hdr.VersionMajor
								hdrVersionMinor := hdr.VersionMinor

								/*
								 * Check if header could be read and values are expected.
								 */
								if err != nil {
									reason := err.Error()
									errResult = fmt.Errorf("Failed to read database header: %s", reason)
								} else if hdrMagic != MAGIC_NUMBER {
									errResult = fmt.Errorf("File is not a geographical database. Expected magic number 0x%016x, but found 0x%016x.", MAGIC_NUMBER, hdrMagic)
								} else if (hdrVersionMajor != VERSION_MAJOR) || (hdrVersionMinor < VERSION_MINOR) {
									errResult = fmt.Errorf("File is in version %d.%d, but we expect %d.x (at least %d.%d).", hdrVersionMajor, hdrVersionMinor, VERSION_MAJOR, VERSION_MAJOR, VERSION_MINOR)
								}

							}

						}

					}

					posRestored, err := fd.Seek(posStored, io.SeekStart)

					/*
					 * Check if file pointer could be restored.
					 */
					if err != nil {
						reason := err.Error()
						errResult = fmt.Errorf("%s", "Failed to restore file pointer: %s", reason)
					} else if posRestored != posStored {
						errResult = fmt.Errorf("%s", "Failed to restore file pointer: Tried to restore to %d, but ended up at %d.", posStored, posRestored)
					}

				}

			}

		}

	}

	return fileSize, errResult
}

/*
 * Creates a new database for storing geographic data, backed by Storage, which
 * will usually be a file descriptor available for reading and writing.
 */
func Create(fd Storage) (Database, error) {
	result := (*databaseStruct)(nil)
	errResult := error(nil)
	fileSize, err := prepareStorage(fd)

	/*
	 * Check if storage was prepared.
	 */
	if err != nil {
		reason := err.Error()
		errResult = fmt.Errorf("Failed to prepare storage: %s", reason)
	} else {
		fileSize64 := uint64(fileSize)
		locationCount := uint32(0)

		/*
		 * Calculate location count.
		 */
		if fileSize64 >= SIZE_DATABASE_HEADER {
			locationCount64 := (fileSize64 - SIZE_DATABASE_HEADER) / SIZE_DATABASE_ENTRY

			/*
			 * Limit to 32 bit field.
			 */
			if locationCount64 > math.MaxUint32 {
				locationCount = math.MaxUint32
			} else {
				locationCount = uint32(locationCount64)
			}

		}

		/*
		 * Create database accessor.
		 */
		result = &databaseStruct{
			fd:            fd,
			locationCount: locationCount,
		}

	}

	return result, errResult
}
