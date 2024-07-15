package tileutil

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/andrepxx/location-visualizer/tile"
	"github.com/andrepxx/location-visualizer/tile/tiledb"
	"github.com/andrepxx/location-visualizer/tile/tileserver"
)

const (
	REX_OSM_TILE_NAME = "^osm-(\\d*)-(\\d*)-(\\d*)\\.png$"
	MAX_TILE_SIZE     = 1048576
	MAX_ZOOM_LEVEL    = 19
	MODE_DIR          = 0755
	MODE_FILE         = 0644
	SIZE_BUFFER       = 8096
)

/*
 * Utility for accessing a tile database.
 */
type TileUtil interface {
	Cleanup() error
	Export(w io.Writer, creationTime time.Time) error
	Fetch(server tileserver.OSMTileServer, id tile.Id) (tile.Image, error)
	Import(r io.Reader) error
	Prefetch(server tileserver.OSMTileServer, maxZoom uint8)
}

/*
 * Data structure representing the utility.
 */
type tileUtilStruct struct {
	mutex         sync.RWMutex
	imageDatabase tiledb.ImageDatabase
	indexDatabase tiledb.IndexDatabase
}

/*
 * Remove all images from ImageDatabase that are no longer referenced from IndexDatabase.
 */
func (this *tileUtilStruct) Cleanup() error {
	errResult := error(nil)
	this.mutex.Lock()
	imgdb := this.imageDatabase
	idxdb := this.indexDatabase
	numEntries, err := idxdb.Length()

	/*
	 * Check if we could get the number of entries from the index database.
	 */
	if err != nil {
		msg := err.Error()
		return fmt.Errorf("Failed to get number of entries from index database: %s", msg)
	} else {
		allHandles := make(map[tiledb.ImageHandle]bool)

		/*
		 * Iterate over all entries in index database and collect all handles.
		 */
		for idx := uint64(0); (idx < numEntries) && (errResult == nil); idx++ {
			_, metadata, err := idxdb.Entry(idx)

			/*
			 * Check if error occured retrieving entry from index database.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Failed to get entry %d from index database: %s", idx, msg)
			} else {
				handle := metadata.Handle()
				allHandles[handle] = true
			}

		}

		/*
		 * If no error occured so far, continue to cleanup image database.
		 */
		if errResult == nil {

			/*
			 * The cleanup condition.
			 */
			condition := func(handle tiledb.ImageHandle) bool {
				result := allHandles[handle]
				return result
			}

			err := imgdb.Cleanup(condition)

			/*
			 * Check if error occured during cleanup.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Error occured during image database cleanup: %s", msg)
			}

		}

	}

	this.mutex.Unlock()
	return errResult
}

/*
 * Export a single entry from index database into a tarball.
 */
func (this *tileUtilStruct) exportEntry(w *tar.Writer, idx uint64, tilesPath string, buf []byte) error {
	errResult := error(nil)
	idxdb := this.indexDatabase
	imgdb := this.imageDatabase
	tileId, tileMetadata, err := idxdb.Entry(idx)

	/*
	 * Check if entry could be read.
	 */
	if err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Failed to read entry from index database: %s", msg)
	} else {
		x := tileId.X()
		y := tileId.Y()
		z := tileId.Z()
		path := fmt.Sprintf("%sosm-%d-%d-%d.png", tilesPath, z, x, y)
		timestamp := tileMetadata.TimestampMs()
		modTime := time.UnixMilli(timestamp)
		modTimeUtc := modTime.UTC()
		handle := tileMetadata.Handle()
		img, err := imgdb.Open(handle)

		/*
		 * Check if image could be opened.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to retrieve image from image database: %s", msg)
		} else {
			size, err := img.Seek(0, io.SeekEnd)

			/*
			 * Check if seek to end of file was successful.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Failed to seek to end of file for image '%s': %s", path, msg)
			} else {
				_, err = img.Seek(0, io.SeekStart)

				/*
				 * Check if seek to beginning of file was successful.
				 */
				if err != nil {
					msg := err.Error()
					errResult = fmt.Errorf("Failed to seek to beginning of file for image '%s': %s", path, msg)
				} else {

					/*
					 * Create header for image file.
					 */
					hdr := tar.Header{
						Typeflag: tar.TypeReg,
						Name:     path,
						Mode:     MODE_FILE,
						Size:     size,
						ModTime:  modTimeUtc,
						Format:   tar.FormatPAX,
					}

					err = w.WriteHeader(&hdr)

					/*
					 * Check if we could write file header.
					 */
					if err != nil {
						msg := err.Error()
						errResult = fmt.Errorf("Failed to write header for image '%s': %s", path, msg)
					} else {
						_, err := io.CopyBuffer(w, img, buf)

						/*
						 * Check if error occured.
						 */
						if err != nil {
							msg := err.Error()
							errResult = fmt.Errorf("Failed to write image '%s' to archive: %s", path, msg)
						}

					}

				}

			}

		}

	}

	return errResult
}

/*
 * Export tiles from a tile database into a tarball.
 */
func (this *tileUtilStruct) Export(w io.Writer, creationTime time.Time) error {
	tilePath := "tile/"
	errResult := error(nil)
	gzw, err := gzip.NewWriterLevel(w, gzip.BestCompression)

	/*
	 * Check if gzipped file could be opened for writing.
	 */
	if err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Failed to open gzipped file for writing: %s", msg)
	} else {
		tw := tar.NewWriter(gzw)

		/*
		 * Create header for tile directory.
		 */
		hdrTileDir := tar.Header{
			Typeflag: tar.TypeDir,
			Name:     tilePath,
			Mode:     MODE_FILE,
			ModTime:  creationTime,
			Format:   tar.FormatPAX,
		}

		err := tw.WriteHeader(&hdrTileDir)

		/*
		 * Check if directory could be created.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to create directory: %s", msg)
		} else {
			this.mutex.RLock()
			idxdb := this.indexDatabase
			numEntries, err := idxdb.Length()

			/*
			 * Check if number of entries could be determined.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Failed to determine number of entries in index database: %s", msg)
			} else {
				buf := make([]byte, SIZE_BUFFER)
				iterateIdx := true

				/*
				 * Iterate over all entries in index database.
				 */
				for idx := uint64(0); iterateIdx && (idx < numEntries); idx++ {
					err := this.exportEntry(tw, idx, tilePath, buf)

					/*
					 * Check if an error occured exporting the current entry.
					 */
					if err != nil {
						msg := err.Error()
						errResult = fmt.Errorf("Error exporting entry number %d (of %d): %s", idx, numEntries, msg)
					}

				}

			}

			this.mutex.RUnlock()
		}

		err = tw.Close()

		/*
		 * Check if error occured and it's the first one.
		 */
		if (err != nil) && (errResult == nil) {
			msg := err.Error()
			errResult = fmt.Errorf("Error closing tar archive: %s", msg)
		}

	}

	err = gzw.Close()

	/*
	 * Check if error occured and it's the first one.
	 */
	if (err != nil) && (errResult == nil) {
		msg := err.Error()
		errResult = fmt.Errorf("Error closing gzip stream: %s", msg)
	}

	return errResult
}

/*
 * Fetch tile from cache.
 *
 * This assumes that the databases are locked for either reading or writing.
 */
func (this *tileUtilStruct) fetchFromCache(id tile.Id) (tile.Image, error) {
	result := tile.Image(nil)
	errResult := error(nil)
	idxdb := this.indexDatabase
	idx, found := idxdb.Search(id)
	x := id.X()
	y := id.Y()
	z := id.Z()

	/*
	 * Check if tile could be found in index database.
	 */
	if !found {
		errResult = fmt.Errorf("Tile (%d, %d, %d) not found.", x, y, z)
	} else {
		tid, metadata, err := idxdb.Entry(idx)
		xx := tid.X()
		yy := tid.Y()
		zz := tid.Z()

		/*
		 * Check if correct tile was found.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to read entry from index database: %s", msg)
		} else if xx != x || yy != y || zz != z {
			errResult = fmt.Errorf("Tile IDs don't match: Expected (%d, %d, %d), got (%d, %d, %d).", x, y, z, xx, yy, zz)
		} else {
			imgdb := this.imageDatabase
			handle := metadata.Handle()
			img, err := imgdb.Open(handle)

			/*
			 * Check if image could be opened.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Failed to open image: %s", msg)
			} else {
				result = img
			}

		}

	}

	return result, errResult
}

func (this *tileUtilStruct) fetchFromServer(server tileserver.OSMTileServer, id tile.Id) (tile.Image, error) {
	z := id.Z()
	x := id.X()
	y := id.Y()
	result, errResult := server.Get(z, x, y)

	/*
	 * If we could fetch tile from server, store it in cache.
	 */
	if errResult == nil {
		content, err := io.ReadAll(result)

		/*
		 * Check if tile content could be read.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to read tile content: %s", msg)
		} else {
			imgdb := this.imageDatabase
			handle, err := imgdb.Insert(content)

			/*
			 * Check if tile was inserted into image database.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Failed to insert tile into image database: %s", msg)
			} else {
				t := time.Now()
				timestamp := t.UnixMilli()
				metadata := tiledb.CreateTileMetadata(timestamp, handle)
				idxdb := this.indexDatabase
				err := idxdb.Insert(id, metadata)

				/*
				 * Check if tile was inserted into index database.
				 */
				if err != nil {
					msg := err.Error()
					errResult = fmt.Errorf("Failed to insert tile into index database: %s", msg)
				}

			}

		}

	}

	return result, errResult
}

/*
 * Lookup tile in cache or fetch it from server and store it in cache.
 */
func (this *tileUtilStruct) fetch(server tileserver.OSMTileServer, id tile.Id, forceUpdate bool) (tile.Image, error) {
	result := tile.Image(nil)
	errResult := error(nil)

	/*
	 * Check if we shall perform a forced update.
	 */
	if forceUpdate {
		result, errResult = this.fetchFromServer(server, id)
	} else {
		this.mutex.RLock()
		result, errResult = this.fetchFromCache(id)
		this.mutex.RUnlock()

		/*
		 * If tile could not be loaded from cache, fetch it from server.
		 */
		if errResult != nil {
			this.mutex.Lock()
			result, errResult = this.fetchFromCache(id)

			/*
			 * Verify that we still have a cache miss, since we re-acquired the lock.
			 */
			if errResult != nil {
				result, errResult = this.fetchFromServer(server, id)
			}

			this.mutex.Unlock()
		}

	}

	return result, errResult
}

/*
 * Lookup tile in cache or fetch it from server and store it in cache.
 */
func (this *tileUtilStruct) Fetch(server tileserver.OSMTileServer, id tile.Id) (tile.Image, error) {
	result, errResult := this.fetch(server, id, false)
	return result, errResult
}

/*
 * Import tiles from a tarball into a tile database.
 */
func (this *tileUtilStruct) Import(r io.Reader) error {
	errResult := error(nil)
	gzr, err := gzip.NewReader(r)

	/*
	 * Check if gzipped file could be opened for reading.
	 */
	if err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Failed to open gzipped file for reading: %s", msg)
	} else {
		this.mutex.Lock()
		idxdb := this.indexDatabase
		imgdb := this.imageDatabase
		rex, _ := regexp.Compile(REX_OSM_TILE_NAME)
		tr := tar.NewReader(gzr)
		hdr, errNext := tr.Next()

		/*
		 * Iterate over all files in the tarball.
		 */
		for (errResult == nil) && (errNext != io.EOF) {
			entryType := hdr.Typeflag

			/*
			 * Only handle regular files.
			 */
			if entryType == tar.TypeReg {
				filePath := hdr.Name
				dirName, fileName := path.Split(filePath)
				fileSize := hdr.Size

				/*
				 * Only import files in the "tile/" subdirectory that do not exceed
				 * a certain maximum file size.
				 */
				if (dirName == "tile/") && (fileSize <= MAX_TILE_SIZE) {
					groups := rex.FindStringSubmatch(fileName)
					numGroups := len(groups)

					/*
					 * Check that we found four groups.
					 */
					if numGroups == 4 {
						zStr := groups[1]
						z64, errZ := strconv.ParseUint(zStr, 10, 8)
						z := uint8(z64)
						xStr := groups[2]
						x64, errX := strconv.ParseUint(xStr, 10, 32)
						x := uint32(x64)
						yStr := groups[3]
						y64, errY := strconv.ParseUint(yStr, 10, 32)
						y := uint32(y64)

						/*
						 * Check that all coordinates could be parsed.
						 */
						if errZ == nil && errX == nil && errY == nil {
							id := tile.CreateId(z, x, y)
							content, err := io.ReadAll(tr)

							/*
							 * Check if file content could be read.
							 */
							if err != nil {
								msg := err.Error()
								errResult = fmt.Errorf("Failed to read contents of file '%s': %s", filePath, msg)
							} else {
								handle, err := imgdb.Insert(content)

								/*
								 * Check if image was stored in image database.
								 */
								if err != nil {
									msg := err.Error()
									errResult = fmt.Errorf("Failed to insert image '%s' into image database: %s", filePath, msg)
								} else {
									modTime := hdr.ModTime
									timestamp := modTime.UnixMilli()
									metadata := tiledb.CreateTileMetadata(timestamp, handle)
									err := idxdb.Insert(id, metadata)

									/*
									 * Check if image was stored in index database.
									 */
									if err != nil {
										msg := err.Error()
										errResult = fmt.Errorf("Failed to insert image '%s' into index database: %s", filePath, msg)
									}

								}

							}

						}

					}

				}

			}

			hdr, errNext = tr.Next()
		}

		this.mutex.Unlock()
	}

	return errResult
}

/*
 * Prefetch tiles from server up to a certain zoom level.
 */
func (this *tileUtilStruct) Prefetch(server tileserver.OSMTileServer, zoomLevel uint8) {

	/*
	 * Limit zoom level to allowed maximum.
	 */
	if zoomLevel > MAX_ZOOM_LEVEL {
		zoomLevel = MAX_ZOOM_LEVEL
	}

	/*
	 * Fetch tiles for every zoom level.
	 */
	for z := uint8(0); z <= zoomLevel; z++ {
		tilesPerAxis := uint32(1) << z

		/*
		 * Fetch every row of tiles.
		 */
		for y := uint32(0); y < tilesPerAxis; y++ {

			/*
			 * Fetch every tile in the row.
			 */
			for x := uint32(0); x < tilesPerAxis; x++ {
				id := tile.CreateId(z, x, y)
				this.fetch(server, id, false)
			}

		}

	}

}

/*
 * Create a new util for handling tiles.
 */
func CreateTileUtil(idxdb tiledb.IndexDatabase, imgdb tiledb.ImageDatabase) TileUtil {

	/*
	 * Create util.
	 */
	util := tileUtilStruct{
		imageDatabase: imgdb,
		indexDatabase: idxdb,
	}

	return &util
}
