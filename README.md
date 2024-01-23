# location-visualizer

This software allows you to perform fitness / activity and location tracking, as well as visualization of that data, on your own infrastructure.

Data can be imported from GeoJSON files, for example from your location history, which you can obtain from Google Takeout, and stored in a local database, which uses the OpenGeoDB binary file format for efficient storage and access of geographic features.

Starting with v1.3.0, data can also be imported from GPX files, which are usually created by dedicated GPS devices or dedicated GPS tracking apps. This is also in response to Google announcing its new version of the Timeline feature, for which location history data will be kept on the device itself instead of being stored in the cloud. It is unclear whether the export of location data will still be possible with the new system and in what format this data would be. In addition, this also makes *location-visualizer* interoperable with a wide range of GPS devices and related software, most of which can handle files in GPX format.

The software displays the aggregated location data as an interactive plot that you can navigate with either mouse and scroll wheel on your computer or with touch input on a mobile device.

It also allows you to annotate your location data with metadata like time stamps and begin of exercises, distances travelled, energy used, etc.

In addition, the software also allows export of the aggregated location data as OpenGeoDB, CSV, GeoJSON, and, as of v1.3.0, also GPX files.

## Building the software

To download and build the software from source for your system, run the following commands in a shell.

```
cd ~/go/src/
go get -d github.com/andrepxx/location-visualizer
cd github.com/andrepxx/location-visualizer/
make keys
make
```

This will create an RSA key pair for the TLS connection between the user-interface and the actual data processing backend (`make keys`) and then build the software for your system (`make`). The resulting executable is called `locviz`.

Location data will be stored in the file `data/locations.geodb`, while activity data is stored in `data/activitydb.json` and user account data is stored in `data/userdb.json`. All these paths can be adjusted in `config/config.json`.

To use the software, create a user, set a password and add permissions to fetch tiles, render data overlays, read and write activity data, read from and write to the geographical database, as well as download its contents.

```
./locviz create-user root
./locviz set-password root secret
./locviz add-permission root get-tile
./locviz add-permission root render
./locviz add-permission root activity-read
./locviz add-permission root activity-write
./locviz add-permission root geodb-read
./locviz add-permission root geodb-write
./locviz add-permission root geodb-download
```

Finally, run the server.

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

## Uploading geo data

To upload geo data to the database, log in with a user account, which has at least `geodb-read` and `geodb-write` permissions. Open the sidebar, click on the *GeoDB* button, then choose the import and sort strategies from the dropdown. Afterwards, open a file explorer on your system and move the GeoJSON files via drag and drop into the browser window. An import report will be displayed after the data has been imported.

## Clearing the database

To clear the database, you will have to terminate the application and delete the database file storing the geo data. (This will by default reside under `data/locations.geodb`.) An empty database will be created on next startup of the application.

There is currently no way to clear the database from within the web interface. This serves to prevent accidental deletion of data. In the future, we plan to offer clearing the database on the web interface as well, and put in other safeguards to prevent accidental deletion of data.
