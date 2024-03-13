# Data formats used / supported by location-visualizer

This document specifies the data formats supported by *location-visualizer*. It is primarily aimed at developers who wish to interoperate with *location-visualizer*, either by preparing (geographical or activity) data for it to be later stored in the respective database, or to consume data produced by *location-visualizer*, for example for further or custom data analysis or custom filtering, editing and / or merging of that data.

For example, you could use either *R* or *Python* with *Pandas* and *NumPy* to process data exported by *location-visualizer*.

## Geographic data

The following sections describe the data formats *location-visualizer* supports to exchange actual geographical information (timestamps and coordinates) with other devices or applications.

### OpenGeoDB (\*.geodb)

The *OpenGeoDB* file format is a compact binary file format with an open specification that is used to store geographical data and enables fast (random) access. It is also the data format that *location-visualizer* uses internally for its geographical database.

The file format has the capabilities to store versioning information. So far, there is only one version of the file format (version 1.0). The following sections therefore describe the format of files making use of that particular version of the file format. Care must be taken when encountering files with different version information, since these files may not follow the structure described in this document.

An *OpenGeoDB* file consists of a 10 byte header followed by an arbitrary amount of 14 byte records. All numbers are stored in *big-endian* (most significant byte first) or *network byte order*.

#### File header

The header of an *OpenGeoDB* file starts at offset 0, has a size of 10 bytes and has the following structure.

| Offset (bytes) | Size (bytes) | Content | Data type |
| --- | --- | --- | --- |
| 0 | 8 | Magic number `(0x47656f44420a0004)` | 64 bit unsigned integer |
| 8 | 1 | Major version number `(1)` | 8 bit unsigned integer |
| 9 | 1 | Minor version number `(0)` | 8 bit unsigned integer |

#### Records

The records in an *OpenGeoDB* file start at offset $` 10 + (14 \cdot n) `$ for the $` n `$-th entry and each have the following structure.

| Offset (bytes) | Size (bytes) | Content | Data type |
| --- | --- | --- | --- |
| 0 | 6 | Timestamp (in milliseconds since the Epoch) | 48 bit unsigned integer |
| 6 | 4 | Latitude in units of $` 10^{-7 \circ} `$ | 32 bit signed integer |
| 10 | 4 | Longitude in units of $` 10^{-7 \circ} `$ | 32 bit signed integer |

(Offsets are relative to the begin of the respective record.)

Latitudes on the northern hemisphere are considered "positive", while latitudes on the southern hemisphere are considered "negative".

Longitudes east of the zero meridian are considered "positive", while longitudes west of the zero meridian are considered "negative".

### Comma-separated values / RFC 4180 (\*.csv)

Files in *comma separated values* (CSV) file format are text files making use of the *UTF-8* character encoding and Unix line endings (`\n`). Individual records within the files are stored in rows separated by line breaks (`\n`), while individual attributes of each record are stored in fields seprarated by commas (`,`). Individual fields may be enclosed in a pair of quote (`"`) characters and may then include further quotes (which have to be doubled to "escape" them), line breaks, commas, etc. Refer to *RFC 4180* for more information about the CSV format in general.

This section is about the use of the CSV file format for storing and exchanging **coordinate data** between *location-visualizer* and other applications and / or devices. *location-visualizer* also makes use of the CSV file format to exchange activity data. The data format used for this purpose is described in a later section in this document.

Each record in the CSV file has the following fields ("columns") in exactly this order.

1. Timestamp
2. Latitude
3. Longitude

The timestamp is in the format described by *RFC 3339* and has millisecond precision.

Both latitude and longitude are unsigned fixed-point numbers with a variable amount of decimal digits to the left of the decimal separator (`.`) and exactly seven (7) digits to the right of the decimal separator, followed by a single letter describing the direction and thereby replacing the sign. The final letter will always be uppercase in data exported from *location-visualizer*. However, when importing data, *location-visualizer* is a bit more relaxed in the sense that it will also accept the respective lowercase letters in these positions.

Latitudes always have the letters `'N'` or `'S'`, for "north" and "south" (relative to the equator, or as indication of hemisphere), respectively, as the trailing characters in the string, while longitudes always have the letters `'E'` or `'W'`, for "east" and "west" (relative to the zero meridian), respectively, as the trailing characters in the string.

When using **signed** data types, most formats consider northern latitudes and eastern longitudes "positive" (`> 0`), while southern latitudes and western longitudes are considered "negative" (`< 0`). For a zero value, the distinction is arbitrary, but *location-visualizer* will export them as "positive" values, and will therefore indicate `0.0000000N` for a zero latitude and `0.0000000E` for a zero longitude.

When importing data, *location-visualizer* is more tolerant in the sense that it will also accept the letters `'n'` and `'s'` for North and South respectively in latitude values, and the letters `'w'` and `'e'` for West and East respectively in longitude values.

The files produced or expected by *location-visualizer* do **not** include the optional header line that *RFC 4180* describes.

The overall document structure (also compare to other sections) is therefore like this.

```
2024-03-31T17:05:10.125Z,52.5186111N,13.4083333E
...
```

### GPS Exchange (\*.gpx)

When importing files in *GPS Exchange* (GPX) format, *location-visualizer* will only import all track points (`trkpt`) in all track segments (`trkseg`) in all tracks (`trk`) inside the GPX document root element (`gpx`).

When exporting files in *GPS Exchange* (GPX) format, *location-visualizer* will generate, under the GPX document root element (`gpx`) a single track (`trk`) containing a single track segment (`trkseg`) containing all track points (`trkpt`).

Each track point (`trkpt`) will have its mandatory latitude (`lat`) and longutide (`lon`) attributes set and contain a single timestamp (`time`) element node, which in turn contains a text node which consists of a timestamp in the format described by *RFC 3339*.

The overall document structure is therefore like this.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1">
   <trk>
      <trkseg>
         <trkpt lat="52.5186111" lon="13.4083333">
         	<time>
         		2024-03-31T17:05:10.125Z
         	</time>
         </trkpt>
         <!-- More track points here ... -->
      </trkseg>
   </trk>
</gpx>
```

### Records JSON (\*.json)

*Records JSON* is a JSON-based file format used by many online services and some more modern GPS devices and applications in addition to or instead of the older XML-based *GPS Exchange* or GPX file format. It is a de-facto standard format and is **not** to be confused with the *GeoJSON* format described in *RFC 7946*.

A *Records JSON* file consists of a single root object with the following structure.

```C++
struct root_object {
	location_object[] locations;
};
```

Where each `location_object` in turn is a data structure defined like this.

```C++
struct location_object {
	string timestamp;
	string timestampMs;
	int32_t latitudeE7;
	int32_t longitudeE7;
};
```

The `timestamp` field, if present, holds a timestamp in *RFC 3339* format describing the point in time where this position was acquired. The `timestampMs` field, if present, holds an equivalent timestamp in milliseconds since the Epoch instead, represented as an integer in decimal encoding. When both `timestamp` and `timestampMs` are given, `timestampMs` takes precedence when importing data into the geographical database. Upon export to *Records JSON* format, *location-visualizer* will populate both the `timestamp` and `timestampMs` fields.

The `latitudeE7` and `longitudeE7` fields represent latitude and longitude values stored as signed integer values in units of $` 10^{-7 \circ} `$.

The overall document structure is therefore like this.

```json
{
   "locations": [
      {
         "timestamp": "2024-03-31T17:05:10.125Z",
         "timestampMs": "1711897510125",
         "latitudeE7": 525186111,
         "longitudeE7": 134083333
      },
      ...
   ]
}
```

## Activity data

The following section describes the data format *location-visualizer* supports to exchange activity data with other devices or applications.

### Comma-separated values / RFC 4180 (\*.csv)

Files in *comma separated values* (CSV) file format are text files making use of the *UTF-8* character encoding and Unix line endings (`\n`). Individual records within the files are stored in rows separated by line breaks (`\n`), while individual attributes of each record are stored in fields seprarated by commas (`,`). Individual fields may be enclosed in a pair of quote (`"`) characters and may then include further quotes (which have to be doubled to "escape" them), line breaks, commas, etc. Refer to *RFC 4180* for more information about the CSV format in general.

This section is about the use of the CSV file format for storing and exchanging **activity data** between *location-visualizer* and other applications and / or devices. *location-visualizer* also makes use of the CSV file format to exchange coordinate data. The data format used for this purpose is described in a previous section in this document.

Each record in the CSV file describes what we call an **activity group** and has the following fields ("columns") in exactly this order.

1. Timestamp of the beginning of this activity group
2. Weight in \[kg\] at the beginning of this activity group, as fixed-point number with 1 decimal place to the right of the decimal point
3. Duration of time spent **running** (e. g. `1h30m15s` for "1 hour, 30 minutes, 15 seconds")
4. Distance in \[km\] covered while **running**, as fixed-point number with 1 decimal place to the right of the decimal point
5. Number of steps taken while **running**, as (unsigned) integer
6. Amount of energy in \[kJ\] consumed while **running**, as (unsigned) integer
7. Duration of time spent **cycling** (e. g. `1h30m15s` for "1 hour, 30 minutes, 15 seconds")
8. Distance in \[km\] covered while **cycling**, as fixed-point number with 1 decimal place to the right of the decimal point
9. Amount of energy in \[kJ\] consumed while **cycling**, as (unsigned) integer
10. Amount of energy in \[kJ\] consumed while performing **other** activities, as (unsigned) integer

The timestamp describing the beginning of an activity group is in the format described by *RFC 3339* and has millisecond precision. Activity groups are considered consecutive and without gaps. They subdivide the entire timeline (of GPS data) into segments. Therefore, they don't have an explicit "end" timestamp. Rather, each activity group is considered to end when the subsequent activity group begins. The last activity group is implicitly (and rather arbitrarily) considered to end 24 hours after it began.

An activity group is therefore basically a section of time (in which activities are performed). For me personally, each **day** is an activity group. It's not necessarily 24 hours, since there might be time zone changes - for example when I travel or due to daylight saving changes. However, the concept of activity groups itself is flexible and you can basically define an activity group to be whatever timespan you want. For example, when you run a marathon on a certain day, you could easily define the time before the run, the actual running time, and the time after the run, as seperate activity groups.

The "other" activity serves to capture the energy consumption you have during the time within each activity group that you spend neither running or cycling. (For example, you could be eating or sleeping or dancing or driving.)
