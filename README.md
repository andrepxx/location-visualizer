# location-visualizer

This software visualizes your location history, which you can obtain from Google Takeout, and displays it as an interactive plot that you can navigate with either mouse and scroll wheel on your computer or with touch input on a mobile device.

## Building the software

To download and build the software from source for your system, run the following commands in a shell (assuming that `~/go` is your `$GOPATH`).

```
cd ~/go/src/
go get -d github.com/andrepxx/location-visualizer
cd github.com/andrepxx/location-visualizer/
make keys
make
```

This will create an RSA key pair for the TLS connection between the user-interface and the actual data processing backend (`make keys`) and then build the software for your system (`make`). The resulting executable is called `locviz`.

Put your location data JSON file under `data/Standortverlauf.json` or adjust the path to the data file in `config/config.json`, then run the executable.

```
./locviz
```

After the following message appears in your console ...

```
Web interface ready: https://localhost:8443/
```

... point your web browser to <https://localhost:8443/> to fire up the web interface and interact with the visualization.

## Work in progress

This software is still work in progress. We currently work on integration with sources of map data like OpenStreetMaps (OSM) to plot location data overlaid on an actual map. However, the *tile* package, which is responsible for interaction with OSM, is not complete yet. Therefore, OSM integration is currently disabled via the configuration file and we suggest that you keep it disabled. When enabled, the application will start fetching tiles and therefore create traffic on OSM servers without displaying anything on the map yet. It will also make the display of the location data slower, since the server will only generate a response after all relevant map tiles for the area about to be displayed have been fetched from OSM.

