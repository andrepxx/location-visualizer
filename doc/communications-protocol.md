# Protocol for communication between location-visualizer and client applications

This document describes the communications protocol that is used between *location-visualizer* and client applications, like the web-based UI client or official CLI client that this project provides. Implementers of third-party applications who want to integrate with *location-visualizer*, for example for uploading location data, can refer to this document.

Note that even though the protocol is documented here, it is not considered an open API, but rather an internal implementation detail of *location-visualizer*, which means that it may change at any time and in non-backward compatible ways.

## Transport protocols and API endpoint

The API endpoint you will talk to is `/cgi-bin/locviz`.

As a sidenote: Please don't pay too much attention to the term *CGI* appearing all over the place. The application technically doesn't make use of CGI anywhere. When it appears inside of messages, the appropriate term would probably be something like *action*.

As long as *location-visualizer* runs with its default configuration, you can only talk to the server via TLS (HTTPS), by default on port 8443. It also has a non-TLS (HTTP) port open, by default port 8080, but all it will do is redirect your client to the TLS part.

There is a configuration option called *TLSDisabled* in `config/config.json`, which I added since some people had very specific use cases, like operating *location-visualizer* behind a reverse proxy. However, it'd **strongly** advise **against** it for general use. The "official" CLI client also communicates exclusively via TLS and does **not** support "plain" HTTP.

Most calls can be done via GET or POST method and with either "application/x-www-form-urlencoded" or "multipart/form-data" encoding, but the convention is (and the "official" client does it this way) to always use POST, use "multipart/form-data" when submitting files (no other choice) and use "application/x-www-form-urlencoded" otherwise.

## Authentication protocol

*location-visualizer* is designed to use **strong, challenge-response based authentication** and a client will have to get past this authentication before it is authorized to access any data.

(**Note:** A token-based authentication method for use by third-party applications is also available for submission of coordinate data. Please see the section below for more details.)

**1. Send an authentication request to the server**

Request: `cgi=auth-request&name=[your username]`

The response will be an "application/json", which looks like this:

```json
{
	"Success": true,
	"Reason": "",
	"Nonce": "[64 bytes of Base64 encoded stuff]",
	"Salt": "[64 bytes of Base64 encoded stuff]"
}
```

The reason will be different from "" if success == false and will contain a natural-language explanation of what went wrong.

**2. Calculate the authentication response**

If your user entered a password, the authentication response will be H(nonce . H(salt . H(password))), where H is the SHA-512 function and "." is concatenation of the byte streams.

Note that the nonce and salt come from the server Base64 encoded, but you will have to decode them and obtain the raw bytes before doing the above calculation / hashing. Calculate the resulting hash and Base64 encode it before sending it back to the server in an authentication response.

**3. Send the authentication response to the server**

Request: `cgi=auth-response&name=[your username]&hash=[the authentication response]`

The response will be an "application/json", which looks like this:

```json
{
	"Success": true,
	"Reason": "",
	"Token": "[64 bytes of Base64 encoded stuff]"
}
```

This token is what you will need for authorization. You can just save it as a string, or you can Base64-decode it and save the binary data it represents and re-encode it on demand. (The official client does the latter, which is a bit more "strict" / "validating".)

**4. Perform whatever actions you need to perform**

For example, import / upload or export / download coordinate data, activity data, etc.

**5. Terminate the session**

Finally, after the upload, you should terminate the session again.

Request: `cgi=auth-logout&token=[the session token]`

Hopefully, this will return the following.

```json
{
	"Success": true,
	"Reason": ""
}
```

## Uploading coordinate data

You can use an authenticated session to upload coordiante data to the server.

To do so, you currently **have** to use "multipart/form-data" encoding.

You have to encode the following fields.

```
"token": [the session token]
"cgi": "import-geodata"
"format": [one of "opengeodb", "csv", "gpx" or "json"]
"strategy": [one of "all", "newer" or "none"]
"file": [the data, in the specified format, as a file upload per the MIME specification]
```

If you only want to post a single location, then CSV might be a good choice for that format, since it doesn't contain any headers / trailers, so you may just submit a single line or an arbitrary number of lines.

The server will respond with a rather large and complex, JSON-based "import report", in which it tries to explain how it merged the data you submitted into its current dataset. If you don't care about the details, just ignore it - except perhaps the boolean indicating whether the import was successful at all.

## Alternative submission of coordinate data for third-party applications

For third-party clients or endpoints, which do not implement the full challenge-response based authentication and application protocol, there is a special "CGI" (action) called `submit-coordinates`, which allows authentication by device tokens. Such endpoints are typically IoT devices or other embedded systems like sensors deployed in the field, but also third-party mobile applications or message brokers.

Device tokens are static secrets of (currently) 64 bit size, which are sent along a request from an endpoint. You can think of a device token as a "static" session id for a session, which does not expire (unless the device token is explicitly revoked). Device tokens are stored and displayed and transmitted to the server in hexadecimal encoded.

Device tokens are bound to the users, for which they are generated. Each user can have an arbitrary number of device tokens attached. *location-visualizer* stores metadata for each device token, such as its creation time and a textual description, which allows for identification of the device the token was issued for. Device tokens can be individually revoked (removed from the user account) in case they get compromised.

A size of 64 bits is considered insufficient for cryptographic keys. However, device tokens only serve as a means of authenticating the submitting device or sensor to the data recipient. A rogue client submitting requests to an instance of *location-visualizer* every 5 milliseconds would need, on average, 35 billion years, to randomly guess a 64 bit token and successfully submit a fake measurement.

To submit data to *location-visualizer* via an API token, send the following request to the API endpoint.

Request: `cgi=submit-coordinates&name=[your username]&devicetoken=[the device token]&time=[timestamp]&latitude=[latitude]&longitude=[longitude]`

Provide the name of the user which provisioned the device token data submitted by the sensor will be considered as being uploaded by the respective user, which means that this user must have the `geodb-write` permission for the submission to succeed.

The device token must be transmitted in hexadecimal encoding.

The timestamp must be in the format described by *RFC 3339* and has millisecond precision.

Latitude and longitude values must be in decimal representation, using '.' as the decimal separator. Longitudes on the northern hemisphere (north of the equator) are considered "positive", while longitudes on the souther hemisphere (south of the equator) are considered "negative". Latitudes east of the zero meridian are considered "positive", while latitudes west of the zero meridian are considered "positive". Negative values are prefixed by unary minus (`-`), while positive values have no prefix. A zero longitude or latitude is, by convention, considered "positive".

Longitudes and latitudes are stored with a resolution of $ 10^{-7} $ degrees. Values with less than seven digits after the decimal separator are filled up with zeros after their last digit. Values with more than seven digits after the decimal separator are truncated (**not** rounded) to a precision of seven digits after the decimal separator. To achieve optimal precision when submitting data to *location-visualizer*, consider rounding sensor data and storing the result in a fixed-point representation on your sensor platform.