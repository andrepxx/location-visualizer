# location-visualizer

This software allows you to perform fitness / activity and location tracking, as well as visualization of that data, on your own infrastructure.

Data can be imported from JSON files, for example from your location history, which you can obtain from Google Takeout, and stored in a local database, which uses the OpenGeoDB binary file format for efficient storage and access of geographic features.

Starting with v1.3.0, data can also be imported from GPX files, which are usually created by dedicated GPS devices or dedicated GPS tracking apps. This is also in response to Google announcing its new version of the Timeline feature, for which location history data will be kept on the device itself instead of being stored in the cloud. It is unclear whether the export of location data will still be possible with the new system and in what format this data would be. In addition, this also makes *location-visualizer* interoperable with a wide range of GPS devices and related software, most of which can handle files in GPX format.

Starting from v1.7.0, data can also be imported from CSV files as defined in RFC 4180. This is useful to "round-trip" data that has been exported by *location-visualizer*. Since the CSV format provides little metadata and can contain a variety of data in a variety of different formats, *location-visualizer* will, in an attempt to minimize data corruption by user error, reject a lot of data that is not of the same format that *location-visualizer* produces. For exchanging data with third-party applications, please perfer using the more structured GPX or JSON formats. The CSV format is mainly useful for exchange with data analysis software, like *R* or *Pandas*, or spreadsheet applications that are part of common office software suites.

Starting from v1.9.0, data can also be imported from files in OpenGeoDB format.

The software displays the aggregated location data as an interactive plot that you can navigate with either mouse and scroll wheel on your computer or with touch input on a mobile device.

It also allows you to annotate your location data with metadata like time stamps and begin of exercises, distances travelled, energy used, etc.

In addition, the software also allows export of the aggregated location data as OpenGeoDB, CSV, JSON, and, as of v1.3.0, also GPX files.

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

Location data will be stored in the file `data/locations.geodb`, while activity data is stored in `data/activitydb.json`, user account data is stored in `data/userdb.json`, and map data / tiles are cached in `data/tile.bin` and `data/tile.idx`. All these paths can be adjusted in `config/config.json`.

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
- `cleanup-tiles`: Perform a cleanup of the tile database.
- `clear-password name`: Set the password of user `name` to an empty string.
- `create-user name`: Create a new user `name`.
- `export-tiles path/file.tar.gz`: Export map tiles from tile database to `path/file.tar.gz`.
- `has-permission name permission`: Check if user `name` has permission `permission`.
- `import-tiles path/file.tar.gz`: Import map tiles to tile database from `path/file.tar.gz`.
- `list-permissions name`: List all permissions of user `name`.
- `list-users`: List all users.
- `remove-permission name permission`: Removes the permission `permission` from the user `name`.
- `remove-user name`: Removes the user `name`.
- `set-password name password`: Sets the password of user `name` to `password`.

## Integration with a map service like OpenStreetMaps

This software can use data from sources of map data, like the OpenStreetMaps project (OSM), to plot location data overlaid on an actual map. However, since OpenStreetMaps is a free service running on donated ressources, access to the map data is rather slow for "third-party" users (i. e. everything but the "official" openstreetmaps.org map viewer). When OSM integration is enabled on both server and client side, the application may become slow / unresponsive until a significant amount of data has been replicated to the server's local cache. In addition, we do not want to place an unnecessary burden on OSM servers. Therefore, OSM integration is disabled via the configuration file when you download this software, and we strongly suggest that you keep it disabled unless you actually **need** it.

To enable integration with a map service, open the `config/config.json` file and replace the entry `"UseMap": false,` with `"UseMap": true,`. Then enter the URL of the map server to use, making use of the placeholders `${z}`, `${x}` and `${y}` for zoom level, X and Y coordinate of the map tile, respectively.

Therefore, a URL might look like the following: `https://tile.example.com/${z}/${x}/${y}.png`

Replace `tile.example.com` with the domain name (or IP address) of the actual tile server you want to use. This can be a public tile-server or one that you self-host. If you use a public tile server, please **pay close attention** to the provider's tile usage policy.

When enabled, note that response from the server may be **very** slow until a significant amount of map data has been cached locally. Map data stored in the cache never expires and can therefore become outdated. A proper cache update mechanism is not implemented yet. Also note that there is no bound up to which the cache will grow. **All** data fetched from OSM **will** be cached by the server indefinitely, in order to minimize the load on the map provider's infrastructure.

The tile cache is stored in binary files that use a proprietary (*location-visualizer* specific) file format. However, an interface is provided to import map tiles from or export map tiles to *Gzip*-compressed tarballs (`.tar.gz` files). To import data from a directory, you will have to archive it. The directory inside the archive **needs** to have the name `tile/` for the import to succeed. If you still have a "legacy" cache directory (from *location-visualizer* versions before v1.8.0), and you did not change the file naming conventions, you can archive the directory (the directory itself, **not** just the files within it) and import the result.

### Pre-fetching data from a map service

Since the application will be unresponsive unless all map data required to display the current viewport has been fetched from OSM, this application allows to pre-fetch map data from OSM in a bulk transfer. This is useful after initial setup, since otherwise, it may take a **very** long time to navigate even zoomed-out views of the map. Pre-fetch of map data may take a few hours. We suggest to pre-fetch map data up to a zoom level of 7 or 8.

```
./locviz -prefetch 7
```

If you want to pre-fetch zoom levels beyond 8, you will have to additionally specify the `-hard` option in order to confirm that you are aware that you are placing a significant load on OSM infrastructure, that the pre-fetch will take a long time and will use a lot of disk space (perhaps even more than you might have available on your system, potentially rendering it unstable).

### Importing and exporting map data

If you use *location-visualizer* v1.8.0 or newer, map tiles are stored in a binary database that consists of two files, normally residing under `data/tile.bin` and `data/tile.idx`, respectively. These two files always belong together, so backup, restore, delete, ... them always together. You can export the contents of the tile database to an archive using the `export-tiles` command, and import tiles from an archive into the database using the `import-tiles` command.

To reclaim storage occupied by outdated (unreferenced) images, you can run the `cleanup-tiles` command.

## Uploading geo data

To upload geo data to the geo database, log in with a user account, which has at least `geodb-read` and `geodb-write` permissions. Open the sidebar, click on the *GeoDB* button, then choose the import and sort strategies from the dropdown. Afterwards, open a file explorer on your system and move the CSV, GPX or JSON files via drag and drop into the browser window. An import report will be displayed after the data has been imported.

## Clearing the geo database

To clear the database, you will have to terminate the application and delete the database file storing the geo data. (This will by default reside under `data/locations.geodb`.) An empty database will be created on next startup of the application.

There is currently no way to clear the database from within the web interface. This serves to prevent accidental deletion of data. In the future, we plan to offer clearing the database on the web interface as well, and put in other safeguards to prevent accidental deletion of data.

## Exchanging data with location-visualizer

Please refer to [our documentation of data formats](doc/data-formats.md) if you want to exchange location and / or activity data with *location-visualizer*.
