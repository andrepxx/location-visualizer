package tile

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	ALL            = -1
	BASE_DECIMAL   = 10
	MAX_ZOOM_LEVEL = 19
	TEMPLATE_X     = "${x}"
	TEMPLATE_Y     = "${y}"
	TEMPLATE_ZOOM  = "${z}"
	TILE_SIZE      = 256
)

/*
 * A tile ID.
 */
type TileId interface {
	X() uint32
	Y() uint32
	Zoom() uint8
}

/*
 * A map tile.
 */
type Tile interface {
	Data() io.ReadSeeker
	Id() TileId
}

/*
 * A source for map tiles.
 */
type Source interface {
	Get(zoom uint8, x uint32, y uint32) (Tile, error)
	Prefetch(level uint8)
}

/*
 * A tile source for OpenStreetMaps.
 */
type osmSourceStruct struct {
	cachePath string
	mutex     sync.RWMutex
	uri       string
}

/*
 * A tile ID for an OpenStreetMaps tile.
 */
type osmTileIdStruct struct {
	zoom uint8
	x    uint32
	y    uint32
}

/*
 * A tile from an OpenStreetMaps server.
 */
type osmTileStruct struct {
	data   *bytes.Reader
	tileId osmTileIdStruct
}

/*
 * Returns the X coordinate of this map tile.
 */
func (this *osmTileIdStruct) X() uint32 {
	x := this.x
	return x
}

/*
 * Returns the Y coordinate of this map tile.
 */
func (this *osmTileIdStruct) Y() uint32 {
	y := this.y
	return y
}

/*
 * Returns the zoom level of this map tile.
 */
func (this *osmTileIdStruct) Zoom() uint8 {
	zoom := this.zoom
	return zoom
}

/*
 * Returns the image data from this map tile.
 */
func (this *osmTileStruct) Data() io.ReadSeeker {
	data := this.data
	return data
}

/*
 * Returns the ID of this map tile.
 */
func (this *osmTileStruct) Id() TileId {
	id := this.tileId
	return &id
}

/*
 * Builds a tile path from a template, zoom level, x and y coordinate.
 */
func (this *osmSourceStruct) tilePath(template string, zoom uint8, x uint32, y uint32) string {
	zoom64 := uint64(zoom)
	zoomString := strconv.FormatUint(zoom64, BASE_DECIMAL)
	x64 := uint64(x)
	xString := strconv.FormatUint(x64, BASE_DECIMAL)
	y64 := uint64(y)
	yString := strconv.FormatUint(y64, BASE_DECIMAL)
	template = strings.Replace(template, TEMPLATE_ZOOM, zoomString, ALL)
	template = strings.Replace(template, TEMPLATE_X, xString, ALL)
	template = strings.Replace(template, TEMPLATE_Y, yString, ALL)
	return template
}

/*
 * Obtains an OSM tile from cache or from the OSM server.
 */
func (this *osmSourceStruct) getTile(id osmTileIdStruct) *osmTileStruct {
	zoom := id.zoom
	x := id.x
	y := id.y
	maxId := uint32((1 << zoom) - 1)
	templateFile := this.cachePath

	/*
	 * Check if tile ID and file path template are valid.
	 */
	if templateFile == "" || x > maxId || y > maxId {
		buf := &bytes.Buffer{}
		rect := image.Rect(0, 0, TILE_SIZE, TILE_SIZE)
		img := image.NewNRGBA(rect)

		/*
		 * Create a PNG encoder.
		 */
		encoder := png.Encoder{
			CompressionLevel: png.BestCompression,
		}

		encoder.Encode(buf, img)
		content := buf.Bytes()
		r := bytes.NewReader(content)

		/*
		 * Create OSM tile.
		 */
		t := osmTileStruct{
			data:   r,
			tileId: id,
		}

		return &t
	} else {
		readFromFile := false
		pathFile := this.tilePath(templateFile, zoom, x, y)
		this.mutex.RLock()
		fd, err := os.Open(pathFile)
		content := []byte{}

		/*
		 * Check if file exists.
		 */
		if err == nil {
			buf, err := io.ReadAll(fd)

			/*
			 * Check if image was loaded from file.
			 */
			if err == nil {
				content = buf
				readFromFile = true
			}

			fd.Close()
		}

		this.mutex.RUnlock()

		/*
		 * If tile was not found in cache, download it from tile server and
		 * store it in cache.
		 */
		if !readFromFile {
			templateUri := this.uri

			/*
			 * Only download from OSM server if URI is not empty.
			 */
			if templateUri != "" {
				pathUri := this.tilePath(templateUri, zoom, x, y)
				fmt.Printf("Fetching from URI: %s\n", pathUri)
				client := &http.Client{}
				req, err := http.NewRequest("GET", pathUri, nil)

				/*
				 * Check if we have a valid request.
				 */
				if err == nil {
					this.mutex.Lock()
					req.Header.Set("User-Agent", "location-visualizer")
					resp, err := client.Do(req)

					/*
					 * Check if we got a response and store it in cache.
					 */
					if err == nil {
						body := resp.Body
						fd, err := os.Create(pathFile)

						/*
						 * Check if file was created.
						 */
						if err == nil {
							r := io.TeeReader(body, fd)
							buf, err := io.ReadAll(r)

							/*
							 * Check if image was loaded from file.
							 */
							if err == nil {
								content = buf
								readFromFile = true
							}

							fd.Close()
						}

						body.Close()
					}

					this.mutex.Unlock()
				}

			}

		}

		r := bytes.NewReader(content)

		/*
		 * Create OSM tile.
		 */
		t := osmTileStruct{
			data:   r,
			tileId: id,
		}

		return &t
	}

}

/*
 * Fetches a map tile from OSM or from the cache.
 */
func (this *osmSourceStruct) Get(zoom uint8, x uint32, y uint32) (Tile, error) {

	/*
	 * Check if zoom level is in range.
	 */
	if zoom > MAX_ZOOM_LEVEL {
		err := fmt.Errorf("Zoom level %d not allowed. (Maximum: %d)", zoom, MAX_ZOOM_LEVEL)
		return nil, err
	} else {
		tilesPerAxis := uint32(1) << zoom
		maxTileId := tilesPerAxis - 1

		/*
		 * Check if tile IDs are in range.
		 */
		if (x > maxTileId) || (y > maxTileId) {
			msg := "Cannot fetch tile (%d, %d). Maximum tile ID is (%d, %d) at zoom level %d."
			err := fmt.Errorf(msg, x, y, maxTileId, maxTileId, zoom)
			return nil, err
		} else {

			/*
			 * Create OSM tile id.
			 */
			tileId := osmTileIdStruct{
				zoom: zoom,
				x:    x,
				y:    y,
			}

			t := this.getTile(tileId)
			return t, nil
		}

	}

}

/*
 * Pre-fetch data from OSM to fill the caches.
 */
func (this *osmSourceStruct) Prefetch(zoomLevel uint8) {

	/*
	 * Limit zoom level to allowed maximum.
	 */
	if zoomLevel > MAX_ZOOM_LEVEL {
		zoomLevel = MAX_ZOOM_LEVEL
	}

	/*
	 * Fetch tiles for every zoom level.
	 */
	for zoom := uint8(0); zoom <= zoomLevel; zoom++ {
		tilesPerAxis := uint32(1) << zoom

		/*
		 * Fetch every row of tiles.
		 */
		for y := uint32(0); y < tilesPerAxis; y++ {

			/*
			 * Fetch every tile in the row.
			 */
			for x := uint32(0); x < tilesPerAxis; x++ {

				/*
				 * Create OSM tile id.
				 */
				id := osmTileIdStruct{
					zoom: zoom,
					x:    x,
					y:    y,
				}

				this.getTile(id)
			}

		}

	}

}

/*
 * Creates a tile source based on OpenStreetMaps data.
 */
func CreateOSMSource(uri string, cachePath string) Source {

	/*
	 * Create OSM tile source.
	 */
	src := osmSourceStruct{
		cachePath: cachePath,
		uri:       uri,
	}

	return &src
}
