package tileserver

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/andrepxx/location-visualizer/tile"
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
 * A remote tile server serving OpenStreetMaps data.
 */
type OSMTileServer interface {
	Get(z uint8, x uint32, y uint32) (tile.Image, error)
}

/*
 * Provides a no-op Close method for an io.ReadSeeker and io.ReaderAt.
 */
type readSeekerReaderAtWithNopCloserStruct struct {
	io.ReadSeeker
	io.ReaderAt
}

/*
 * Provides a close method that does nothing.
 */
func (this *readSeekerReaderAtWithNopCloserStruct) Close() error {
	return nil
}

/*
 * Data structure representing the remote tile server.
 */
type osmTileServerStruct struct {
	mutex sync.Mutex
	uri   string
}

/*
 * Build a tile path from a template, zoom level, x and y coordinate.
 */
func (this *osmTileServerStruct) tilePath(template string, zoom uint8, x uint32, y uint32) string {
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
 * Obtain a tile from an OpenStreetMaps tile server.
 */
func (this *osmTileServerStruct) getTile(id tile.Id) *bytes.Reader {
	x := id.X()
	y := id.Y()
	z := id.Z()
	maxId := uint32((1 << z) - 1)

	/*
	 * Check if tile ID is valid.
	 */
	if x > maxId || y > maxId {
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
		return r
	} else {
		templateUri := this.uri
		content := []byte{}

		/*
		 * Only download from OpenStreetMaps server if URI is not empty.
		 */
		if templateUri != "" {
			pathUri := this.tilePath(templateUri, z, x, y)
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
					buf, err := io.ReadAll(body)

					/*
					 * Check if image was loaded.
					 */
					if err == nil {
						content = buf
					}

					body.Close()
				}

				this.mutex.Unlock()
			}

		}

		r := bytes.NewReader(content)
		return r
	}

}

/*
 * Fetch a map tile from an OpenStreetMaps tile server.
 */
func (this *osmTileServerStruct) Get(z uint8, x uint32, y uint32) (tile.Image, error) {

	/*
	 * Check if zoom level is in range.
	 */
	if z > MAX_ZOOM_LEVEL {
		err := fmt.Errorf("Zoom level %d not allowed. (Maximum: %d)", z, MAX_ZOOM_LEVEL)
		return nil, err
	} else {
		tilesPerAxis := uint32(1) << z
		maxTileId := tilesPerAxis - 1

		/*
		 * Check if tile IDs are in range.
		 */
		if (x > maxTileId) || (y > maxTileId) {
			msg := "Cannot fetch tile (%d, %d). Maximum tile ID is (%d, %d) at zoom level %d."
			err := fmt.Errorf(msg, x, y, maxTileId, maxTileId, z)
			return nil, err
		} else {
			tileId := tile.CreateId(z, x, y)
			t := this.getTile(tileId)

			/*
			 * Provide "close" method.
			 */
			result := &readSeekerReaderAtWithNopCloserStruct{
				t,
				t,
			}

			return result, nil
		}

	}

}

/*
 * Creates a connection to a remote tile server serving OpenStreetMaps data.
 */
func CreateOSMTileServer(uri string) OSMTileServer {

	/*
	 * Create remote OpenStreetMaps tile server.
	 */
	src := osmTileServerStruct{
		uri: uri,
	}

	return &src
}
