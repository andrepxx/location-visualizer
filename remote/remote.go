package remote

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/andrepxx/location-visualizer/auth/publickey"
	"github.com/andrepxx/location-visualizer/auth/rand"
	"github.com/andrepxx/location-visualizer/remote/multipart"
)

const (
	COMPARISON_RESULT_EQUAL  = 0
	CONTENT_TYPE_ANY         = ""
	CONTENT_TYPE_BINARY      = "application/octet-stream"
	CONTENT_TYPE_CSV         = "text/csv"
	CONTENT_TYPE_JSON        = "application/json"
	CONTENT_TYPE_MULTIPART   = "multipart/form-data"
	CONTENT_TYPE_URLENCODED  = "application/x-www-form-urlencoded"
	HTTP_METHOD_POST         = "POST"
	PEM_TYPE_CERTIFICATE     = "CERTIFICATE"
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
	ImportGeodata(format string, strategy string, data io.ReadSeekCloser) (io.ReadCloser, error)
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
func (this *sessionStruct) ImportGeodata(format string, strategy string, data io.ReadSeekCloser) (io.ReadCloser, error) {
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
		tokenPair := multipart.CreateKeyValuePair("token", encodedToken)
		cgiPair := multipart.CreateKeyValuePair("cgi", "import-geodata")
		formatPair := multipart.CreateKeyValuePair("format", format)
		strategyPair := multipart.CreateKeyValuePair("strategy", strategy)

		/*
		* Create metadata key value pairs.
		 */
		metadata := []multipart.KeyValuePair{
			tokenPair,
			cgiPair,
			formatPair,
			strategyPair,
		}

		fileEntry := multipart.CreateFileEntry("file", "locations", data)

		/*
		* Create file entries.
		 */
		fileEntries := []multipart.FileEntry{
			fileEntry,
		}

		requestData, mimeType := multipart.CreateMultipartProvider(metadata, fileEntries)
		result, errResult = connection.requestMultipart(requestData, mimeType, CONTENT_TYPE_JSON)
	}

	return result, errResult
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
	LoginPrivateKey(name string, privateKey *rsa.PrivateKey) (Session, error)
}

/*
 * Data structure representing a connection to a remote host.
 */
type connectionStruct struct {
	host        string
	port        uint16
	client      *http.Client
	endpointURI string
	userAgent   string
	csprng      io.Reader
}

/*
 * Performs an HTTP POST request.
 *
 * Equivalent to net/http.Post(string, string, io.Reader), but sets "User-Agent" header.
 */
func (this *connectionStruct) post(uri string, contentType string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequest(HTTP_METHOD_POST, uri, body)

	/*
	 * Check if error occured.
	 */
	if err != nil {
		return nil, err
	} else {
		userAgent := this.userAgent
		hdr := request.Header
		hdr.Set("Content-Type", contentType)
		hdr.Set("User-Agent", userAgent)
		client := this.client
		response, err := client.Do(request)
		return response, err
	}

}

/*
 * Performs an HTTP POST request for form data.
 *
 * Equivalent to net/http.PostForm(string, string, io.Reader), but sets "User-Agent" header.
 */
func (this *connectionStruct) postForm(uri string, data url.Values) (*http.Response, error) {
	dataString := data.Encode()
	fd := strings.NewReader(dataString)
	response, err := this.post(uri, CONTENT_TYPE_URLENCODED, fd)
	return response, err
}

/*
 * Perform a POST request sending data and retrieving a response.
 */
func (this *connectionStruct) request(data url.Values, expectedContentType string) (io.ReadCloser, error) {
	result := io.ReadCloser(nil)
	errResult := error(nil)
	host := this.host
	port := this.port
	endpointURI := this.endpointURI
	url := fmt.Sprintf("https://%s:%d%s", host, port, endpointURI)
	resp, err := this.postForm(url, data)

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
 * Perform a multipart POST request sending data and retrieving a response.
 */
func (this *connectionStruct) requestMultipart(data io.Reader, providedContentType string, expectedContentType string) (io.ReadCloser, error) {
	result := io.ReadCloser(nil)
	errResult := error(nil)
	host := this.host
	port := this.port
	endpointURI := this.endpointURI
	url := fmt.Sprintf("https://%s:%d%s", host, port, endpointURI)
	resp, err := this.post(url, providedContentType, data)

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
 * Performs a public-key authentication response, establishing a session and returns a session token.
 */
func (this *connectionStruct) authResponsePublicKey(name string, signature []byte) ([SIZE_KEY_BYTES]byte, error) {
	sessionToken := [SIZE_KEY_BYTES]byte{}
	errResult := error(nil)
	enc := base64.StdEncoding
	encodedSignature := enc.EncodeToString(signature)
	requestData := url.Values{}
	requestData.Set("cgi", "auth-response-public-key")
	requestData.Set("name", name)
	requestData.Set("signature", encodedSignature)
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
 * Logs in at a remote host with an RSA private key, establishing an
 * authenticated session.
 */
func (this *connectionStruct) LoginPrivateKey(name string, privateKey *rsa.PrivateKey) (Session, error) {
	session := Session(nil)
	errResult := error(nil)
	_, nonce, err := this.authRequest(name)

	/*
	 * Check if authentication request was successful.
	 */
	if err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Error during authentication request: %s", msg)
	} else {
		nonceSlice := nonce[:]
		csprng := this.csprng
		sig, err := publickey.SignPSS(nonceSlice, privateKey, csprng)

		/*
		 * Check if signature could be created.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to generate signature: %s", msg)
		} else {
			token, err := this.authResponsePublicKey(name, sig)

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

	}

	return session, errResult
}

/*
 * Creates a new connection to a remote host, expecting a certain certificate chain.
 */
func CreateConnection(host string, port uint16, userAgent string, certificateChain []byte) (Connection, error) {
	result := Connection(nil)
	errResult := error(nil)

	/*
	 * Certificate verification function.
	 */
	v := func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		errResult := error(nil)

		/*
		 * Ensure that the system didn't already verify the certificates.
		 */
		if verifiedChains != nil {
			errResult = fmt.Errorf("%s", "System-side certificate validation shall not occur")
		} else if certificateChain != nil {
			allCerts := []byte{}

			/*
			 * Concatenate all certificates in chain.
			 */
			for _, cert := range rawCerts {
				allCerts = append(allCerts, cert...)
			}

			comparisonResult := bytes.Compare(certificateChain, allCerts)

			/*
			 * If there is a difference, output certificate chain and report error.
			 */
			if comparisonResult != COMPARISON_RESULT_EQUAL {
				errResult = fmt.Errorf("%s", "Certificate mismatch")

				/*
				 * Create PEM block.
				 */
				pemBlock := pem.Block{
					Type:    PEM_TYPE_CERTIFICATE,
					Headers: nil,
					Bytes:   allCerts,
				}

				certificateBytes := pem.EncodeToMemory(&pemBlock)
				certificateString := string(certificateBytes)
				fmt.Printf("%s", certificateString)
			}

		}

		return errResult
	}

	/*
	 * Create TLS configuration.
	 */
	cfg := tls.Config{
		InsecureSkipVerify:    true,
		VerifyPeerCertificate: v,
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

	r := rand.SystemPRNG()
	seed := make([]byte, rand.SEED_SIZE)
	_, err := r.Read(seed)

	/*
	 * Check if seed could be read from system.
	 */
	if err != nil {
		errResult = fmt.Errorf("Failed to obtain entropy from system.")
	} else {
		prng, err := rand.CreatePRNG(seed)

		/*
		 * Check if PRNG could be created.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to create pseudo-random number generator: %s", msg)
		} else {

			/*
			* Create new connection.
			 */
			conn := connectionStruct{
				host:        host,
				port:        port,
				client:      client,
				endpointURI: "/cgi-bin/locviz",
				userAgent:   userAgent,
				csprng:      prng,
			}

			result = &conn
		}

	}

	return result, errResult
}
