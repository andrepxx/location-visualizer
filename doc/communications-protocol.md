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

(**Note:** A token-based authentication method for use by third-party applications with certain API endpoints may be added at some point.)

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
