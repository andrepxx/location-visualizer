package tile

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
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
	Id() TileId
	Image() *image.NRGBA
}

/*
 * A source for map tiles.
 */
type Source interface {
	Get(zoom uint8, x uint32, y uint32, colorScale float64) (Tile, error)
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
	imageData *image.NRGBA
	tileId    osmTileIdStruct
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
 * Returns the ID of this map tile.
 */
func (this *osmTileStruct) Id() TileId {
	id := this.tileId
	return &id
}

/*
 * Returns the image data from this map tile.
 */
func (this *osmTileStruct) Image() *image.NRGBA {
	img := this.imageData
	return img
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
		rect := image.Rect(0, 0, TILE_SIZE, TILE_SIZE)
		img := image.NewNRGBA(rect)

		/*
		 * Create OSM tile.
		 */
		t := osmTileStruct{
			imageData: img,
			tileId:    id,
		}

		return &t
	} else {
		readFromFile := false
		pathFile := this.tilePath(templateFile, zoom, x, y)
		this.mutex.RLock()
		fd, err := os.Open(pathFile)
		rect := image.ZR
		img := image.NewNRGBA(rect)

		/*
		 * Check if file exists.
		 */
		if err == nil {
			imgPng, err := png.Decode(fd)

			/*
			 * Check if image was decoded from file.
			 */
			if err == nil {
				rect = imgPng.Bounds()
				img = image.NewNRGBA(rect)
				rectMin := rect.Min
				minX := rectMin.X
				minY := rectMin.Y
				rectMax := rect.Max
				maxX := rectMax.X
				maxY := rectMax.Y

				/*
				 * Read image line by line.
				 */
				for y := minY; y < maxY; y++ {

					/*
					 * Read line pixel by pixel.
					 */
					for x := minX; x < maxX; x++ {
						c := imgPng.At(x, y)
						img.Set(x, y, c)
					}

				}

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
							imgPng, err := png.Decode(r)

							/*
							 * Check if image was decoded from response.
							 */
							if err == nil {
								rect = imgPng.Bounds()
								img = image.NewNRGBA(rect)
								rectMin := rect.Min
								minX := rectMin.X
								minY := rectMin.Y
								rectMax := rect.Max
								maxX := rectMax.X
								maxY := rectMax.Y

								/*
								 * Read image line by line.
								 */
								for y := minY; y < maxY; y++ {

									/*
									 * Read line pixel by pixel.
									 */
									for x := minX; x < maxX; x++ {
										c := imgPng.At(x, y)
										img.Set(x, y, c)
									}

								}

							}

							fd.Close()
						}

						body.Close()
					}

					this.mutex.Unlock()
				}

			}

		}

		/*
		 * Create OSM tile.
		 */
		t := osmTileStruct{
			imageData: img,
			tileId:    id,
		}

		return &t
	}

}

/*
 * Perform color transformation on OSM data.
 */
func (this *osmSourceStruct) transformColor(in color.NRGBA, scale float64) color.NRGBA {
	r := in.R
	g := in.G
	b := in.B
	rFloat := float64(r) / 255.0
	gFloat := float64(g) / 255.0
	bFloat := float64(b) / 255.0
	rInvFloat := 1.0 - rFloat
	gInvFloat := 1.0 - gFloat
	bInvFloat := 1.0 - bFloat
	lumaFloat := (0.22 * rInvFloat) + (0.72 * gInvFloat) + (0.06 * bInvFloat)
	lumaFloatScaled := scale * lumaFloat
	lumaFloatByte := math.Round(lumaFloatScaled * 255.0)
	lumaByte := uint8(lumaFloatByte)

	/*
	 * Create resulting color value.
	 */
	c := color.NRGBA{
		R: lumaByte,
		G: lumaByte,
		B: lumaByte,
		A: 255,
	}

	return c
}

/*
 * Fetches a map tile from OSM or from the cache.
 */
func (this *osmSourceStruct) Get(zoom uint8, x uint32, y uint32, colorScale float64) (Tile, error) {

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
			 * Make sure that color scale is in range.
			 */
			if colorScale < 0.0 {
				colorScale = 0.0
			} else if colorScale > 1.0 {
				colorScale = 1.0
			}

			/*
			 * Create OSM tile id.
			 */
			tileId := osmTileIdStruct{
				zoom: zoom,
				x:    x,
				y:    y,
			}

			tileSource := this.getTile(tileId)
			imgSource := tileSource.imageData
			rect := imgSource.Bounds()
			imgTarget := image.NewNRGBA(rect)
			rectMin := rect.Min
			minX := rectMin.X
			minY := rectMin.Y
			rectMax := rect.Max
			maxX := rectMax.X
			maxY := rectMax.Y

			/*
			 * Read image line by line.
			 */
			for y := minY; y < maxY; y++ {

				/*
				 * Read line pixel by pixel.
				 */
				for x := minX; x < maxX; x++ {
					sourceColor := imgSource.NRGBAAt(x, y)
					targetColor := this.transformColor(sourceColor, colorScale)
					imgTarget.Set(x, y, targetColor)
				}

			}

			/*
			 * Create OSM tile.
			 */
			tileTarget := osmTileStruct{
				imageData: imgTarget,
				tileId:    tileId,
			}

			return &tileTarget, nil
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
