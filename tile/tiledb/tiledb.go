package tiledb

import (
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/andrepxx/location-visualizer/tile"
)

const (
	MAGIC_IMAGEDB      = 0x496d616765444204
	MAGIC_INDEXDB      = 0x496e646578444204
	SIZE_BUFFER        = 8192
	SIZE_HASH          = 64
	SIZE_INDEXDB_ENTRY = 81
	SIZE_LENGTH_FIELD  = 4
	SIZE_MAGIC         = 8
)

/*
 * A handle to an image stored in an image database.
 *
 * An image handle is similar to a file name. However, it is computed from the
 * contents of the image.
 */
type ImageHandle [SIZE_HASH]byte

/*
 * A database storing images and allowing lookup by image handles.
 */
type ImageDatabase interface {
	Cleanup(keep func(ImageHandle) bool) error
	Close() error
	Insert(buf []byte) (ImageHandle, error)
	Open(handle ImageHandle) (tile.Image, error)
}

/*
 * A database mapping OSM tile IDs to image handles.
 */
type IndexDatabase interface {
	Close() error
	Entry(idx uint64) (tile.Id, TileMetadata, error)
	Insert(id tile.Id, metadata TileMetadata) error
	Length() (uint64, error)
	Search(id tile.Id) (uint64, bool)
}

/*
 * Interface that a storage backing a database will have to implement.
 *
 * Typically, this storage will be a file.
 */
type Storage interface {
	ReadAt(buf []byte, offset int64) (int, error)
	Seek(offset int64, whence int) (int64, error)
	Truncate(size int64) error
	WriteAt(buf []byte, offset int64) (int, error)
}

/*
 * Data structure representing an Image stored in an ImageDatabase.
 */
type imageStruct struct {
	mutex sync.RWMutex
	db    *imageDatabaseStruct
	r     *io.SectionReader
}

/*
 * Closes this Image, releasing a read lock on the ImageDatabase.
 *
 * Any file operation on a closed Image will yield an error. This includes
 * closing an already closed Image.
 *
 * Implements the Close method from io.Closer.
 */
func (this *imageStruct) Close() error {
	errResult := error(nil)
	this.mutex.Lock()
	db := this.db

	/*
	 * Check if image is still open.
	 */
	if db == nil {
		errResult = fmt.Errorf("%s", "Image already closed.")
	} else {
		this.r = nil
		db.mutex.RUnlock()
		this.db = nil
	}

	this.mutex.Unlock()
	return errResult
}

/*
 * Implements the Read method from io.Reader.
 */
func (this *imageStruct) Read(buf []byte) (int, error) {
	bytesRead := int(0)
	errResult := error(nil)
	this.mutex.Lock()
	r := this.r

	/*
	 * Check if image is still open.
	 */
	if r == nil {
		errResult = fmt.Errorf("%s", "Image already closed.")
	} else {
		n, err := r.Read(buf)

		/*
		 * Check if read was successful and pass through EOF.
		 */
		if err == io.EOF {
			errResult = io.EOF
		} else if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Error reading from database: %s", msg)
		} else {
			bytesRead = n
		}

	}

	this.mutex.Unlock()
	return bytesRead, errResult
}

/*
 * Implements the ReadAt method from io.ReaderAt.
 */
func (this *imageStruct) ReadAt(buf []byte, offset int64) (int, error) {
	bytesRead := int(0)
	errResult := error(nil)
	this.mutex.RLock()
	r := this.r

	/*
	 * Check if image is still open.
	 */
	if r == nil {
		errResult = fmt.Errorf("%s", "Image already closed.")
	} else {
		n, err := r.ReadAt(buf, offset)

		/*
		 * Check if read was successful.
		 */
		if err != nil {
			msg := err.Error()
			offset64 := uint64(offset)
			errResult = fmt.Errorf("Error reading from database at offset 0x%016x: %s", offset64, msg)
		} else {
			bytesRead = n
		}

	}

	this.mutex.RUnlock()
	return bytesRead, errResult
}

/*
 * Implements the Seek method from io.Seeker.
 */
func (this *imageStruct) Seek(offset int64, whence int) (int64, error) {
	offsetNew := int64(0)
	errResult := error(nil)
	this.mutex.RLock()
	r := this.r

	/*
	 * Check if image is still open.
	 */
	if r == nil {
		errResult = fmt.Errorf("%s", "Image already closed.")
	} else {
		n, err := r.Seek(offset, whence)

		/*
		 * Check if read was successful.
		 */
		if err != nil {
			msg := err.Error()
			offset64 := uint64(offset)
			errResult = fmt.Errorf("Error seeking to offset 0x%016x (whence: %d): %s", offset64, whence, msg)
		} else {
			offsetNew = n
		}

	}

	this.mutex.RUnlock()
	return offsetNew, errResult
}

/*
 * Data structure representing an ImageDatabase.
 *
 * The index points to the offset in the file where the image size is stored.
 */
type imageDatabaseStruct struct {
	mutex sync.RWMutex
	fd    Storage
	index map[ImageHandle]uint64
	size  uint64
}

/*
 * Initialize image database by either writing header to file descriptor (if
 * file is empty) or filling the index by walking the file.
 */
func (this *imageDatabaseStruct) initialize() error {
	errResult := error(nil)
	fd := this.fd

	/*
	 * Verify that file descriptor is not nil.
	 */
	if fd == nil {
		errResult = fmt.Errorf("%s", "File descriptor must not be nil.")
	} else {
		size, errSeekEnd := fd.Seek(0, io.SeekEnd)
		offset, errSeekStart := fd.Seek(0, io.SeekStart)

		/*
		 * Check if determining file size was successful.
		 */
		if (size < 0) || (errSeekEnd != nil) {
			errResult = fmt.Errorf("%s", "Failed to seek to end of file.")
		} else if (offset != 0) || (errSeekStart != nil) {
			errResult = fmt.Errorf("%s", "Failed to seek to beginning of file.")
		} else {
			this.size = uint64(size)

			/*
			 * If file is empty, write header. If file is non-empty but too small, fail.
			 * Otherwise, index file.
			 */
			if size == 0 {
				endian := binary.BigEndian
				w := io.NewOffsetWriter(fd, 0)
				data := uint64(MAGIC_IMAGEDB)
				err := binary.Write(w, endian, data)

				/*
				 * Check if magic number was written to file.
				 */
				if err != nil {
					errResult = fmt.Errorf("%s", "Failed to write magic number to file.")
				}

				size, err = fd.Seek(0, io.SeekEnd)

				/*
				 * Check if we could retrieve size of file.
				 */
				if err != nil {
					errResult = fmt.Errorf("%s", "Failed to obtain size of file.")
				}

				this.size = uint64(size)
			} else if size < SIZE_MAGIC {
				errResult = fmt.Errorf("File too small: Should have at least %d bytes.", SIZE_MAGIC)
			} else {
				endian := binary.BigEndian
				r := io.NewSectionReader(fd, 0, size)
				magic := uint64(0)
				err := binary.Read(r, endian, &magic)

				/*
				 * Verify magic number was read correctly.
				 */
				if err != nil {
					errResult = fmt.Errorf("%s", "Failed to read magic number from file.")
				} else if magic != MAGIC_IMAGEDB {
					errResult = fmt.Errorf("Failed to read magic number from file: Expected 0x%016x, found 0x%016x.", MAGIC_IMAGEDB, magic)
				} else {
					offset += SIZE_MAGIC
					buf := make([]byte, SIZE_BUFFER)
					h := sha512.New()
					bufSum := [SIZE_HASH]byte{}
					index := this.index

					/*
					 * Build index until reaching end of file or an error occurs.
					 */
					for (offset < size) && (errResult == nil) {
						actualOffset, err := r.Seek(offset, io.SeekStart)

						/*
						 * Check if seeking to length field was sucessful.
						 */
						if err != nil {
							errResult = fmt.Errorf("Failed to seek to offset %d (0x%016x).", offset, offset)
						} else if actualOffset != offset {
							errResult = fmt.Errorf("Tried to seek to offset %d (0x%016x), but arrived at %d (0x%016x).", offset, offset, actualOffset, actualOffset)
						} else {
							sizeSection := uint32(0)
							err = binary.Read(r, endian, &sizeSection)

							/*
							 * Check if reading length field was successful.
							 */
							if err != nil {
								errResult = fmt.Errorf("Error reading length field at offset %d (0x%016x).", offset, offset)
							} else {
								offsetSectionStart := uint64(offset)
								offset += SIZE_LENGTH_FIELD
								sizeSectionSigned := int64(sizeSection)
								section := io.LimitReader(r, sizeSectionSigned)
								h.Reset()
								n, err := io.CopyBuffer(h, section, buf)
								offset += n

								/*
								 * Check if section got added to hash.
								 */
								if err != nil {
									errResult = fmt.Errorf("Read error at offset %d (0x%016x) inside section of size %d (0x%08x) starting at offset %d (0x%016x).", offset, offset, sizeSection, sizeSection, offsetSectionStart, offsetSectionStart)
								} else if n != sizeSectionSigned {
									errResult = fmt.Errorf("Read incorrect amount of bytes from section at offset %d (0x%016x). Expected %d (0x%016x), got %d (0x%016x).", offset, offset, sizeSectionSigned, sizeSectionSigned, n, n)
								} else {
									sectionHash := bufSum[:0]
									// h.Sum can write in-place or allocate a new buffer.
									sectionHash = h.Sum(sectionHash)
									m := copy(bufSum[:], sectionHash)

									/*
									 * If resulting hash is smaller than buffer,
									 * zero the rest of the buffer.
									 */
									if m < SIZE_HASH {
										bufToZero := bufSum[m:SIZE_HASH]

										/*
										 * Zero remaining part of buffer.
										 */
										for i := range bufToZero {
											bufToZero[i] = 0
										}

									}

									handle := ImageHandle(bufSum)
									index[handle] = offsetSectionStart
								}

							}

						}

					}

				}

			}

		}

	}

	return errResult
}

/*
 * Cleans up the database by iterating over its entries and presenting their
 * handles to a function, which then decides whether this entry shall be kept
 * (true) or discarded (false).
 *
 * Cleaning up a closed database is an error.
 *
 * I/O errors occuring during cleanup are also reported and might leave the
 * database in an inconsistent / corrupted state.
 */
func (this *imageDatabaseStruct) Cleanup(keep func(ImageHandle) bool) error {
	offsetRead := int64(SIZE_MAGIC)
	offsetWrite := offsetRead
	errResult := error(nil)
	endian := binary.BigEndian
	this.mutex.Lock()
	fd := this.fd
	sizeDatabase := this.size
	sizeDatabaseSigned := int64(sizeDatabase)
	r := io.NewSectionReader(fd, 0, sizeDatabaseSigned)
	buf := make([]byte, SIZE_BUFFER)
	h := sha512.New()
	bufSum := [SIZE_HASH]byte{}

	/*
	 * Read images until reaching the end of the database or an error occurs.
	 */
	for (offsetRead < sizeDatabaseSigned) && (errResult == nil) {
		currentOffset, err := r.Seek(offsetRead, io.SeekStart)

		/*
		 * Check if seeking was successful.
		 */
		if err != nil {
			errResult = fmt.Errorf("Failed to seek to offset %d (0x%016x).", offsetRead, offsetRead)
		} else if currentOffset != offsetRead {
			errResult = fmt.Errorf("Failed to seek to offset %d (0x%016x). Arrived at offset %d (0x%016x) instead.", offsetRead, offsetRead, currentOffset, currentOffset)
		} else {
			sizeSection := uint32(0)
			err = binary.Read(r, endian, &sizeSection)

			/*
			 * Check if reading length field was successful.
			 */
			if err != nil {
				errResult = fmt.Errorf("Error reading length field at offset %d (0x%016x).", offsetRead, offsetRead)
			} else {
				sizeSectionSigned := int64(sizeSection)
				offsetRead += SIZE_LENGTH_FIELD
				offsetSectionStart := int64(offsetRead)
				section := io.NewSectionReader(fd, offsetSectionStart, sizeSectionSigned)
				h.Reset()
				n, err := io.CopyBuffer(h, section, buf)

				/*
				 * Check if section got added to hash.
				 */
				if err != nil {
					errResult = fmt.Errorf("Read error at offset %d (0x%016x) inside section of size %d (0x%08x) starting at offset %d (0x%016x).", offsetRead, offsetRead, sizeSection, sizeSection, offsetSectionStart, offsetSectionStart)
				} else if n != sizeSectionSigned {
					errResult = fmt.Errorf("Read incorrect amount of bytes from section at offset %d (0x%016x). Expected %d (0x%016x), got %d (0x%016x).", offsetRead, offsetRead, sizeSectionSigned, sizeSectionSigned, n, n)
				} else {
					sectionHash := bufSum[:0]
					// h.Sum can write in-place or allocate a new buffer.
					sectionHash = h.Sum(sectionHash)
					m := copy(bufSum[:], sectionHash)

					/*
					 * If resulting hash is smaller than buffer,
					 * zero the rest of the buffer.
					 */
					if m < SIZE_HASH {
						bufToZero := bufSum[m:SIZE_HASH]

						/*
						 * Zero remaining part of buffer.
						 */
						for i := range bufToZero {
							bufToZero[i] = 0
						}

					}

					handle := ImageHandle(bufSum)
					keepImage := keep(handle)

					/*
					 * Check if we shall keep the image.
					 */
					if keepImage {

						/*
						 * If offsets match, we can just skip over the image
						 * instead of moving it, since we would just write it
						 * back to its original location anyhow.
						 */
						if currentOffset == offsetWrite {
							offsetWrite += SIZE_LENGTH_FIELD
							offsetWrite += sizeSectionSigned
						} else {
							lengthField := io.NewSectionReader(fd, currentOffset, SIZE_LENGTH_FIELD)
							w := io.NewOffsetWriter(fd, offsetWrite)
							n, err := io.CopyBuffer(w, lengthField, buf)

							/*
							* Check if length field was copied successfully.
							 */
							if err != nil {
								errResult = fmt.Errorf("Failed to copy length field from offset %d (0x%016x) to offset %d (0x%08x).", offsetSectionStart, offsetSectionStart, offsetWrite, offsetWrite)
							} else if n != SIZE_LENGTH_FIELD {
								errResult = fmt.Errorf("Failed to copy length field from offset %d (0x%016x) to offset %d (0x%08x). Copied %d (0x%08x) bytes.", offsetSectionStart, offsetSectionStart, offsetWrite, offsetWrite, n, n)
							} else {
								offsetWrite += SIZE_LENGTH_FIELD
								offsetInSection, err := section.Seek(0, io.SeekStart)

								/*
								* Check if an error occured while seeking back
								* to beginning of the section.
								 */
								if err != nil {
									errResult = fmt.Errorf("Failed to seek back to start of section.")
								} else if offsetInSection != 0 {
									errResult = fmt.Errorf("Failed to seek back to start of section. Arrived at offset %d (0x%016x) in section.", offsetInSection, offsetInSection)
								} else {
									w := io.NewOffsetWriter(fd, offsetWrite)
									n, err := io.CopyBuffer(w, section, buf)

									/*
									* Check if section was copied successfully.
									 */
									if err != nil {
										errResult = fmt.Errorf("Failed to copy section of size %d (0x%016x) from offset %d (0x%016x) to offset %d (0x%08x).", sizeSection, sizeSection, offsetSectionStart, offsetSectionStart, offsetWrite, offsetWrite)
									} else if n != sizeSectionSigned {
										errResult = fmt.Errorf("Failed to copy section of size %d (0x%016x) from offset %d (0x%016x) to offset %d (0x%08x). Copied %d (0x%08x) bytes.", sizeSection, sizeSection, offsetSectionStart, offsetSectionStart, offsetWrite, offsetWrite, n, n)
									}

									offsetWrite += n
								}

							}

						}

					}

				}

				offsetRead += sizeSectionSigned
			}

		}

	}

	/*
	 * If no error occured so far, finally truncate file.
	 */
	if errResult == nil {
		err := fd.Truncate(offsetWrite)

		/*
		* Check if file got truncated correctly.
		 */
		if err != nil {
			errResult = fmt.Errorf("Failed to truncate image database to size %d (0x%016x).", offsetWrite, offsetWrite)
		}

	}

	this.mutex.Unlock()
	return errResult
}

/*
 * Closes the image database, releasing the associated file descriptor.
 *
 * Closing an image database, which has already been closed, is an error.
 */
func (this *imageDatabaseStruct) Close() error {
	errResult := error(nil)
	this.mutex.Lock()
	fd := this.fd

	/*
	 * If database is already closed, return error, otherwise close file descriptor.
	 */
	if fd == nil {
		errResult = fmt.Errorf("%s", "Image database is already closed.")
	} else {
		this.fd = nil
	}

	this.mutex.Unlock()
	return errResult
}

/*
 * Inserts an image into the database, yielding a handle and, potentially, an
 * error.
 *
 * Inserting an image into the database, which already exists, is not an error,
 * but a no-op.
 *
 * (The new image will not be inserted and looking up the handle will yield the
 * existing image.)
 */
func (this *imageDatabaseStruct) Insert(buf []byte) (ImageHandle, error) {
	hash := sha512.Sum512(buf)
	handle := ImageHandle(hash)
	errResult := error(nil)
	this.mutex.Lock()
	fd := this.fd

	/*
	 * If database is already closed, return error, otherwise write length
	 * information and image data to file descriptor.
	 */
	if fd == nil {
		errResult = fmt.Errorf("%s", "Image database is already closed.")
	} else {
		index := this.index
		_, present := index[handle]

		/*
		 * If image is not already present in the database, it has to be
		 * inserted.
		 */
		if !present {
			offsetLengthField := this.size
			offsetLengthFieldSigned := int64(offsetLengthField)

			/*
			 * Check if offset is still in range.
			 */
			if offsetLengthField > math.MaxInt64 {
				errResult = fmt.Errorf("Offset too large: 0x%016x (Maximum allowed is 0x%016x.)", offsetLengthField, math.MaxInt64)
			} else {
				lengthFieldWriter := io.NewOffsetWriter(fd, offsetLengthFieldSigned)
				dataSize := len(buf)
				dataSize64 := uint64(dataSize)

				/*
				 * Check if data size is still in range.
				 */
				if dataSize64 > math.MaxUint32 {
					errResult = fmt.Errorf("Data size too large: 0x%016x (Maximum allowed is 0x%08x.)", dataSize64, math.MaxUint32)
				} else {
					dataSize32 := uint32(dataSize64)
					endian := binary.BigEndian
					err := binary.Write(lengthFieldWriter, endian, dataSize32)

					/*
					 * Check if length field could be written.
					 */
					if err != nil {
						errResult = fmt.Errorf("Failed to write length field at offset %d (0x%016x).", offsetLengthField, offsetLengthField)
					} else {
						offsetData := offsetLengthField + SIZE_LENGTH_FIELD

						/*
						 * Check if offset is still in range.
						 */
						if offsetData > math.MaxInt64 {
							errResult = fmt.Errorf("Offset too large: 0x%016x (Maximum allowed is 0x%016x.)", offsetData, math.MaxInt64)
						} else {
							offsetDataSigned := int64(offsetData)
							dataWriter := io.NewOffsetWriter(fd, offsetDataSigned)
							bytesWritten, err := dataWriter.Write(buf)

							/*
							 * Check if data was written.
							 */
							if err != nil {
								errResult = fmt.Errorf("Failed to insert image at offset %d (0x%016x).", offsetData, offsetData)
							} else if bytesWritten != dataSize {
								errResult = fmt.Errorf("Failed to insert image at offset %d (0x%016x). Expected %d (0x%016x) bytes written, but was %d (0x%016x).", offsetData, offsetData, dataSize, dataSize, bytesWritten, bytesWritten)
							} else {
								this.index[handle] = offsetLengthField
								this.size = offsetData + dataSize64
							}

						}

					}

				}

			}

			/*
			 * If an error occured, truncate file to original size.
			 */
			if errResult != nil {
				err := fd.Truncate(offsetLengthFieldSigned)

				/*
				 * Check if truncation was successful.
				 */
				if err != nil {
					panic("Failed to truncate image database to original size after incomplete insertion. Database is corrupted!")
				}

				this.size = offsetLengthField
			}

		}

	}

	this.mutex.Unlock()
	return handle, errResult
}

/*
 * Looks up an ImageHandle in the database and returns an Image to it.
 *
 * The Image works similar to a file descriptor, while the ImageHandle works
 * similar to a file name.
 *
 * Looking up the same ImageHandle in the database will yield separate Image
 * objects, each with their own state (opened / closed, file offset, etc.),
 * providing access to the same underlying image data.
 *
 * As long as an Image is open, the database will also be locked for reading,
 * meaning that no concurrent write operations can take place. A pending write
 * operation or a write operation currently taking place, in turn, will prevent
 * more images from being opened.
 *
 * Closing an Image will yield its particular read lock on the database.
 */
func (this *imageDatabaseStruct) Open(handle ImageHandle) (tile.Image, error) {
	result, errResult := tile.Image(nil), error(nil)
	this.mutex.RLock()
	fd := this.fd

	/*
	 * Check if file is open.
	 */
	if fd == nil {
		errResult = fmt.Errorf("%s", "Image database is not open.")
	} else {
		index := this.index
		offsetLength, ok := index[handle]

		/*
		 * Check if image could be looked up.
		 */
		if !ok {
			errResult = fmt.Errorf("%s", "Not found.")
		} else if offsetLength > math.MaxInt64 {
			errResult = fmt.Errorf("Offset 0x%016x too large. (Maximum is 0x%016x.)", offsetLength, math.MaxInt64)
		} else {
			offsetLengthSigned := int64(offsetLength)
			lengthFieldReader := io.NewSectionReader(fd, offsetLengthSigned, SIZE_LENGTH_FIELD)
			endian := binary.BigEndian
			lengthImage := uint32(0)
			err := binary.Read(lengthFieldReader, endian, &lengthImage)

			/*
			 * Check if length field could be read.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Failed to read from offset 0x%016x: %s", offsetLength, msg)
			} else {
				offsetImage := offsetLength + SIZE_LENGTH_FIELD

				/*
				 * Check if offset is in valid range.
				 */
				if offsetImage > math.MaxInt64 {
					errResult = fmt.Errorf("Offset 0x%016x too large. (Maximum is 0x%016x.)", offsetImage, math.MaxInt64)
				} else {
					offsetImageSigned := int64(offsetImage)
					lengthImageSigned := int64(lengthImage)
					imageReader := io.NewSectionReader(fd, offsetImageSigned, lengthImageSigned)

					/*
					 * Create result image.
					 */
					result = &imageStruct{
						db: this,
						r:  imageReader,
					}

				}

			}

		}

	}

	/*
	 * If we do NOT return a reader, the database has to be unlocked.
	 *
	 * Otherwise, it has to stay locked for reading.
	 */
	if result == nil {
		this.mutex.RUnlock()
	}

	return result, errResult
}

/*
 * Creates an image database backed by Storage.
 */
func CreateImageDatabase(fd Storage) (ImageDatabase, error) {
	idx := make(map[ImageHandle]uint64)

	/*
	 * Create image database..
	 */
	db := &imageDatabaseStruct{
		fd:    fd,
		index: idx,
	}

	err := db.initialize()

	/*
	 * If an error occured during initialization destroy database.
	 */
	if err != nil {
		db = nil
	}

	return db, err
}

/*
 * Data structure representing an entry in IndexDatabase.
 *
 * It maps a tile ID with zoom level, as well as x, y coordinates, to a SHA-512
 * hash of an image and a timestamp in milliseconds since the Epoch.
 *
 * The timestamp shall represent the instant in time when the entry was created
 * or last updated.
 */
type indexDbEntry struct {
	Z           uint8
	X           uint32
	Y           uint32
	TimestampMs int64
	Hash        [SIZE_HASH]byte
}

/*
 * Data structure representing an IndexDatabase.
 */
type indexDatabaseStruct struct {
	mutex sync.RWMutex
	fd    Storage
	index map[tile.Id]uint64
}

/*
 * Calculates the offset of an entry in the index database, given an index.
 */
func (this *indexDatabaseStruct) calculateOffset(idx uint64) int64 {
	const MAX_IDX = (math.MaxInt64 - SIZE_MAGIC) / SIZE_INDEXDB_ENTRY
	offset := int64(-1)

	/*
	 * Check if index is in valid range.
	 */
	if idx <= MAX_IDX {
		offset = int64(SIZE_MAGIC + (idx * SIZE_INDEXDB_ENTRY))
	}

	return offset
}

/*
 * Read entry from storage.
 *
 * This function assumes that the database it locked for either reading or writing.
 */
func (this *indexDatabaseStruct) readEntry(fd Storage, idx uint64, entry *indexDbEntry) error {
	result := error(nil)
	offset := this.calculateOffset(idx)

	/*
	 * Check if offset is correct.
	 */
	if offset < 0 {
		result = fmt.Errorf("%s", "Invalid offset")
	} else {
		r := io.NewSectionReader(fd, offset, SIZE_INDEXDB_ENTRY)
		endian := binary.BigEndian
		err := binary.Read(r, endian, entry)

		/*
		 * Check if entry could be read.
		 */
		if err != nil {
			msg := err.Error()
			result = fmt.Errorf("Failed to read from offset 0x%016x: %s", offset, msg)
		}

	}

	return result
}

/*
 * Write entry to storage.
 *
 * This function assumes that the database is locked for writing.
 */
func (this *indexDatabaseStruct) writeEntry(fd Storage, idx uint64, entry *indexDbEntry) error {
	result := error(nil)
	offset := this.calculateOffset(idx)

	/*
	 * Check if offset is correct.
	 */
	if offset < 0 {
		result = fmt.Errorf("%s", "Invalid offset.")
	} else {
		r := io.NewOffsetWriter(fd, offset)
		endian := binary.BigEndian
		err := binary.Write(r, endian, entry)

		/*
		 * Check if entry could be written.
		 */
		if err != nil {
			msg := err.Error()
			result = fmt.Errorf("Failed to write to offset 0x%016x: %s", offset, msg)
		}

	}

	return result
}

/*
 * Returns the number of entries currently stored in storage.
 *
 * This function assumes that the database it locked for either reading or writing.
 */
func (this *indexDatabaseStruct) numEntries(fd Storage) (uint64, error) {
	offsetSaved, err := fd.Seek(0, io.SeekCurrent)

	/*
	 * Check if we could get the current file offset.
	 */
	if err != nil {
		return 0, fmt.Errorf("%s", "Failed to store current file offset.")
	} else {
		fileSize, err := fd.Seek(0, io.SeekEnd)

		/*
		 * Check if we could seek to the end of the file.
		 */
		if err != nil {
			return 0, fmt.Errorf("%s", "Failed to seek to end of file.")
		} else if fileSize < 0 {
			return 0, fmt.Errorf("%s", "File size is negative.")
		} else {
			offsetRestored, err := fd.Seek(offsetSaved, io.SeekStart)

			/*
			 * Check if we could restore the file offset.
			 */
			if err != nil {
				return 0, fmt.Errorf("%s", "Failed to restore file offset.")
			} else if offsetRestored != offsetSaved {
				return 0, fmt.Errorf("%s", "Restored offset does not match saved offset.")
			} else if fileSize < SIZE_MAGIC {
				return 0, fmt.Errorf("%s", "File too small.")
			} else {
				fileSize64 := uint64(fileSize)
				dataSize := fileSize64 - SIZE_MAGIC

				/*
				 * Check if data area size is a multiple of entry size.
				 */
				if (dataSize % SIZE_INDEXDB_ENTRY) != 0 {
					return 0, fmt.Errorf("%s", "Size of data area is not a multiple of entry size.")
				} else {
					result := dataSize / SIZE_INDEXDB_ENTRY
					return result, nil
				}

			}

		}

	}

}

/*
 * Closes the index database, releasing the associated file descriptor.
 *
 * Closing an index database, which has already been closed, is an error.
 */
func (this *indexDatabaseStruct) Close() error {
	errResult := error(nil)
	this.mutex.Lock()
	fd := this.fd

	/*
	 * If database is already closed, return error, otherwise close file descriptor.
	 */
	if fd == nil {
		errResult = fmt.Errorf("%s", "Index database is already closed.")
	} else {
		this.fd = nil
	}

	this.mutex.Unlock()
	return errResult
}

/*
 * Retrieves an entry from the index database by index.
 */
func (this *indexDatabaseStruct) Entry(idx uint64) (tile.Id, TileMetadata, error) {
	tileId := tile.Id{}
	tileMetadata := TileMetadata{}
	errResult := error(nil)
	this.mutex.RLock()
	fd := this.fd
	numEntries, err := this.numEntries(fd)

	/*
	 * Check if number of entries could be retrieved or index is out of range.
	 */
	if err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Failed to retrieve number of entries from index database: %s", msg)
	} else if idx >= numEntries {
		errResult = fmt.Errorf("Index out of range: %d (database has %d entries)", idx, numEntries)
	} else {
		entry := indexDbEntry{}
		err := this.readEntry(fd, idx, &entry)

		/*
		 * Check if error occured reading entry.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Error occured while reading entry %d from index database: %s", idx, msg)
		} else {
			x := entry.X
			y := entry.Y
			z := entry.Z
			tileId = tile.CreateId(z, x, y)
			timestamp := entry.TimestampMs
			h := entry.Hash
			img := ImageHandle(h)

			/*
			 * Create tile metadata.
			 */
			tileMetadata = TileMetadata{
				handle:      img,
				timestampMs: timestamp,
			}

		}

	}

	this.mutex.RUnlock()
	return tileId, tileMetadata, errResult
}

/*
 * Inserts an entry, mapping a TileId to TileMetadata, into the database.
 *
 * Inserting an entry for a TileId which already exists overwrites the existing
 * entry.
 */
func (this *indexDatabaseStruct) Insert(id tile.Id, metadata TileMetadata) error {
	x := id.X()
	y := id.Y()
	z := id.Z()
	timestamp := metadata.timestampMs
	handle := metadata.handle
	hash := [64]byte(handle)

	/*
	 * Create entry for index database.
	 */
	entry := indexDbEntry{
		Z:           z,
		X:           x,
		Y:           y,
		TimestampMs: timestamp,
		Hash:        hash,
	}

	this.mutex.Lock()
	fd := this.fd
	index := this.index
	idx, found := index[id]
	numEntries := uint64(0)
	errNumEntries := error(nil)
	errResult := error(nil)

	/*
	 * If not found, append entry to the end.
	 */
	if !found {
		numEntries, errNumEntries = this.numEntries(fd)

		/*
		 * Check if error occured retrieving number of entries.
		 */
		if errNumEntries != nil {
			msg := errNumEntries.Error()
			errResult = fmt.Errorf("Failed to retrieve number of entries from index database: %s", msg)
		} else {
			idx = numEntries
			index[id] = idx
		}

	}

	/*
	 * If we did not encounter en error, write entry to storage.
	 */
	if errNumEntries == nil {
		err := this.writeEntry(fd, idx, &entry)

		/*
		 * Check if error occured writing entry.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to write entry %d to index database: %s", idx, msg)
		}

	}

	this.mutex.Unlock()
	return errResult
}

/*
 * Returns the number of entries in this index database.
 */
func (this *indexDatabaseStruct) Length() (uint64, error) {
	this.mutex.RLock()
	fd := this.fd
	numEntries, err := this.numEntries(fd)
	this.mutex.RUnlock()
	numEntries64 := uint64(numEntries)
	return numEntries64, err
}

/*
 * Looks up an entry in the index database by TileId.
 *
 * Returns the index of the entry and a boolean value indicating whether it was
 * found in the database.
 */
func (this *indexDatabaseStruct) Search(id tile.Id) (uint64, bool) {
	idx := uint64(0)
	found := false
	this.mutex.RLock()
	index := this.index
	idx, found = index[id]
	this.mutex.RUnlock()
	return idx, found
}

/*
 * Initialize index database by either writing header to file descriptor (if
 * file is empty) or filling entries and index by walking the file.
 */
func (this *indexDatabaseStruct) initialize() error {
	errResult := error(nil)
	fd := this.fd

	/*
	 * Verify that file descriptor is not nil.
	 */
	if fd == nil {
		errResult = fmt.Errorf("%s", "File descriptor must not be nil.")
	} else {
		size, errSeekEnd := fd.Seek(0, io.SeekEnd)
		offset, errSeekStart := fd.Seek(0, io.SeekStart)

		/*
		 * Check if determining file size was successful.
		 */
		if (size < 0) || (errSeekEnd != nil) {
			errResult = fmt.Errorf("%s", "Failed to seek to end of file.")
		} else if (offset != 0) || (errSeekStart != nil) {
			errResult = fmt.Errorf("%s", "Failed to seek to beginning of file.")
		} else {

			/*
			 * If file is empty, write header. If file is non-empty but too small, fail.
			 * Otherwise, index file.
			 */
			if size == 0 {
				endian := binary.BigEndian
				w := io.NewOffsetWriter(fd, 0)
				data := uint64(MAGIC_INDEXDB)
				err := binary.Write(w, endian, data)

				/*
				 * Check if magic number was written to file.
				 */
				if err != nil {
					errResult = fmt.Errorf("%s", "Failed to write magic number to file.")
				}

			} else if size < SIZE_MAGIC {
				errResult = fmt.Errorf("File too small: Should have at least %d bytes.", SIZE_MAGIC)
			} else {
				endian := binary.BigEndian
				r := io.NewSectionReader(fd, 0, size)
				magic := uint64(0)
				err := binary.Read(r, endian, &magic)

				/*
				 * Verify magic number was read correctly.
				 */
				if err != nil {
					errResult = fmt.Errorf("%s", "Failed to read magic number from file.")
				} else if magic != MAGIC_INDEXDB {
					errResult = fmt.Errorf("Failed to read magic number from file: Expected 0x%016x, found 0x%016x.", MAGIC_INDEXDB, magic)
				} else {
					offset += SIZE_MAGIC
					index := this.index
					entry := indexDbEntry{}
					numEntriesRead := uint64(0)

					/*
					 * Build index until reaching end of file or an error occurs.
					 */
					for (offset < size) && (errResult == nil) {
						actualOffset, err := r.Seek(offset, io.SeekStart)

						/*
						 * Check if seeking to length field was sucessful.
						 */
						if err != nil {
							errResult = fmt.Errorf("Failed to seek to offset %d (0x%016x).", offset, offset)
						} else if actualOffset != offset {
							errResult = fmt.Errorf("Tried to seek to offset %d (0x%016x), but arrived at %d (0x%016x).", offset, offset, actualOffset, actualOffset)
						} else {
							err := binary.Read(r, endian, &entry)

							/*
							 * Check if entry could be read from database.
							 */
							if err != nil {
								msg := err.Error()
								errResult = fmt.Errorf("Failed to read entry %d from offset %d (0x%016x): %s", numEntriesRead, offset, offset, msg)
							} else {
								entryZ := entry.Z
								entryX := entry.X
								entryY := entry.Y
								tileId := tile.CreateId(entryZ, entryX, entryY)
								index[tileId] = numEntriesRead
								numEntriesRead++
								offset += SIZE_INDEXDB_ENTRY
							}

						}

					}

					this.index = index
				}

			}

		}

	}

	return errResult
}

/*
 * Creates an index database backed by Storage.
 */
func CreateIndexDatabase(fd Storage) (IndexDatabase, error) {
	idx := make(map[tile.Id]uint64)

	/*
	 * Create index database.
	 */
	db := &indexDatabaseStruct{
		fd:    fd,
		index: idx,
	}

	err := db.initialize()

	/*
	 * If an error occured during initialization destroy database.
	 */
	if err != nil {
		db = nil
	}

	return db, err
}

/*
 * Data structure representing metadata of a tile.
 */
type TileMetadata struct {
	handle      ImageHandle
	timestampMs int64
}

/*
 * Returns the ImageHandle associated with this tile for lookup in
 * ImageDatabase.
 */
func (this *TileMetadata) Handle() ImageHandle {
	result := this.handle
	return result
}

/*
 * Returns the timestamp in milliseconds since the Epoch associated with this
 * entry in the IndexDatabase.
 *
 * This will usually be the time of the last update to this entry.
 */
func (this *TileMetadata) TimestampMs() int64 {
	result := this.timestampMs
	return result
}

/*
 * Create tile metadata structure.
 *
 * The timestamp is expected in milliseconds since the Epoch and will represent
 * the time when the entry was created (or last updated).
 */
func CreateTileMetadata(timestampMs int64, handle ImageHandle) TileMetadata {

	/*
	 * Create tile metadata.
	 */
	m := TileMetadata{
		handle:      handle,
		timestampMs: timestampMs,
	}

	return m
}
