package remote

import (
	"crypto/sha512"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	CONTENT_TYPE_ANY         = ""
	CONTENT_TYPE_BINARY      = "application/octet-stream"
	CONTENT_TYPE_CSV         = "text/csv"
	CONTENT_TYPE_JSON        = "application/json"
	SIZE_KEY_BYTES           = 64
	TWO_TIMES_SIZE_KEY_BYTES = 2 * SIZE_KEY_BYTES
)

/*
 * Indicates whether a request was successful or not.
 */
type webResponseStruct struct {
	Success bool
	Reason  string
}

/*
 * Web representation of an authentication challenge.
 */
type webAuthChallengeStruct struct {
	webResponseStruct
	Nonce string
	Salt  string
}

/*
 * Web representation of an authentication response.
 */
type webAuthResponseStruct struct {
	Hash string
}

/*
 * Web representation of a session token.
 */
type webTokenStruct struct {
	webResponseStruct
	Token string
}

/*
 * An authenticated session on a remote host.
 */
type Session interface {
	ExportActivityCsv() (io.ReadCloser, error)
	ExportGeodata(format string) (io.ReadCloser, error)
	ImportGeodata(format string, strategy string, data io.ReadSeeker) error
	Logout() error
}

/*
 * Data structure representing an authenticated session at a remote host.
 */
type sessionStruct struct {
	connection *connectionStruct
	token      [SIZE_KEY_BYTES]byte
}

/*
 * Provides an io.ReadCloser exporting activities as CSV.
 */
func (this *sessionStruct) ExportActivityCsv() (io.ReadCloser, error) {
	result := io.ReadCloser(nil)
	errResult := error(nil)
	connection := this.connection

	/*
	 * Check if session is still established.
	 */
	if connection == nil {
		errResult = fmt.Errorf("%s", "No established session")
	} else {
		encoding := base64.StdEncoding
		token := this.token
		tokenSlice := token[:]
		encodedToken := encoding.EncodeToString(tokenSlice)
		requestData := url.Values{}
		requestData.Set("cgi", "export-activity-csv")
		requestData.Set("token", encodedToken)
		response, err := connection.request(requestData, CONTENT_TYPE_CSV)

		/*
		 * Check if an error occured during the request.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Error during TLS request: %s", msg)
		} else {
			result = response
		}

	}

	return result, errResult
}

/*
 * Provides an io.ReadCloser exporting geodata in requested format.
 */
func (this *sessionStruct) ExportGeodata(format string) (io.ReadCloser, error) {
	result := io.ReadCloser(nil)
	errResult := error(nil)
	connection := this.connection

	/*
	 * Check if session is still established.
	 */
	if connection == nil {
		errResult = fmt.Errorf("%s", "No established session")
	} else {
		encoding := base64.StdEncoding
		token := this.token
		tokenSlice := token[:]
		encodedToken := encoding.EncodeToString(tokenSlice)
		requestData := url.Values{}
		requestData.Set("cgi", "export-geodb-content")
		requestData.Set("format", format)
		requestData.Set("token", encodedToken)
		response, err := connection.request(requestData, CONTENT_TYPE_ANY)

		/*
		 * Check if an error occured during the request.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Error during TLS request: %s", msg)
		} else {
			result = response
		}

	}

	return result, errResult
}

/*
 * Imports geodata in specified format from provided io.ReadSeeker.
 */
func (this *sessionStruct) ImportGeodata(format string, strategy string, data io.ReadSeeker) error {
	// TODO: Write code.
	return fmt.Errorf("%s", "Not yet implemented")
}

/*
 * Terminates the session at the remote server.
 */
func (this *sessionStruct) Logout() error {
	errResult := error(nil)
	connection := this.connection

	/*
	 * Check if session is still established.
	 */
	if connection == nil {
		errResult = fmt.Errorf("%s", "No established session")
	} else {
		encoding := base64.StdEncoding
		token := this.token
		tokenSlice := token[:]
		encodedToken := encoding.EncodeToString(tokenSlice)
		requestData := url.Values{}
		requestData.Set("cgi", "auth-logout")
		requestData.Set("token", encodedToken)
		response, err := connection.request(requestData, CONTENT_TYPE_JSON)

		/*
		 * Check if an error occured during the request.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Error during TLS request: %s", msg)
		} else {
			responseData, err := io.ReadAll(response)

			/*
			 * Check if an error occured reading the response.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Error during TLS request: %s", msg)
			} else {
				response := webResponseStruct{}
				err := json.Unmarshal(responseData, &response)

				/*
				 * Check if an error occured while parsing the response.
				 */
				if err != nil {
					msg := err.Error()
					errResult = fmt.Errorf("Error parsing response: %s", msg)
				} else {
					success := response.Success

					/*
					 * If response was not successful, report error, otherwise
					 * invalidate connection.
					 */
					if !success {
						reason := response.Reason
						errResult = fmt.Errorf("Error during logout: %s", reason)
					} else {
						this.connection = nil
					}

				}

			}

		}

	}

	return errResult
}

/*
 * A connection to a remote host.
 */
type Connection interface {
	Login(name string, password string) (Session, error)
}

/*
 * Data structure representing a connection to a remote host.
 */
type connectionStruct struct {
	host        string
	port        uint16
	client      *http.Client
	endpointURI string
}

/*
 * Perform a POST request sending data and retrieving a response.
 */
func (this *connectionStruct) request(data url.Values, expectedContentType string) (io.ReadCloser, error) {
	result := io.ReadCloser(nil)
	errResult := error(nil)
	client := this.client
	host := this.host
	port := this.port
	endpointURI := this.endpointURI
	url := fmt.Sprintf("https://%s:%d%s", host, port, endpointURI)
	resp, err := client.PostForm(url, data)

	/*
	 * Check if an error occured.
	 */
	if err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Error during TLS request: %s", msg)
	} else {
		status := resp.StatusCode
		header := resp.Header
		contentType := header.Get("Content-Type")
		isExpectedContentType := strings.HasPrefix(contentType, expectedContentType)

		/*
		 * Check if status is HTTP 200 OK.
		 */
		if status != http.StatusOK {
			errResult = fmt.Errorf("Error during TLS request: Expected status HTTP 200, but got HTTP %d.", status)
		} else if !isExpectedContentType {
			errResult = fmt.Errorf("Error during TLS request: Expected response to have content type '%s', but actually has '%s'.'", expectedContentType, contentType)
		} else {
			result = resp.Body
		}

	}

	return result, errResult
}

/*
 * Performs an authentication request for a user name and returns salt and nonce.
 */
func (this *connectionStruct) authRequest(name string) ([SIZE_KEY_BYTES]byte, [SIZE_KEY_BYTES]byte, error) {
	salt := [SIZE_KEY_BYTES]byte{}
	nonce := [SIZE_KEY_BYTES]byte{}
	errResult := error(nil)
	requestData := url.Values{}
	requestData.Set("cgi", "auth-request")
	requestData.Set("name", name)
	response, err := this.request(requestData, CONTENT_TYPE_JSON)

	/*
	 * Check if an error occured during the request.
	 */
	if err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Error during TLS request: %s", msg)
	} else {
		responseData, err := io.ReadAll(response)

		/*
		 * Check if an error occured reading the response.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Error during TLS request: %s", msg)
		} else {
			authChallenge := webAuthChallengeStruct{}
			err := json.Unmarshal(responseData, &authChallenge)

			/*
			 * Check if an error occured while parsing the response.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Error parsing response: %s", msg)
			} else {
				success := authChallenge.Success

				/*
				 * Check if challenge could be retrieved.
				 */
				if !success {
					reason := authChallenge.Reason
					errResult = fmt.Errorf("Failed to retrieve challenge: %s", reason)
				} else {
					enc := base64.StdEncoding
					saltString := authChallenge.Salt
					saltStringBytes := []byte(saltString)
					saltSlice := salt[:]
					n, err := enc.Decode(saltSlice, saltStringBytes)

					/*
					 * Check if salt could be decoded.
					 */
					if err != nil {
						errResult = fmt.Errorf("%s", "Failed to decode salt")
					} else if n != SIZE_KEY_BYTES {
						errResult = fmt.Errorf("Size of salt does not match: Expected %d, found %d", SIZE_KEY_BYTES, n)
					} else {
						nonceString := authChallenge.Nonce
						nonceStringBytes := []byte(nonceString)
						nonceSlice := nonce[:]
						n, err := enc.Decode(nonceSlice, nonceStringBytes)

						/*
						 * Check if nonce could be decoded.
						 */
						if err != nil {
							errResult = fmt.Errorf("%s", "Failed to decode nonce")
						} else if n != SIZE_KEY_BYTES {
							errResult = fmt.Errorf("Size of nonce does not match: Expected %d, found %d", SIZE_KEY_BYTES, n)
						}

					}

				}

			}

		}

		err = response.Close()

		/*
		 * Check if an error occured while closing the connection.
		 */
		if (err != nil) && (errResult == nil) {
			msg := err.Error()
			errResult = fmt.Errorf("Error during TLS request: %s", msg)
		}

	}

	return salt, nonce, err
}

/*
 * Performs an authentication response, establishing a session and returns a session token.
 */
func (this *connectionStruct) authResponse(name string, hash [SIZE_KEY_BYTES]byte) ([SIZE_KEY_BYTES]byte, error) {
	sessionToken := [SIZE_KEY_BYTES]byte{}
	errResult := error(nil)
	hashSlice := hash[:]
	enc := base64.StdEncoding
	encodedHash := enc.EncodeToString(hashSlice)
	requestData := url.Values{}
	requestData.Set("cgi", "auth-response")
	requestData.Set("name", name)
	requestData.Set("hash", encodedHash)
	response, err := this.request(requestData, CONTENT_TYPE_JSON)

	/*
	 * Check if an error occured during the request.
	 */
	if err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Error during TLS request: %s", msg)
	} else {
		responseData, err := io.ReadAll(response)

		/*
		 * Check if an error occured reading the response.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Error during TLS request: %s", msg)
		} else {
			token := webTokenStruct{}
			err := json.Unmarshal(responseData, &token)

			/*
			 * Check if an error occured while parsing the response.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Error parsing response: %s", msg)
			} else {
				success := token.Success

				/*
				 * Check if login was successful.
				 */
				if !success {
					reason := token.Reason
					errResult = fmt.Errorf("Error during login process: %s", reason)
				} else {
					enc := base64.StdEncoding
					tokenString := token.Token
					tokenBytes, err := enc.DecodeString(tokenString)

					/*
					 * Check if session token could be decoded.
					 */
					if err != nil {
						msg := err.Error()
						errResult = fmt.Errorf("Error decoding session token: %s", msg)
					} else {
						sessionTokenSlice := sessionToken[:]
						n := copy(sessionTokenSlice, tokenBytes)

						/*
						 * Check if session token was of expected length.
						 */
						if n != SIZE_KEY_BYTES {
							errResult = fmt.Errorf("Session token was not of expected length: Expected %d bytes, got %d.", SIZE_KEY_BYTES, n)
						}

					}

				}

			}

		}

	}

	return sessionToken, errResult
}

/*
 * Logs in at a remote host with user name and password, establishing an
 * authenticated session.
 */
func (this *connectionStruct) Login(name string, password string) (Session, error) {
	session := Session(nil)
	errResult := error(nil)
	salt, nonce, err := this.authRequest(name)

	/*
	 * Check if authentication request was successful.
	 */
	if err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Error during authentication request: %s", msg)
	} else {
		passwordBytes := []byte(password)
		passwordHash := sha512.Sum512(passwordBytes)
		concatSaltAndPasswordHash := [TWO_TIMES_SIZE_KEY_BYTES]byte{}
		copy(concatSaltAndPasswordHash[0:SIZE_KEY_BYTES], salt[:])
		copy(concatSaltAndPasswordHash[SIZE_KEY_BYTES:TWO_TIMES_SIZE_KEY_BYTES], passwordHash[:])
		saltedHash := sha512.Sum512(concatSaltAndPasswordHash[:])
		concatNonceAndSaltedHash := [TWO_TIMES_SIZE_KEY_BYTES]byte{}
		copy(concatNonceAndSaltedHash[0:SIZE_KEY_BYTES], nonce[:])
		copy(concatNonceAndSaltedHash[SIZE_KEY_BYTES:TWO_TIMES_SIZE_KEY_BYTES], saltedHash[:])
		resultingHash := sha512.Sum512(concatNonceAndSaltedHash[:])
		token, err := this.authResponse(name, resultingHash)

		/*
		 * Check if authentication response was successful and a session was established.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Error during authentication response: %s", msg)
		} else {

			/*
			 * Create session.
			 */
			session = &sessionStruct{
				connection: this,
				token:      token,
			}

		}

	}

	return session, errResult
}

/*
 * Creates a new connection to a remote host.
 */
func CreateConnection(host string, port uint16) Connection {

	/*
	 * Create TLS configuration.
	 */
	cfg := tls.Config{
		InsecureSkipVerify: true,
	}

	/*
	 * Create TLS transport.
	 */
	transport := http.Transport{
		TLSClientConfig: &cfg,
	}

	/*
	 * Create TLS client.
	 */
	client := &http.Client{
		Transport: &transport,
	}

	/*
	 * Create new connection.
	 */
	conn := connectionStruct{
		host:        host,
		port:        port,
		client:      client,
		endpointURI: "/cgi-bin/locviz",
	}

	return &conn
}
