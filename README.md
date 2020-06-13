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

Put your location data JSON file under `data/Standortverlauf.json` or adjust the path to the data file in `config/config.json`.

Next, create a user, set a password and add permissions to fetch tiles and render data overlays.

```
./locviz create-user root
./locviz set-password root secret
./locviz add-permission root get-tile
./locviz add-permission root render
```

Finally, run the executable.

```
./locviz
```

After the following message appears in your console ...

```
Web interface ready: https://localhost:8443/
```

... point your web browser to <https://localhost:8443/> to fire up the web interface and interact with the visualization.

Log in with the user name and password defined above, in our example, these were `root` and `secret`, respectively.

Commands:

- `add-permission name permission`: Adds the permission `permission` to the user `name`.
- `clear-password name`: Set the password of user `name` to an empty string.
- `create-user name`: Create a new user `name`.
- `has-permission name permission`: Check if user `name` has permission `permission`.
- `list-permissions name`: List all permissions of user `name`.
- `list-users`: List all users.
- `remove-permission name permission`: Removes the permission `permission` from the user `name`.
- `remove-user name`: Removes the user `name`.
- `set-password name password`: Sets the password of user `name` to `password`.

## Integration with a map service like OpenStreetMaps

This software can use data from sources of map data, like the OpenStreetMaps project (OSM), to plot location data overlaid on an actual map. However, since OpenStreetMaps is a free service running on donated ressources, access to the map data is rather slow for "third-party" users (i. e. everything but the "official" openstreetmaps.org map viewer). When OSM integration is enabled on both server and client side, the server will only generate a response after all map data required to display the current viewport has been fetched from OSM, which will make the application unresponsive until a significant amount of data has been replicated to the server's local cache. In addition, we do not want to place an unnecessary burden on OSM servers. Therefore, OSM integration is disabled via the configuration file when you download this software, and we strongly suggest that you keep it disabled unless you actually **need** it.

To enable integration with a map service, open the `config/config.json` file and replace the entry `"UseMap": false,` with `"UseMap": true,`. Then enter the URL of the map server to use, making use of the placeholders `${z}`, `${x}` and `${y}` for zoom level, X and Y coordinate of the map tile, respectively.

Therefore, a URL might look like the following: `https://tile.example.com/${z}/${x}/${y}.png`

Replace `tile.example.com` with the domain name (or IP address) of the actual tile server you want to use. This can be a public tile-server or one that you self-host. If you use a public tile server, please **pay close attention** to the provider's tile usage policy.

When enabled, note that response from the server will be **very** slow until a significant amount of map data has been cached locally. Map data stored in the cache never expires and can therefore become out-of-date. A proper cache update mechanism is not implemented yet. Also note that there is no bound up to which the cache will grow. **All** data fetched from OSM **will** be cached by the server indefinitely, in order to minimize the load on the map provider's infrastructure.

### Pre-fetching data from a map service

Since the application will be unresponsive unless all map data required to display the current viewport has been fetched from OSM, this application allows to pre-fetch map data from OSM in a bulk transfer. This is useful after initial setup, since otherwise, it will take a **very** long time to navigate even zoomed-out views of the map. Pre-fetch of map data may take a few hours. We suggest to pre-fetch map data up to a zoom level of 7 or 8.

```
./locviz -prefetch 7
```

If you want to pre-fetch zoom levels beyond 8, you will have to additionally specify the `-hard` option in order to confirm that you are aware that you are placing a significant load on OSM infrastructure, that the pre-fetch will take a long time and will use a lot of disk space (perhaps even more than you might have available on your system, potentially rendering it unstable).

