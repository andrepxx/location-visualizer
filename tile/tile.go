package tile

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
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
 * A source for map tiles.
 */
type Source interface {
	Get(xres uint32, yres uint32, minX float64, maxX float64, minY float64, maxY float64) (*image.NRGBA, error)
}

/*
 * A tile source for OpenStreetMaps.
 */
type osmSourceStruct struct {
	cachePath string
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
			} else {
				msg := err.Error()
				fmt.Printf("[DEBUG] Tile source: Error decoding PNG file '%s': %s\n", pathFile, msg)
			}

			fd.Close()
		} else {
			msg := err.Error()
			fmt.Printf("[DEBUG] Tile source: Error reading from file '%s': %s\n", pathFile, msg)
		}

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
				fmt.Printf("[DEBUG] Tile source: Fetching from URI: %s\n", pathUri)
				client := &http.Client{}
				req, err := http.NewRequest("GET", pathUri, nil)

				/*
				 * Check if we have a valid request.
				 */
				if err == nil {
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
 * Fetches map data from OpenStreetMaps and renders it into an image.
 */
func (this *osmSourceStruct) Get(xres uint32, yres uint32, minX float64, maxX float64, minY float64, maxY float64) (*image.NRGBA, error) {
	tileSizeFloat := float64(TILE_SIZE)
	xresFloat := float64(xres)
	numTilesX := xresFloat / tileSizeFloat
	dx := math.Abs(maxX - minX)
	dxPerTile := dx / numTilesX
	zoomFloat := -math.Log2(dxPerTile)

	/*
	 * Zoom cannot be negative.
	 */
	if zoomFloat < 0.0 {
		zoomFloat = 0.0
	}

	zoom := uint8(zoomFloat)

	/*
	 * Limit to maximum OSM zoom level.
	 */
	if zoom > MAX_ZOOM_LEVEL {
		zoom = MAX_ZOOM_LEVEL
	}

	zoomFloat = float64(zoom)
	dxPerTile = math.Pow(2.0, -zoomFloat)
	idxMinXFloat := (minX + 0.5) / dxPerTile
	idxMinX := int32(idxMinXFloat)
	idxMaxXFloat := (maxX + 0.5) / dxPerTile
	idxMaxX := int32(idxMaxXFloat)
	idxMinYFloat := (0.5 - maxY) / dxPerTile
	idxMinY := int32(idxMinYFloat)
	idxMaxYFloat := (0.5 - minY) / dxPerTile
	idxMaxY := int32(idxMaxYFloat)
	tiles := []*osmTileStruct{}

	/*
	 * Iterate over the X axis.
	 */
	for idxX := idxMinX; idxX <= idxMaxX; idxX++ {

		/*
		 * Iterate over the Y axis.
		 */
		for idxY := idxMinY; idxY <= idxMaxY; idxY++ {
			idxXX := uint32(idxX)
			idxYY := uint32(idxY)

			/*
			 * Create OSM tile ID.
			 */
			id := osmTileIdStruct{
				zoom: zoom,
				x:    idxXX,
				y:    idxYY,
			}

			tile := this.getTile(id)
			tiles = append(tiles, tile)
		}

	}

	// TODO: Interpolate tiles and put them into an image.

	return nil, fmt.Errorf("%s", "Not yet implemented.")
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
