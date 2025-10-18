# location-visualizer

This software allows you to perform fitness / activity and location tracking, as well as visualization of that data, on your own infrastructure.

Data can be imported from JSON files, for example from your location history, which you can obtain from Google Takeout, and stored in a local database, which uses the OpenGeoDB binary file format for efficient storage and access of geographic features.

Starting with v1.3.0, data can also be imported from GPX files, which are usually created by dedicated GPS devices or dedicated GPS tracking apps. This is also in response to Google announcing its new version of the Timeline feature, for which location history data will be kept on the device itself instead of being stored in the cloud. It is unclear whether the export of location data will still be possible with the new system and in what format this data would be. In addition, this also makes *location-visualizer* interoperable with a wide range of GPS devices and related software, most of which can handle files in GPX format.

Starting from v1.7.0, data can also be imported from CSV files as defined in RFC 4180. This is useful to "round-trip" data that has been exported by *location-visualizer*. Since the CSV format provides little metadata and can contain a variety of data in a variety of different formats, *location-visualizer* will, in an attempt to minimize data corruption by user error, reject a lot of data that is not of the same format that *location-visualizer* produces. For exchanging data with third-party applications, please perfer using the more structured GPX or JSON formats. The CSV format is mainly useful for exchange with data analysis software, like *R* or *Pandas*, or spreadsheet applications that are part of common office software suites.

Starting from v1.9.0, data can also be imported from files in OpenGeoDB format. The OpenGeoDB format is used by *location-visualizer* for its internal storage, but also by some recent GNSS receiver / logger modules as their internal storage format or wire format that they use to send position information to a host, usually over a serial interface. Supporting this "native" / wire format directly in *location-visualizer* means that the output of some GNSS modules may be imported into *location-visualizer* directly or with little post-processing. Of course, it is still possible to turn the raw GNSS receiver output into a more structured and higher-level representation, like CSV or GPX, and then import it into *location-visualizer*.

Data can also be streamed live from sensors deployed in a field or submitted by other applications. Especially, since v1.12.0, a token-based authentication method is allowed for data submission by third-party applications or IoT devices, such as field sensors, which do not implement the full challenge-response based authentication protocol, which is usually required for integration with *location-visualizer*.

The software displays the aggregated location data as an interactive plot that you can navigate with either mouse and scroll wheel on your computer or with touch input on a mobile device.

It also allows you to annotate your location data with metadata like time stamps and begin of exercises, distances travelled, energy used, etc.

In addition, the software also allows export of the aggregated location data as OpenGeoDB, CSV, JSON, and, as of v1.3.0, also GPX files.

## Building the software

To download and build the software from source for your system, run the following commands in a shell.

```bash
cd ~/go/src/
go get -d github.com/andrepxx/location-visualizer
cd github.com/andrepxx/location-visualizer/
make keys
make
```

This will create an RSA key pair for the TLS connection between the user-interface and the actual data processing backend (`make keys`) and then build the software for your system (`make`). The resulting executable is called `locviz`.

Location data will be stored in the file `data/locations.geodb`, while activity data is stored in `data/activitydb.json`, user account data is stored in `data/userdb.json`, and map data / tiles are cached in `data/tile.bin` and `data/tile.idx`. All these paths can be adjusted in `config/config.json`.

*location-visualizer* is a scalable, high-performance software solution that not only supports private, single-user scenarios, but also larger, enterprise-grade deployments. Therefore, besides supporting state-of-the-art cryptography, like TLS 1.3 in combination with RSA, ECDH over Curve25519, ChaCha20 and Poly1305, it also features strong, challenge-response based user authentication and secure storage of passwords as salted hashes.

Therefore, before the software can be used, at least one user account has to be created. Then, set a password and add permissions to fetch tiles, render data overlays, read and write activity data, read from and write to the geographical database, as well as download its contents.

**HINT:** These commands are to be executed in a shell when the server does **not** run. User management is currently completely "offline". This is mainly a security measure since we do not want user or permissions management to be available remotely, even in the case of a "privileged" account getting compromised. However, we do consider adding some form of "dynamic", on-demand user management at a later point in time for improved maintenance and scalability.

```bash
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

Optionally, if you want to allow clearing the geographical database, you can also add a permission for that.

```bash
./locviz add-permission root geodb-clear
```

Finally, run the server.

```bash
./locviz
```

After the following message appears in your console ...

```
Web interface ready: https://localhost:8443/
```

... point your web browser to <https://localhost:8443/> to fire up the web interface and interact with the visualization.

Log in with the user name and password defined above, in our example, these were `root` and `secret`, respectively.

Commands:

- `add-permission name permission`: Add the permission `permission` to the user `name`.
- `cleanup-tiles`: Perform a cleanup of the tile database.
- `clear-password name`: Set the password of user `name` to an empty string.
- `create-device-token name description`: Create a new device token associated with user `name` and description `description`. (Hint: Put quotes around the description.)
- `create-user name`: Create a new user `name`.
- `export-tiles path/file.tar.gz`: Export map tiles from tile database to `path/file.tar.gz`.
- `has-device-token name token`: Check if user `name` has device token `token` associated.
- `has-permission name permission`: Check if user `name` has permission `permission`.
- `import-tiles path/file.tar.gz`: Import map tiles to tile database from `path/file.tar.gz`.
- `list-device-tokens name`: List all device tokens associated with user `name`, along with their creation time and an optional description.
- `list-permissions name`: List all permissions of user `name`.
- `list-users`: List all users.
- `remote command host port certificate_file name password [...]`: Perform `command` on remote host `host` : `port`. Verify the server's certificate against `certificate_file` and authenticate as user `name` using `password`. (See command-line client section below.)
- `remove-device-token name token`: Remove the device token `token` from user `name`.
- `remove-permission name permission`: Remove the permission `permission` from the user `name`.
- `remove-user name`: Remove the user `name`.
- `set-password name password`: Set the password of user `name` to `password`.

## Integration with a map service like OpenStreetMap

This software can use data from sources of map data, like the OpenStreetMap project (OSM), to plot location data overlaid on an actual map. However, since OpenStreetMap is a free service running on donated ressources, access to the map data is rather slow for "third-party" users (i. e. everything but the "official" openstreetmap.org map viewer). When OSM integration is enabled on both server and client side, the application may become slow / unresponsive until a significant amount of data has been replicated to the server's local cache. In addition, we do not want to place an unnecessary burden on OSM servers. Therefore, OSM integration is disabled via the configuration file when you download this software, and we strongly suggest that you keep it disabled unless you actually **need** it.

To enable integration with a map service, open the `config/config.json` file and replace the entry `"UseMap": false,` with `"UseMap": true,`. Then enter the URL of the map server to use, making use of the placeholders `${z}`, `${x}` and `${y}` for zoom level, X and Y coordinate of the map tile, respectively.

Therefore, a URL might look like the following: `https://tile.example.com/${z}/${x}/${y}.png`

Replace `tile.example.com` with the domain name (or IP address) of the actual tile server you want to use. This can be a public tile-server or one that you self-host. If you use a public tile server, please **pay close attention** to the provider's tile usage policy.

When enabled, note that response from the server may be **very** slow until a significant amount of map data has been cached locally. Map data stored in the cache never expires and can therefore become outdated. A proper cache update mechanism is not implemented yet. Also note that there is no bound up to which the cache will grow. **All** data fetched from OSM **will** be cached by the server indefinitely, in order to minimize the load on the map provider's infrastructure.

The tile cache is stored in binary files that use a proprietary (*location-visualizer* specific) file format. However, an interface is provided to import map tiles from or export map tiles to *Gzip*-compressed tarballs (`.tar.gz` files). To import data from a directory, you will have to archive it. The directory inside the archive **needs** to have the name `tile/` for the import to succeed. If you still have a "legacy" cache directory (from *location-visualizer* versions before v1.8.0), and you did not change the file naming conventions, you can archive the directory (the directory itself, **not** just the files within it) and import the result.

### Pre-fetching data from a map service

Pre-fetching of map data was more relevant in the past, when OSM heavily throttled third-party applications and *location-visualizer*'s tile handling wasn't as efficient as it is now and we generally don't recommend it anymore, in order not to cause unnecessary load on the map provider's infrastructure.

If you still want to prefetch map data, we suggest to pre-fetch map data up to a zoom level of 7 or 8.

```bash
./locviz -prefetch 7
```

If you want to pre-fetch zoom levels beyond 8, you will have to additionally specify the `-hard` option in order to confirm that you are aware that you are placing a significant load on the map provider's infrastructure, that the pre-fetch will take a long time and will use a lot of disk space (perhaps even more than you might have available on your system, potentially rendering it unstable).

### Importing and exporting map data

If you use *location-visualizer* v1.8.0 or newer, map tiles are stored in a binary database that consists of two files, normally residing under `data/tile.bin` and `data/tile.idx`, respectively. These two files always belong together, so backup, restore, delete, ... them always together. You can export the contents of the tile database to an archive using the `export-tiles` command, and import tiles from an archive into the database using the `import-tiles` command.

To reclaim storage occupied by outdated (unreferenced) images, you can run the `cleanup-tiles` command.

## Uploading geo data

To upload geo data to the geo database, log in with a user account, which has at least `geodb-read` and `geodb-write` permissions. Open the sidebar, click on the *GeoDB* button, then choose the import and sort strategies from the dropdown. Afterwards, open a file explorer on your system and move the CSV, GPX or JSON files via drag and drop into the browser window. An import report will be displayed after the data has been imported.

## Interaction via command-line client

Starting from version v1.11.0, *location-visualizer* also implements a command-line client. It is accessed via the command `./locviz remote [...]`. Each command expects further parameters. The first five parameters (`host`, `port`, `certificate_file_path`, `name`, `password`) are required for connection and session establishment and are common to all commands.

The following commands are currently supported:

- `export-activity-csv host port certificate_file_path name password output_file_path`: Export activity CSV data to path `output_file_path`. (File must not exist!)
- `export-geodata host port certificate_file_path name password format output_file_path`: Export data from the geographical database to path `output_file_path`. (File must not exist!) Use format `format`, which may be any of `opengeodb`, `csv`, `json`, `json-pretty`, `gpx` or `gpx-pretty`.
- `import-geodata host port certificate_file_path name password format strategy input_file_path`: Import data into the geographical database from path `input_file_path`. Use format `format`, which may be any of `opengeodb`, `csv`, `json` or `gpx`, and import strategy `strategy`, which may be any of `all`, `newer` or `none`.

The certificate file provided must be in PEM format. The certificate chain that the server provides is verified to match **exactly** against the one provided in the certificate file.

## Clearing the geo database

To clear the database, you can terminate the application and delete the database file storing the geo data. (This will by default reside under `data/locations.geodb`.) An empty database will be created on next startup of the application.

The database can also be cleared from within the web interface if the respective account performing the action has the required permission. For that, a SHA-512 hash of the database contents (in OpenGeoDB representation) has to be provided. This serves as proof that the person deciding to clear the database has downloaded a backup copy before and serves to prevent accidental deletion of data. You can prevent users from clearing the database by not giving them the required permission.

## Exchanging data with location-visualizer

Please refer to [our documentation of data formats](doc/data-formats.md) if you want to exchange location and / or activity data with *location-visualizer*.

If you want to implement a client interfacing with *location-visualizer*, for example for continuously uploading coordinate data, then, in addition to the data format specification referenced above, please also refer to the [protocol specification](doc/communications-protocol.md). Note that the *location-visualizer* project already provides implementations for both a web-based UI client (supplied by the server) and a command-line client.

## Integrating third-party applications and IoT devices

To provision a new third-party application or IoT device for an existing user account for submission of coordinate data, run the following command.

```bash
./locviz create-device-token root "Some description for the device"
```

Replace `root` by the name of the user you want to provision the third-party application or device to. A device can only submit data as long as the user it is associated with / provisioned for has the `geodb-write` permission.

This will return a device token in hexadecimal encoding, which the IoT device or third-party app can then use to submit data to an instance of `location-visualizer` running on a publicly-accessible server.

Use the following endpoint to submit your data: `https://[hostname]:[port]/cgi-bin/locviz?cgi=submit-coordinates&name=[your username]&devicetoken=[the device token]&time=[timestamp]&latitude=[latitude]&longitude=[longitude]`

For example, when provisioning the *mandhak/gpslogger* app, you can use the following URI with placeholders: `https://[hostname]:[port]/cgi-bin/locviz?cgi=submit-coordinates&name=[your username]&devicetoken=[the device token]&time=%TIME&latitude=%LAT&longitude=%LON`

You can provision as many devices (generate as many device tokens) for a user as you want.

Replace all the placeholders in square brackets with the appropriate values.

You can list all device tokens currently associated with a user with the following command.

```bash
./locviz list-device-tokens root
```

This not only lists the tokens themselves (their hexadecimal values), but also the (UTC) time when they were generated and the description that was provided when they were generated.

In case a device token gets compromised or a sensor gets decommissioned, you can remove (or dissociate) it from its user again.

```bash
./locviz remove-device-token root [the device token]
```

The capability to capture live sensor data, in conjunction with its multi-user capabilities, strong authentication, fine-grained access control and support for a multitude of standardized formats (including GNSS wire formats), make *location-visualizer* suitable for professional (commercial or government) applications, which involve capturing live data from sensors deployed in the field. The ability to provision device tokens extends the capability to submit coordinate data to third-party devices or applications, which do not implement the more secure challenge-response based authentication methods and application protocol, allowing for easy integration of third-party sensors into your workflow. At the same time, access to these less secure types of sensors or applications is limited to submitting coordinates, keeping the collected data secure and the attack surface minimal.

## Q and A

**Q:** Is *location-visualizer* the name of specifically this software or is it a generic term, for example the name for a category of software like this?

**A:** *location-visualizer* is the proper name of exactly this piece of software. It is not a generic term for a class or category of software. The class or category of software *location-visualizer* belongs to, which deals with analyzing and visualizing geospatial data, is commonly called "geo-information systems" or GIS. To the best of our knowledge, this piece of software ("andrepxx/location-visualizer") is, as of October 2025, also the only software with the name "location-visualizer" we could find in an online search. An online search does not reveal the term "location-visualizer" to be used to classify a broader category of software.

**Q:** Is *location-visualizer* a community project?

**A:** *location-visualizer* is open-source software licensed under a free and permissive license, Apache license 2.0. Like most open-source software, it is a community project, even though the principal developer is a bit of a "benevolent dictator for life", having conceptualized the software, carrying basically all the development and also representing the project to the outside world.

**Q:** I also heard of "location history visualizer", "GPS visualizer", etc. How is *location-visualizer* related to these?

**A:** There are pieces of software or online services with similar names, like "location history visualizer", "GPS visualizer" and so on, which are from different authors and - despite potentially similar or overlapping functionality - are **not** related to *location-visualizer* in any way.

**Q:** Who uses *location-visualizer* and what is it commonly used for?

**A:** This is always hard to tell with open-source software, which anyone can freely download, clone, modify, build, fork and which does not have any telemetry or anything similar embedded. What we know is that the user base of *location-visualizer* is very diverse. On one hand, there is definitely a group of private individuals, who use *location-visualizer* as their personal fitness, activity and location-tracking service, especially since Google's *Timeline* feature is no longer available on the web. On the other hand, we are aware of some professional, commercial or government uses of *location-visualizer* as well. Examples include transportation companies visualizing relations within their network or tracking and visualizing the positions of their vehicles, mobile network operators optimizing their network by visualizing the flow of mobile nodes between regions covered by different cellular towers, but also public health authorities making use of *location-visualizer* to visualize movement data of individuals during a disease outbreak, like that of the COVID-19 pandemic from December 2019 till May 2023, either specifically for contact tracing or as a more general analysis of crowd behaviour like larger-scale traffic flows or general mobility analysis. Potential uses also include environmental research like wildlife tracking. Last but not least, some individuals within the OpenStreetMap project also use *location-visualizer* to review and improve the map, for example by overlaying GPS traces they captured onto the OpenStreetMap material and looking for discrepancies. On the other hand we, as developers of *location-visualizer*, in turn also support and contribute to OpenStreetMap.

**Q:** To the developer / maintainer of the project: Why did you develop *location-visualizer* and what do you mainly use it for?

**A:** This will get long as there are many different aspects to it, which in the end led to the development of what is now *location-visualizer*.

Towards the end of 2018, I came across a few research papers about new concepts in data visualization. Some were talking about concepts that we now find in a data visualization technique that is now known as "abstract rendering", which can specifically deal with very large datasets and attempts to visualize them with minimal data loss at a given resolution and color space, preserving as much detail as possible. This technique is, for example, implemented for the programming language *Python* in a library called *Datashader* by the *HoloViz* project and used mainly by data scientists to visualize large sets of coordinate data. My goal was to implement a somewhat reduced and more rigid, but thereby also higher-performing variant of this concept in the *Go* programming language, which has a considerable advantage over *Python* regarding execution speed anyhow, since *Go* is strongly and very strictly typed, ahead-of-time compiled and statically linked. So I went ahead and created the *sydney* graphics library, which implements high-performance data visualization in *Go*.

After I was somewhat confident about my implementation, I thought about what one might actually use this for. I recalled that Google had a lot of location data stored in my account, and that existing GPS visualization tools like *Google Earth* or similar would never be able to keep up with this amount of data. That's how it all began.

See, I'm very active in the open-source and "hacker" community, especially in Berlin, at least since 2013. My projects, which also include the real-time signal processing software *go-dsp-guitar*, drew quite a bit of attention in the community and are well-renowned and -recognized open-source projects and widely used pieces of software by now. As of September 2025, the *go-dsp-guitar* repository has been accessed over 80000 times and the *location-visualizer* repository has been accessed over 20000 times. We can assert that these pieces of software have a significant user base, especially given their rather specific target audience and the fact that they have no commercial company and therefore no professional marketing behind them. As a result of that, I also get invited to present my work at conferences, etc., which are related to open-source software and mostly organized by non-profit organizations from this area or by enthusiasts, mainly from circles close to Berlin's (or another city's) "hacker community". Besides that, I also have a job in research, working as a scientist or research associate at a large, publicly-owned research institute. Last but not least, I also have a comprehensive and very diverse social network in general, including people I know from university, my work as a researcher, my open-source involvement or other (mainly non-profit) things I do, like my involvement in culture, music and organizations which support disadvantaged or minority groups and many of the people who are "socially close" to me are far away geographically, often living in other countries, sometimes even on other continents, which is both a consequence of living in a multicultural city like Berlin, which also attracts many visitors, and of being present at events, which tend to also attract many people from further away.

All this results in me living a rather nomadic life, traveling and being on the move basically all the time, often over long distances. On the other hand, having my "home base" in Berlin, a vibrant and lively metropolis in the heart of Europe known for its historical sights giving insight into the city's turbulent past, but also cultural attractions, freedom and general quality of living with all its open and green spaces and all its diversity and of course its nightlife, I also often have visitors here and then I like to show them the city. So when I'm not heading from one conference to another, I tend to either explore Berlin (which is basically a lifetime task), introduce other people to Berlin's "secrets", or spend time in another place, often abroad, exploring a different place or spending time with friends and loved ones there.

I also do a lot of endurance sports, mainly running and cycling, and I ride a motorcycle. And I like to keep track of all that, both for statistics, but also for the memories. Often when I see a point on the map, I immediately know from the spatial relation what event I attended there or what other reason I had for being there, even though I usually won't recall the exact dates when I was there. So this location tracking is more or less a bit like a really detailed diary that writes itself and since, with all the involvement in research and open-source and other things, I do lead a rather exciting (but of course also demanding and often strenuous) life, there is really a lot to look at and it's just so much stuff that happens in such a short amount of time that I could never really keep track of and remember most of it. So having this logged automatically as I'm on the move and being able to see all of it is really cool. Seeing the data mapped out is often also a trigger, which then brings back more detailed and personal memories. And regarding endurance sports, having all these statistics available is also something that motivates me to improve. When you can quantify something, then you can see whether you improve and by how much and that will motivate you and you will keep improving. Regarding city exploration, when you have a record that allows you to see where you've been, that means you can also see where you **haven't** been yet. And then you can take a deeper look what is there that might look interesting and this way you can find exciting new places to explore.

A lot of development of *location-visualizer* also happened during the COVID-19 pandemic (between December 2019 and May 2023) and I personally know many people who, like probably most, suffered in different ways from its effects and who I supported through that challenging time. For me personally, it also had a great impact, mainly due to the travel restrictions that came with it, upended my "nomadic life" and cut me off (at least physically) from people who were socially close to me, but geographically far away. It definitely was a very tough time, especially since it was totally unclear how long it would last and how it would end. I did contribute to scientific studies at that time, also as a participant, since this made me feel like I could at least somewhat improve the situation, and it also motivated me to improve *location-visualizer* further, since I was aware of the fact that the software helped, for example, researchers analyse movement patterns and identify potential hotspots and contact points where people might have aggregated and spread the disease.

Last but not least, I already noticed quite early that the data, which Google provides via their *Takeout* service, was not super reliable. I mean normally, I would expect that, if you do a *Takeout* at some point in time, and do another one somewhat later, that the only difference between the two would be additional data points that came in after the previous export. However, I noticed that this was not always the case. Every now and then, Google would decide to change some past data, remove some data points, change some details. Especially, when I once went into the office of a public health institute, Google decided not just to remove the visit there, but also basically the entire trip towards that location and back again, making it seem like I was never there - probably because it is "health-related information" and they're not "supposed" to store it, but it still meant that the data that was coming through that service wasn't completely trustworthy. Therefore, I wanted to be able to at least "freeze" the data that I got from them, so that there would at least be a "canonical version" that I knew for sure wouldn't change / degrade anymore. That was one reason why I said there has to be a piece of software, which provides local storage, and can then aggregate new location data as it comes in, but the existing data would remain unchanged. Then I added support for other data formats, so that one could not only get data from Google via "the cloud", but also from physical GPS loggers and so on. So over time, this became a more comprehensive solution. Then Google decided to discontinue their cloud-based "Timeline" service and migrate to the new on-device version and it became clear that a lot of data gets lost in the process and that "Timeline" would no longer be accessible from the computer and that people had all sorts of issues with it. So I'm really glad that we have an alternative now - or by now we do in fact have multiple, but *location-visualizer* is one of them and it started quite early, back in 2019, when there basically wasn't much, if anything, else in that regard.

So there were lots of reasons for me to develop *location-visualizer* and my hope as an open-source developer, of course, is that it will not only be useful for me, but for many other people as well.

**Q:** Is *location-visualizer* a ready-to-use application or is it a library for building visualizations for location-based or geospatial data?

**A:** *location-visualizer* is a lightweight, ready-to-use ("production-ready" if you will) application. So far, there are no "official" binaries available, so it has to be built from source. However, that is not uncommon, specifically for Go-based open-source applications.

**Q:** Why don't you provide binary builds for *location-visualizer*?

**A:** *location-visualizer* is a server application intended mainly for professional use by data scientists and enthusiasts, who will be able to install the relevant toolchains, check out the source code from our repository and build this application from source. *location-visualizer* is fully portable and has no system-specific dependencies, so it can be built for basically any platform that the Go compiler toolchain supports. We won't be able to cover all platforms (combinations of instruction set architecture and operating system) with binary builds anyhow. We may consider starting to provide binary builds for certain platforms at a later point in time. However, currently this is not regarded as a priority.

**Q:** Is there support for "raw" and / or "semantic" location data in *location-visualizer*?

**A:** So far, there is only support for "raw" location data. After Google removed Timeline from the cloud, there is basically no "semantic" location information anymore and we do not want to use third-party geolocation services for reasons of privacy, but also independance. "Semantic" location is also known to be particularly unreliable, especially as the underlying map changes over time. To do it correctly, one would have to resolve each "semantic" location entry against the state of the map exactly at that time when it was captured / created.

**Q:** What file formats does *location-visualizer* support?

**A:** *location-visualizer* can import and export **coordinate data** in the OpenGeoDB binary format (\*.geodb), a comma-separated value (CSV) format following the RFC 4180 specification (\*.csv), the XML-based "GPS Exchange" format (\*.gpx) and the JSON-based "Records JSON" format (\*.json), and **activity data** in a CSV-based format.

**Q:** What is the OpenGeoDB file format?

**A:** The *OpenGeoDB* file format is a compact binary file format with an open specification that is used to store geographical data and enables fast (random) access. It is also the data format that *location-visualizer* uses internally for its geographical database.

**Q:** Is the OpenGeoDB file format proprietary to *location-visualizer*?

**A:** The format is neither proprietary nor limited to *location-visualizer*. Its compactness, fixed-sized record structure and binary representation make it well-suited as a so called *wire-format* for GNSS receivers, i. e. the data format that the receiver uses to send positional data to a receiving host (e. g. PC or microcontroller), for example over serial interfaces like I2C, RS232 or USB, or over network protocols like TCP or MQTT. In this sense, it can fulfil functions similar to what the NMEA 0183 protocol is doing. Similarly, the *OpenGeoDB* format can be used by hardware GNSS logging devices to store records in their internal memory, before they are potentially encoded into a different format and sent over the external interface. In fact, some positional devices use either the *OpenGeoDB* format, or a similar format derived from it, for exactly these purposes.

**Q:** Is the OpenGeoDB file format related to the postal code database of the same name?

**A:** No, the *OpenGeoDB* file or wire format for storing and / or transmitting positional data is not related to the hierarchical location and postal code database of the same name.

**Q:** What is the CSV format that *location-visualizer* supports for coordinate data?

**A:** Each record in the CSV file has the following fields ("columns") in exactly this order.

1. Timestamp
2. Latitude
3. Longitude

The timestamp is in the format described by *RFC 3339* and has millisecond precision.

Both latitude and longitude are unsigned fixed-point numbers with a variable amount of decimal digits to the left of the decimal separator (`.`) and exactly seven (7) digits to the right of the decimal separator, followed by a single letter describing the direction and thereby replacing the sign. The final letter will always be uppercase in data exported from *location-visualizer*. However, when importing data, *location-visualizer* is a bit more relaxed in the sense that it will also accept the respective lowercase letters in these positions.

Latitudes always have the letters `'N'` or `'S'`, for "north" and "south" (relative to the equator, or as indication of hemisphere), respectively, as the trailing characters in the string, while longitudes always have the letters `'E'` or `'W'`, for "east" and "west" (relative to the zero meridian), respectively, as the trailing characters in the string.

**Q:** How will *location-visualizer* handle GPX files?

**A:** When importing files in *GPS Exchange* (GPX) format, *location-visualizer* will only import all track points (`trkpt`) in all track segments (`trkseg`) in all tracks (`trk`) inside the GPX document root element (`gpx`).

When exporting files in *GPS Exchange* (GPX) format, *location-visualizer* will generate, under the GPX document root element (`gpx`) a single track (`trk`) containing a single track segment (`trkseg`) containing all track points (`trkpt`).

Each track point (`trkpt`) will have its mandatory latitude (`lat`) and longutide (`lon`) attributes set and contain a single timestamp (`time`) element node, which in turn contains a text node which consists of a timestamp in the format described by *RFC 3339*.

**Q:** What is the *Records JSON* format?

**A:** *Records JSON* is a JSON-based file format used by many online services (including, but not limited to, Google's "Location History" and provided through its "Takeout" service), as well as some more modern GPS devices and applications in addition to or instead of the older XML-based *GPS Exchange* or GPX file format. It is a de-facto standard format and is **not** to be confused with the *GeoJSON* format described in *RFC 7946*.
