package controller

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/andrepxx/location-visualizer/filter"
	"github.com/andrepxx/location-visualizer/geo"
	"github.com/andrepxx/location-visualizer/tile"
	"github.com/andrepxx/location-visualizer/webserver"
	"github.com/andrepxx/sydney/color"
	"github.com/andrepxx/sydney/coordinates"
	"github.com/andrepxx/sydney/scene"
	"image"
	imagecolor "image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"math"
	"os"
	"runtime"
	"strconv"
)

/*
 * Constants for the controller.
 */
const (
	CONFIG_PATH = "config/config.json"
)

/*
 * The configuration for the controller.
 */
type configStruct struct {
	GeoData   string
	MapServer string
	MapCache  string
	UseMap    bool
	WebServer webserver.Config
}

/*
 * The controller for the DSP.
 */
type controllerStruct struct {
	colorMapping color.Mapping
	config       configStruct
	data         []geo.Location
	tileSource   tile.Source
}

/*
 * The controller interface.
 */
type Controller interface {
	Operate()
	Prefetch(zoomLevel uint8)
}

/*
 * Renders a map tile.
 */
func (this *controllerStruct) getTileHandler(request webserver.HttpRequest) webserver.HttpResponse {
	xIn := request.Params["x"]
	x64, _ := strconv.ParseUint(xIn, 10, 32)
	x := uint32(x64)
	yIn := request.Params["y"]
	y64, _ := strconv.ParseUint(yIn, 10, 32)
	y := uint32(y64)
	zIn := request.Params["z"]
	z64, _ := strconv.ParseUint(zIn, 10, 8)
	z := uint8(z64)
	tileSource := this.tileSource
	t, err := tileSource.Get(z, x, y)

	/*
	 * Check if tile could be fetched.
	 */
	if err != nil {
		msg := err.Error()
		customMsg := fmt.Sprintf("Failed to fetch map tile: %s\n", msg)
		contentType := this.config.WebServer.ErrorMime

		/*
		 * Create HTTP response.
		 */
		response := webserver.HttpResponse{
			Header: map[string]string{"Content-type": contentType},
			Body:   bytes.NewBufferString(customMsg).Bytes(),
		}

		return response
	} else {
		img := t.Image()
		id := t.Id()
		idX := id.X()
		idY := id.Y()
		idZ := id.Zoom()

		/*
		 * Ensure that the tile IDs match.
		 */
		if (x != idX) || (y != idY) || (z != idZ) {
			msg := "Something is wrong here: (%d, %d, %d) != (%d, %d, %d)"
			customMsg := fmt.Sprintf(msg, idX, idY, idZ, x, y, z)
			contentType := this.config.WebServer.ErrorMime

			/*
			 * Create HTTP response.
			 */
			response := webserver.HttpResponse{
				Header: map[string]string{"Content-type": contentType},
				Body:   bytes.NewBufferString(customMsg).Bytes(),
			}

			return response
		} else {

			/*
			 * Create a PNG encoder.
			 */
			encoder := png.Encoder{
				CompressionLevel: png.BestCompression,
			}

			buf := &bytes.Buffer{}
			err := encoder.Encode(buf, img)

			/*
			 * Check if image could be encoded.
			 */
			if err != nil {
				msg := err.Error()
				customMsg := fmt.Sprintf("Failed to encode image: %s\n", msg)
				contentType := this.config.WebServer.ErrorMime

				/*
				 * Create HTTP response.
				 */
				response := webserver.HttpResponse{
					Header: map[string]string{"Content-type": contentType},
					Body:   bytes.NewBufferString(customMsg).Bytes(),
				}

				return response
			} else {
				bufBytes := buf.Bytes()

				/*
				 * Create HTTP response.
				 */
				response := webserver.HttpResponse{
					Header: map[string]string{"Content-type": "image/png"},
					Body:   bufBytes,
				}

				return response
			}

		}

	}

}

/*
 * Renders location data into an image.
 */
func (this *controllerStruct) renderHandler(request webserver.HttpRequest) webserver.HttpResponse {
	xresIn := request.Params["xres"]
	xres64, _ := strconv.ParseUint(xresIn, 10, 16)
	xres := uint32(xres64)
	yresIn := request.Params["yres"]
	yres64, _ := strconv.ParseUint(yresIn, 10, 16)
	yres := uint32(yres64)
	xposIn := request.Params["xpos"]
	xpos, _ := strconv.ParseFloat(xposIn, 64)
	yposIn := request.Params["ypos"]
	ypos, _ := strconv.ParseFloat(yposIn, 64)
	zoomIn := request.Params["zoom"]
	zoom, _ := strconv.ParseUint(zoomIn, 10, 8)
	useBGIn := request.Params["usebg"]
	useBG, _ := strconv.ParseBool(useBGIn)
	minTimeIn := request.Params["mintime"]
	minTime, _ := filter.ParseTime(minTimeIn)
	maxTimeIn := request.Params["maxtime"]
	maxTime, _ := filter.ParseTime(maxTimeIn)
	zoomFloat := float64(zoom)
	zoomExp := -0.2 * zoomFloat
	zoomFac := math.Pow(2.0, zoomExp)
	halfWidth := 0.5 * zoomFac
	xresFloat := float64(xres)
	yresFloat := float64(yres)
	aspectRatio := yresFloat / xresFloat
	halfHeight := aspectRatio * halfWidth
	minX := xpos - halfWidth
	maxX := xpos + halfWidth
	minY := ypos - halfHeight
	maxY := ypos + halfHeight
	scn := scene.Create(xres, yres, minX, maxX, minY, maxY)
	filteredData := this.data
	minTimeIsZero := minTime.IsZero()
	maxTimeIsZero := maxTime.IsZero()

	/*
	 * Apply filter if at least one of the limits is set.
	 */
	if !minTimeIsZero || !maxTimeIsZero {
		flt := filter.Time(minTime, maxTime)
		filteredData = filter.Apply(flt, filteredData)
	}

	numDataPoints := len(filteredData)
	projectedData := make([]coordinates.Cartesian, numDataPoints)

	/*
	 * Obtain projected data points.
	 */
	for i, elem := range filteredData {
		projectedData[i] = elem.Projected()
	}

	scn.Aggregate(projectedData)
	mapping := this.colorMapping
	xresInt := int(xres)
	yresInt := int(yres)
	dim := image.Rect(0, 0, xresInt, yresInt)
	target := image.NewNRGBA(dim)

	/*
	 * Check if we should render a map background.
	 */
	if useBG {

		/*
		 * The background color.
		 */
		c := imagecolor.NRGBA{
			R: 0,
			G: 0,
			B: 0,
			A: 255,
		}

		uniform := image.NewUniform(c)
		draw.Draw(target, dim, uniform, image.ZP, draw.Over)
		tileSource := this.tileSource

		/*
		 * Check if tile source was set.
		 */
		if tileSource != nil {
			imgMap, err := tileSource.Render(xres, yres, minX, maxX, minY, maxY)

			/*
			 * Draw map tile if it was successfully rendered.
			 */
			if err == nil {
				draw.Draw(target, dim, imgMap, image.ZP, draw.Over)
			}

		}

	}

	imgData, err := scn.Render(mapping)

	/*
	 * Draw data if it was successfully retrieved.
	 */
	if err == nil {
		draw.Draw(target, dim, imgData, image.ZP, draw.Over)
	}

	/*
	 * Check if image could be rendered.
	 */
	if err != nil {
		msg := err.Error()
		customMsg := fmt.Sprintf("Failed to render image: %s", msg)
		contentType := this.config.WebServer.ErrorMime

		/*
		 * Create HTTP response.
		 */
		response := webserver.HttpResponse{
			Header: map[string]string{"Content-type": contentType},
			Body:   bytes.NewBufferString(customMsg).Bytes(),
		}

		return response
	} else {

		/*
		 * Create a PNG encoder.
		 */
		encoder := png.Encoder{
			CompressionLevel: png.BestCompression,
		}

		buf := &bytes.Buffer{}
		err := encoder.Encode(buf, target)

		/*
		 * Check if image could be encoded.
		 */
		if err != nil {
			msg := err.Error()
			customMsg := fmt.Sprintf("Failed to encode image: %s\n", msg)
			contentType := this.config.WebServer.ErrorMime

			/*
			 * Create HTTP response.
			 */
			response := webserver.HttpResponse{
				Header: map[string]string{"Content-type": contentType},
				Body:   bytes.NewBufferString(customMsg).Bytes(),
			}

			return response
		} else {
			bufBytes := buf.Bytes()

			/*
			 * Create HTTP response.
			 */
			response := webserver.HttpResponse{
				Header: map[string]string{"Content-type": "image/png"},
				Body:   bufBytes,
			}

			return response
		}

	}

}

/*
 * Handles CGI requests that could not be dispatched to other CGIs.
 */
func (this *controllerStruct) errorHandler(request webserver.HttpRequest) webserver.HttpResponse {

	/*
	 * Create HTTP response.
	 */
	response := webserver.HttpResponse{
		Header: map[string]string{"Content-type": this.config.WebServer.ErrorMime},
		Body:   bytes.NewBufferString("This CGI call is not implemented.").Bytes(),
	}

	return response
}

/*
 * Dispatch CGI requests to the corresponding CGI handlers.
 */
func (this *controllerStruct) dispatch(request webserver.HttpRequest) webserver.HttpResponse {
	cgi := request.Params["cgi"]
	response := webserver.HttpResponse{}

	/*
	 * Find the right CGI to handle the request.
	 */
	switch cgi {
	case "get-tile":
		response = this.getTileHandler(request)
	case "render":
		response = this.renderHandler(request)
	default:
		response = this.errorHandler(request)
	}

	return response
}

/*
 * Initialize the controller.
 */
func (this *controllerStruct) initialize() error {
	content, err := ioutil.ReadFile(CONFIG_PATH)

	/*
	 * Check if file could be read.
	 */
	if err != nil {
		return fmt.Errorf("Could not open config file: '%s'", CONFIG_PATH)
	} else {
		config := configStruct{}
		err = json.Unmarshal(content, &config)
		this.config = config

		/*
		 * Check if file failed to unmarshal.
		 */
		if err != nil {
			return fmt.Errorf("Could not decode config file: '%s'", CONFIG_PATH)
		} else {
			geoDataPath := config.GeoData
			contentGeo, err := ioutil.ReadFile(geoDataPath)

			/*
			 * Check if file could be read.
			 */
			if err != nil {
				return fmt.Errorf("Could not read geo data file '%s'.", geoDataPath)
			} else {
				dataSet, err := geo.FromBytes(contentGeo)

				/*
				 * Check if geo data could be decoded.
				 */
				if err != nil {
					msg := err.Error()
					return fmt.Errorf("Could not decode geo data file '%s': %s", geoDataPath, msg)
				} else {
					numLocs := dataSet.LocationCount()
					data := make([]geo.Location, numLocs)

					/*
					 * Iterate over the locations and project them.
					 */
					for i := 0; i < numLocs; i++ {
						loc, err := dataSet.LocationAt(i)

						/*
						 * Verify that location could be obtained.
						 */
						if err == nil {
							data[i] = loc.Optimize()
						}

					}

					this.data = data
					this.colorMapping = color.DefaultMapping()
					cachePath := config.MapCache
					uri := config.MapServer
					useMap := config.UseMap

					/*
					 * Create OSM tile source if map should be used
					 * and cache path is set.
					 */
					if useMap && cachePath != "" {
						tileSource := tile.CreateOSMSource(uri, cachePath)
						this.tileSource = tileSource
					} else {
						this.tileSource = nil
					}

					return nil
				}

			}

		}

	}

}

/*
 * Main routine of our controller. Performs initialization, then runs the message pump.
 */
func (this *controllerStruct) Operate() {
	err := this.initialize()

	/*
	 * Check if initialization was successful.
	 */
	if err != nil {
		msg := err.Error()
		msgNew := "Initialization failed: " + msg
		fmt.Printf("%s\n", msgNew)
	} else {
		cfg := this.config
		serverCfg := cfg.WebServer
		server := webserver.CreateWebServer(serverCfg)

		/*
		 * Check if we got a web server.
		 */
		if server == nil {
			fmt.Printf("%s\n", "Web server did not enter message loop.")
		} else {
			requests := server.RegisterCgi("/cgi-bin/locviz")
			server.Run()
			tlsPort := serverCfg.TLSPort
			fmt.Printf("Web interface ready: https://localhost:%s/\n", tlsPort)

			/*
			 * A worker processing HTTP requests.
			 */
			worker := func(requests <-chan webserver.HttpRequest) {

				/*
				 * This is the actual message pump.
				 */
				for request := range requests {
					response := this.dispatch(request)
					respond := request.Respond
					respond <- response
				}

			}

			numCPU := runtime.NumCPU()

			/*
			 * Spawn as many workers as we have CPUs.
			 */
			for i := 0; i < numCPU; i++ {
				go worker(requests)
			}

			stdin := os.Stdin
			scanner := bufio.NewScanner(stdin)

			/*
			 * Read from standard input forever.
			 */
			for {
				scanner.Scan()
			}

		}

	}

}

/*
 * Pre-fetch tile data from OSM.
 */
func (this *controllerStruct) Prefetch(zoomLevel uint8) {
	err := this.initialize()

	/*
	 * Check if initialization was successful.
	 */
	if err != nil {
		msg := err.Error()
		msgNew := "Initialization failed: " + msg
		fmt.Printf("%s\n", msgNew)
	} else {
		tileSource := this.tileSource
		tileSource.Prefetch(zoomLevel)
	}

}

/*
 * Creates a new controller.
 */
func CreateController() Controller {
	controller := controllerStruct{}
	return &controller
}
