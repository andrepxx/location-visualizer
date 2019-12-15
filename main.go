package main

import (
	"flag"
	"fmt"
	"github.com/andrepxx/location-visualizer/controller"
)

const (
	PREFETCH_LIMIT = 8
)

/*
 * The entry point of our program.
 */
func main() {
	prefetch := flag.Int("prefetch", -1, "Prefetch tile data from OSM up to this zoom level")
	hard := flag.Bool("hard", false, "Disable the limitation of pre-fetching only low zoom levels")
	flag.Parse()
	prefetchZoom := *prefetch
	hardFlag := *hard
	cn := controller.CreateController()

	/*
	 * If we shall pre-fetch OSM tiles, do it.
	 * Otherwise, start our the server.
	 */
	if (prefetchZoom >= 0) && (prefetchZoom < 256) {

		/*
		 * Limit prefetch to level 8, unless "-hard" is specified.
		 */
		if (prefetchZoom > PREFETCH_LIMIT) && (!hardFlag) {
			msg := "Zoom level %d requested, but limited to %d to avoid high load on OSM infrastructure.\n"
			fmt.Printf(msg, prefetchZoom, PREFETCH_LIMIT)
			prefetchZoom = PREFETCH_LIMIT
		}

		zoomLevel := uint8(prefetchZoom)
		cn.Prefetch(zoomLevel)
	} else {
		cn.Operate()
	}

}
