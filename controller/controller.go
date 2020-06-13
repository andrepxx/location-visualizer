package controller

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/andrepxx/location-visualizer/auth/rand"
	"github.com/andrepxx/location-visualizer/auth/session"
	"github.com/andrepxx/location-visualizer/auth/user"
	"github.com/andrepxx/location-visualizer/filter"
	"github.com/andrepxx/location-visualizer/geo"
	"github.com/andrepxx/location-visualizer/tile"
	"github.com/andrepxx/location-visualizer/webserver"
	"github.com/andrepxx/sydney/color"
	"github.com/andrepxx/sydney/coordinates"
	"github.com/andrepxx/sydney/scene"
	"image/png"
	"io/ioutil"
	"math"
	"os"
	"runtime"
	"strconv"
	"time"
)

/*
 * Constants for the controller.
 */
const (
	CONFIG_PATH                    = "config/config.json"
	PERMISSIONS_USERDB os.FileMode = 0644
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
 * The configuration for the controller.
 */
type configStruct struct {
	GeoData       string
	MapServer     string
	MapCache      string
	SessionExpiry string
	UseMap        bool
	UserDB        string
	WebServer     webserver.Config
}

/*
 * The controller for the DSP.
 */
type controllerStruct struct {
	config         configStruct
	data           []geo.Location
	tileSource     tile.Source
	userDBPath     string
	userManager    user.Manager
	sessionManager session.Manager
}

/*
 * The controller interface.
 */
type Controller interface {
	Operate(args []string)
	Prefetch(zoomLevel uint8)
}

/*
 * Marshals an object into a JSON representation or an error.
 * Returns the appropriate MIME type and binary representation.
 */
func (this *controllerStruct) createJSON(obj interface{}) (string, []byte) {
	buffer, err := json.MarshalIndent(obj, "", "\t")

	/*
	 * Check if we got an error during marshalling.
	 */
	if err != nil {
		conf := this.config
		confServer := conf.WebServer
		contentType := confServer.ErrorMime
		errString := err.Error()
		bufString := bytes.NewBufferString(errString)
		bufBytes := bufString.Bytes()
		return contentType, bufBytes
	} else {
		return "application/json; charset=utf-8", buffer
	}

}

/*
 * Check permission of a certain session.
 */
func (this *controllerStruct) checkPermission(encodedToken string, permission string) (bool, error) {
	enc := base64.StdEncoding
	tokenBuffer, err := enc.DecodeString(encodedToken)

	/*
	 * Check if token could be decoded.
	 */
	if err != nil {
		return false, fmt.Errorf("%s", "Failed to decode session token.")
	} else {
		sm := this.sessionManager
		t := sm.CreateToken(tokenBuffer)
		name, err := sm.UserName(t)

		/*
		 * Check if session could be found
		 */
		if err != nil {
			return false, err
		} else {
			um := this.userManager
			permitted, err := um.HasPermission(name, permission)
			return permitted, err
		}

	}

}

/*
 * Client requests to terminate a session.
 */
func (this *controllerStruct) authLogoutHandler(request webserver.HttpRequest) webserver.HttpResponse {
	enc := base64.StdEncoding
	tokenIn := request.Params["token"]
	tokenBuffer, err := enc.DecodeString(tokenIn)
	wr := webResponseStruct{}

	/*
	 * Check if token could be decoded.
	 */
	if err != nil {

		/*
		 * Indicate failure.
		 */
		wr = webResponseStruct{
			Success: false,
			Reason:  "Failed to decode session token.",
		}

	} else {
		sm := this.sessionManager
		token := sm.CreateToken(tokenBuffer)
		err = sm.Terminate(token)

		/*
		 * Check if session was terminated.
		 */
		if err != nil {
			msg := err.Error()
			reason := fmt.Sprintf("Failed to terminate session: %s", msg)

			/*
			 * Indicate failure.
			 */
			wr = webResponseStruct{
				Success: false,
				Reason:  reason,
			}

		} else {

			/*
			 * Indicate success.
			 */
			wr = webResponseStruct{
				Success: true,
				Reason:  "",
			}

		}

	}

	mimeType, buffer := this.createJSON(wr)

	/*
	 * Create HTTP response.
	 */
	response := webserver.HttpResponse{
		Header: map[string]string{"Content-type": mimeType},
		Body:   buffer,
	}

	return response
}

/*
 * Client requests to obtain a challenge to authenticate as a user.
 */
func (this *controllerStruct) authRequestHandler(request webserver.HttpRequest) webserver.HttpResponse {
	name := request.Params["name"]
	wac := webAuthChallengeStruct{}
	sm := this.sessionManager
	c, err := sm.Challenge(name)

	/*
	 * Check if challenge was created.
	 */
	if err != nil {
		msg := err.Error()
		reason := fmt.Sprintf("Failed to create challenge: %s", msg)

		/*
		 * Indicate failure.
		 */
		wac = webAuthChallengeStruct{

			webResponseStruct: webResponseStruct{
				Success: false,
				Reason:  reason,
			},

			Nonce: "",
			Salt:  "",
		}

	} else {
		enc := base64.StdEncoding
		nonce := c.Nonce()
		salt := c.Salt()
		nonceString := enc.EncodeToString(nonce[:])
		saltString := enc.EncodeToString(salt[:])

		/*
		 * Create authentication challenge.
		 */
		wac = webAuthChallengeStruct{

			webResponseStruct: webResponseStruct{
				Success: true,
				Reason:  "",
			},

			Nonce: nonceString,
			Salt:  saltString,
		}

	}

	mimeType, buffer := this.createJSON(wac)

	/*
	 * Create HTTP response.
	 */
	response := webserver.HttpResponse{
		Header: map[string]string{"Content-type": mimeType},
		Body:   buffer,
	}

	return response
}

/*
 * Client sends authentication response to obtain session token.
 */
func (this *controllerStruct) authResponseHandler(request webserver.HttpRequest) webserver.HttpResponse {
	enc := base64.StdEncoding
	name := request.Params["name"]
	hashIn := request.Params["hash"]
	responseToken := webTokenStruct{}
	hash, err := enc.DecodeString(hashIn)

	/*
	 * Check if hash could be decoded.
	 */
	if err != nil {

		/*
		 * Indicate failure.
		 */
		responseToken = webTokenStruct{

			webResponseStruct: webResponseStruct{
				Success: false,
				Reason:  "Failed to decode hash value.",
			},

			Token: "",
		}

	} else {
		sm := this.sessionManager
		t, err := sm.Response(name, hash)

		/*
		 * Check if session was created.
		 */
		if err != nil {
			msg := err.Error()
			reason := fmt.Sprintf("Failed to create session: %s", msg)

			/*
			 * Indicate failure.
			 */
			responseToken = webTokenStruct{

				webResponseStruct: webResponseStruct{
					Success: false,
					Reason:  reason,
				},

				Token: "",
			}

		} else {
			token := t.Token()
			tokenString := enc.EncodeToString(token[:])

			/*
			 * Create data structure for session token.
			 */
			responseToken = webTokenStruct{

				webResponseStruct: webResponseStruct{
					Success: true,
					Reason:  "",
				},

				Token: tokenString,
			}

		}

	}

	mimeType, buffer := this.createJSON(responseToken)

	/*
	 * Create HTTP response.
	 */
	response := webserver.HttpResponse{
		Header: map[string]string{"Content-type": mimeType},
		Body:   buffer,
	}

	return response
}

/*
 * Renders a map tile.
 */
func (this *controllerStruct) getTileHandler(request webserver.HttpRequest) webserver.HttpResponse {
	token := request.Params["token"]
	perm, err := this.checkPermission(token, "get-tile")

	/*
	 * Check permissions
	 */
	if err != nil {
		msg := err.Error()
		customMsg := fmt.Sprintf("Failed to check permission: %s\n", msg)
		customMsgBuf := bytes.NewBufferString(customMsg)
		customMsgBytes := customMsgBuf.Bytes()
		conf := this.config
		confServer := conf.WebServer
		contentType := confServer.ErrorMime

		/*
		 * Create HTTP response.
		 */
		response := webserver.HttpResponse{
			Header: map[string]string{"Content-type": contentType},
			Body:   customMsgBytes,
		}

		return response
	} else if !perm {
		customMsg := fmt.Sprintf("%s", "Forbidden!")
		customMsgBuf := bytes.NewBufferString(customMsg)
		customMsgBytes := customMsgBuf.Bytes()
		conf := this.config
		confServer := conf.WebServer
		contentType := confServer.ErrorMime

		/*
		 * Create HTTP response.
		 */
		response := webserver.HttpResponse{
			Header: map[string]string{"Content-type": contentType},
			Body:   customMsgBytes,
		}

		return response
	} else {
		xIn := request.Params["x"]
		x64, _ := strconv.ParseUint(xIn, 10, 32)
		x := uint32(x64)
		yIn := request.Params["y"]
		y64, _ := strconv.ParseUint(yIn, 10, 32)
		y := uint32(y64)
		zIn := request.Params["z"]
		z64, _ := strconv.ParseUint(zIn, 10, 8)
		z := uint8(z64)
		colorScaleIn := request.Params["colorscale"]
		colorScale64, colorScaleErr := strconv.ParseUint(colorScaleIn, 10, 8)
		colorScaleFloat := 0.5

		/*
		 * Check if color scale is valid and in range.
		 */
		if colorScaleErr == nil && colorScale64 <= 10 {
			colorScaleFloat = 0.1 * float64(colorScale64)
		}

		tileSource := this.tileSource
		t, err := tileSource.Get(z, x, y, colorScaleFloat)

		/*
		 * Check if tile could be fetched.
		 */
		if err != nil {
			msg := err.Error()
			customMsg := fmt.Sprintf("Failed to fetch map tile: %s\n", msg)
			customMsgBuf := bytes.NewBufferString(customMsg)
			customMsgBytes := customMsgBuf.Bytes()
			conf := this.config
			confServer := conf.WebServer
			contentType := confServer.ErrorMime

			/*
			 * Create HTTP response.
			 */
			response := webserver.HttpResponse{
				Header: map[string]string{"Content-type": contentType},
				Body:   customMsgBytes,
			}

			return response
		} else {
			img := t.Image()
			id := t.Id()
			idX := id.X()
			idY := id.Y()
			idZ := id.Zoom()

			/*
			 * Ensure that the tile IDs match.
			 */
			if (x != idX) || (y != idY) || (z != idZ) {
				msg := "Something is wrong here: (%d, %d, %d) != (%d, %d, %d)"
				customMsg := fmt.Sprintf(msg, idX, idY, idZ, x, y, z)
				customMsgBuf := bytes.NewBufferString(customMsg)
				customMsgBytes := customMsgBuf.Bytes()
				conf := this.config
				confServer := conf.WebServer
				contentType := confServer.ErrorMime

				/*
				 * Create HTTP response.
				 */
				response := webserver.HttpResponse{
					Header: map[string]string{"Content-type": contentType},
					Body:   customMsgBytes,
				}

				return response
			} else {

				/*
				 * Create a PNG encoder.
				 */
				encoder := png.Encoder{
					CompressionLevel: png.BestCompression,
				}

				buf := &bytes.Buffer{}
				err := encoder.Encode(buf, img)

				/*
				 * Check if image could be encoded.
				 */
				if err != nil {
					msg := err.Error()
					customMsg := fmt.Sprintf("Failed to encode image: %s\n", msg)
					customMsgBuf := bytes.NewBufferString(customMsg)
					customMsgBytes := customMsgBuf.Bytes()
					conf := this.config
					confServer := conf.WebServer
					contentType := confServer.ErrorMime

					/*
					 * Create HTTP response.
					 */
					response := webserver.HttpResponse{
						Header: map[string]string{"Content-type": contentType},
						Body:   customMsgBytes,
					}

					return response
				} else {
					bufBytes := buf.Bytes()

					/*
					 * Create HTTP response.
					 */
					response := webserver.HttpResponse{
						Header: map[string]string{"Content-type": "image/png"},
						Body:   bufBytes,
					}

					return response
				}

			}

		}

	}

}

/*
 * Renders location data into an image.
 */
func (this *controllerStruct) renderHandler(request webserver.HttpRequest) webserver.HttpResponse {
	token := request.Params["token"]
	perm, err := this.checkPermission(token, "render")

	/*
	 * Check permissions
	 */
	if err != nil {
		msg := err.Error()
		customMsg := fmt.Sprintf("Failed to check permission: %s\n", msg)
		customMsgBuf := bytes.NewBufferString(customMsg)
		customMsgBytes := customMsgBuf.Bytes()
		conf := this.config
		confServer := conf.WebServer
		contentType := confServer.ErrorMime

		/*
		 * Create HTTP response.
		 */
		response := webserver.HttpResponse{
			Header: map[string]string{"Content-type": contentType},
			Body:   customMsgBytes,
		}

		return response
	} else if !perm {
		customMsg := fmt.Sprintf("%s", "Forbidden!")
		customMsgBuf := bytes.NewBufferString(customMsg)
		customMsgBytes := customMsgBuf.Bytes()
		conf := this.config
		confServer := conf.WebServer
		contentType := confServer.ErrorMime

		/*
		 * Create HTTP response.
		 */
		response := webserver.HttpResponse{
			Header: map[string]string{"Content-type": contentType},
			Body:   customMsgBytes,
		}

		return response
	} else {
		xresIn := request.Params["xres"]
		xres64, _ := strconv.ParseUint(xresIn, 10, 16)
		xres := uint32(xres64)
		yresIn := request.Params["yres"]
		yres64, _ := strconv.ParseUint(yresIn, 10, 16)
		yres := uint32(yres64)
		xposIn := request.Params["xpos"]
		xpos, _ := strconv.ParseFloat(xposIn, 64)
		yposIn := request.Params["ypos"]
		ypos, _ := strconv.ParseFloat(yposIn, 64)
		zoomIn := request.Params["zoom"]
		zoom, _ := strconv.ParseUint(zoomIn, 10, 8)
		zoomFloat := float64(zoom)
		zoomExp := -0.2 * zoomFloat
		zoomFac := math.Pow(2.0, zoomExp)
		minTimeIn := request.Params["mintime"]
		minTime, _ := filter.ParseTime(minTimeIn)
		maxTimeIn := request.Params["maxtime"]
		maxTime, _ := filter.ParseTime(maxTimeIn)
		fgColor := request.Params["fgcolor"]
		spreadIn := request.Params["spread"]
		spread64, _ := strconv.ParseUint(spreadIn, 10, 8)
		spread := uint8(spread64)
		halfWidth := 0.5 * zoomFac
		xresFloat := float64(xres)
		yresFloat := float64(yres)
		aspectRatio := yresFloat / xresFloat
		halfHeight := aspectRatio * halfWidth
		minX := xpos - halfWidth
		maxX := xpos + halfWidth
		minY := ypos - halfHeight
		maxY := ypos + halfHeight
		scn := scene.Create(xres, yres, minX, maxX, minY, maxY)
		filteredData := this.data
		minTimeIsZero := minTime.IsZero()
		maxTimeIsZero := maxTime.IsZero()

		/*
		 * Apply filter if at least one of the limits is set.
		 */
		if !minTimeIsZero || !maxTimeIsZero {
			flt := filter.Time(minTime, maxTime)
			filteredData = filter.Apply(flt, filteredData)
		}

		numDataPoints := len(filteredData)
		projectedData := make([]coordinates.Cartesian, numDataPoints)

		/*
		 * Obtain projected data points.
		 */
		for i, elem := range filteredData {
			projectedData[i] = elem.Projected()
		}

		scn.Aggregate(projectedData)
		scn.Spread(spread)
		mapping := color.DefaultMapping()

		/*
		 * Check if custom color mapping is required.
		 */
		switch fgColor {
		case "red":
			mapping = color.SimpleMapping(255, 0, 0)
		case "green":
			mapping = color.SimpleMapping(0, 255, 0)
		case "blue":
			mapping = color.SimpleMapping(0, 0, 255)
		case "yellow":
			mapping = color.SimpleMapping(255, 255, 0)
		case "cyan":
			mapping = color.SimpleMapping(0, 255, 255)
		case "magenta":
			mapping = color.SimpleMapping(255, 0, 255)
		case "gray":
			mapping = color.SimpleMapping(127, 127, 127)
		case "brightblue":
			mapping = color.SimpleMapping(127, 127, 255)
		case "white":
			mapping = color.SimpleMapping(255, 255, 255)
		}

		target, err := scn.Render(mapping)

		/*
		 * Check if image could be rendered.
		 */
		if err != nil {
			msg := err.Error()
			customMsg := fmt.Sprintf("Failed to render image: %s", msg)
			customMsgBuf := bytes.NewBufferString(customMsg)
			customMsgBytes := customMsgBuf.Bytes()
			conf := this.config
			confServer := conf.WebServer
			contentType := confServer.ErrorMime

			/*
			 * Create HTTP response.
			 */
			response := webserver.HttpResponse{
				Header: map[string]string{"Content-type": contentType},
				Body:   customMsgBytes,
			}

			return response
		} else {

			/*
			 * Create a PNG encoder.
			 */
			encoder := png.Encoder{
				CompressionLevel: png.BestCompression,
			}

			buf := &bytes.Buffer{}
			err := encoder.Encode(buf, target)

			/*
			 * Check if image could be encoded.
			 */
			if err != nil {
				msg := err.Error()
				customMsg := fmt.Sprintf("Failed to encode image: %s\n", msg)
				customMsgBuf := bytes.NewBufferString(customMsg)
				customMsgBytes := customMsgBuf.Bytes()
				conf := this.config
				confServer := conf.WebServer
				contentType := confServer.ErrorMime

				/*
				 * Create HTTP response.
				 */
				response := webserver.HttpResponse{
					Header: map[string]string{"Content-type": contentType},
					Body:   customMsgBytes,
				}

				return response
			} else {
				bufBytes := buf.Bytes()

				/*
				 * Create HTTP response.
				 */
				response := webserver.HttpResponse{
					Header: map[string]string{"Content-type": "image/png"},
					Body:   bufBytes,
				}

				return response
			}

		}

	}

}

/*
 * Handles CGI requests that could not be dispatched to other CGIs.
 */
func (this *controllerStruct) errorHandler(request webserver.HttpRequest) webserver.HttpResponse {
	conf := this.config
	confServer := conf.WebServer
	contentType := confServer.ErrorMime
	msgBuf := bytes.NewBufferString("This CGI call is not implemented.")
	msgBytes := msgBuf.Bytes()

	/*
	 * Create HTTP response.
	 */
	response := webserver.HttpResponse{
		Header: map[string]string{"Content-type": contentType},
		Body:   msgBytes,
	}

	return response
}

/*
 * Dispatch CGI requests to the corresponding CGI handlers.
 */
func (this *controllerStruct) dispatch(request webserver.HttpRequest) webserver.HttpResponse {
	cgi := request.Params["cgi"]
	response := webserver.HttpResponse{}

	/*
	 * Find the right CGI to handle the request.
	 */
	switch cgi {
	case "auth-logout":
		response = this.authLogoutHandler(request)
	case "auth-request":
		response = this.authRequestHandler(request)
	case "auth-response":
		response = this.authResponseHandler(request)
	case "get-tile":
		response = this.getTileHandler(request)
	case "render":
		response = this.renderHandler(request)
	default:
		response = this.errorHandler(request)
	}

	return response
}

/*
 * Synchronize user database to disk.
 */
func (this *controllerStruct) syncUserDB() error {
	mgr := this.userManager
	buf, err := mgr.Export()

	/*
	 * Check if export failed.
	 */
	if err != nil {
		msg := err.Error()
		return fmt.Errorf("Error serializing user database: %s", msg)
	} else {
		path := this.userDBPath
		mode := os.ModeExclusive | (os.ModePerm & PERMISSIONS_USERDB)
		err := ioutil.WriteFile(path, buf, mode)

		/*
		 * Check if something went wrong
		 */
		if err != nil {
			msg := err.Error()
			return fmt.Errorf("Error synchronizing user database: %s", msg)
		} else {
			return nil
		}

	}

}

/*
 * Interpret user commands entered into shell.
 */
func (this *controllerStruct) interpret(args []string) {
	numArgs := len(args)

	/*
	 * Ensure that there is at least one argument.
	 */
	if numArgs > 0 {
		cmd := args[0]
		umgr := this.userManager

		/*
		 * Perform action based on command.
		 */
		switch cmd {
		case "add-permission":

			/*
			 * Check number of arguments.
			 */
			if numArgs != 3 {
				fmt.Printf("Command '%s' expects 2 additional arguments: name, permission\n", cmd)
			} else {
				name := args[1]
				permission := args[2]
				err := umgr.AddPermission(name, permission)

				/*
				 * Check if something went wrong.
				 */
				if err != nil {
					msg := err.Error()
					fmt.Printf("Command '%s' failed: %s\n", cmd, msg)
				} else {
					err = this.syncUserDB()

					/*
					 * Check if something went wrong.
					 */
					if err != nil {
						msg := err.Error()
						fmt.Printf("%s\n", msg)
					}

				}

			}

		case "clear-password":

			/*
			 * Check number of arguments.
			 */
			if numArgs != 2 {
				fmt.Printf("Command '%s' expects 1 additional argument: name\n", cmd)
			} else {
				name := args[1]
				err := umgr.SetPassword(name, "")

				/*
				 * Check if something went wrong.
				 */
				if err != nil {
					msg := err.Error()
					fmt.Printf("Command '%s' failed: %s\n", cmd, msg)
				} else {
					err = this.syncUserDB()

					/*
					 * Check if something went wrong.
					 */
					if err != nil {
						msg := err.Error()
						fmt.Printf("%s\n", msg)
					}

				}

			}

		case "create-user":

			/*
			 * Check number of arguments.
			 */
			if numArgs != 2 {
				fmt.Printf("Command '%s' expects 1 additional argument: name\n", cmd)
			} else {
				name := args[1]
				err := umgr.CreateUser(name)

				/*
				 * Check if something went wrong.
				 */
				if err != nil {
					msg := err.Error()
					fmt.Printf("Command '%s' failed: %s\n", cmd, msg)
				} else {
					err = this.syncUserDB()

					/*
					 * Check if something went wrong.
					 */
					if err != nil {
						msg := err.Error()
						fmt.Printf("%s\n", msg)
					}

				}

			}

		case "has-permission":

			/*
			 * Check number of arguments.
			 */
			if numArgs != 3 {
				fmt.Printf("Command '%s' expects 2 additional arguments: name, permission\n", cmd)
			} else {
				name := args[1]
				permission := args[2]
				result, err := umgr.HasPermission(name, permission)

				/*
				 * Check if something went wrong.
				 */
				if err != nil {
					msg := err.Error()
					fmt.Printf("Command '%s' failed: %s\n", cmd, msg)
				} else {
					resultString := strconv.FormatBool(result)
					fmt.Printf("%s\n", resultString)
				}

			}

		case "list-permissions":

			/*
			 * Check number of arguments.
			 */
			if numArgs != 2 {
				fmt.Printf("Command '%s' expects 1 additional argument: name\n", cmd)
			} else {
				name := args[1]
				permissions, err := umgr.Permissions(name)

				/*
				 * Check if something went wrong.
				 */
				if err != nil {
					msg := err.Error()
					fmt.Printf("Command '%s' failed: %s\n", cmd, msg)
				} else {

					/*
					 * Print each permission on a new line.
					 */
					for _, permission := range permissions {
						fmt.Printf("%s\n", permission)
					}

				}

			}

		case "list-users":

			/*
			 * Check number of arguments.
			 */
			if numArgs != 1 {
				fmt.Printf("Command '%s' expects no additional arguments.\n", cmd)
			} else {
				users := umgr.Users()

				/*
				 * Print each user on a new line.
				 */
				for _, user := range users {
					fmt.Printf("%s\n", user)
				}

			}

		case "remove-permission":

			/*
			 * Check number of arguments.
			 */
			if numArgs != 3 {
				fmt.Printf("Command '%s' expects 2 additional arguments: name, permission\n", cmd)
			} else {
				name := args[1]
				permission := args[2]
				err := umgr.RemovePermission(name, permission)

				/*
				 * Check if something went wrong.
				 */
				if err != nil {
					msg := err.Error()
					fmt.Printf("Command '%s' failed: %s\n", cmd, msg)
				} else {
					err = this.syncUserDB()

					/*
					 * Check if something went wrong.
					 */
					if err != nil {
						msg := err.Error()
						fmt.Printf("%s\n", msg)
					}

				}

			}

		case "remove-user":

			/*
			 * Check number of arguments.
			 */
			if numArgs != 2 {
				fmt.Printf("Command '%s' expects 1 additional argument: name\n", cmd)
			} else {
				name := args[1]
				err := umgr.RemoveUser(name)

				/*
				 * Check if something went wrong.
				 */
				if err != nil {
					msg := err.Error()
					fmt.Printf("Command '%s' failed: %s\n", cmd, msg)
				} else {
					err = this.syncUserDB()

					/*
					 * Check if something went wrong.
					 */
					if err != nil {
						msg := err.Error()
						fmt.Printf("%s\n", msg)
					}

				}

			}

		case "set-password":

			/*
			 * Check number of arguments.
			 */
			if numArgs != 3 {
				fmt.Printf("Command '%s' expects 2 additional arguments: name, password\n", cmd)
			} else {
				name := args[1]
				password := args[2]
				err := umgr.SetPassword(name, password)

				/*
				 * Check if something went wrong.
				 */
				if err != nil {
					msg := err.Error()
					fmt.Printf("Command '%s' failed: %s\n", cmd, msg)
				} else {
					err = this.syncUserDB()

					/*
					 * Check if something went wrong.
					 */
					if err != nil {
						msg := err.Error()
						fmt.Printf("%s\n", msg)
					}

				}

			}

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}

	}

}

/*
 * Runs the server and message pump.
 */
func (this *controllerStruct) runServer() {
	cfg := this.config
	serverCfg := cfg.WebServer
	server := webserver.CreateWebServer(serverCfg)

	/*
	 * Check if we got a web server.
	 */
	if server == nil {
		fmt.Printf("%s\n", "Web server did not enter message loop.")
	} else {
		requests := server.RegisterCgi("/cgi-bin/locviz")
		server.Run()
		tlsPort := serverCfg.TLSPort
		fmt.Printf("Web interface ready: https://localhost:%s/\n", tlsPort)

		/*
		 * A worker processing HTTP requests.
		 */
		worker := func(requests <-chan webserver.HttpRequest) {

			/*
			 * This is the actual message pump.
			 */
			for request := range requests {
				response := this.dispatch(request)
				respond := request.Respond
				respond <- response
			}

		}

		numCPU := runtime.NumCPU()

		/*
		 * Spawn as many workers as we have CPUs.
		 */
		for i := 0; i < numCPU; i++ {
			go worker(requests)
		}

		stdin := os.Stdin
		scanner := bufio.NewScanner(stdin)

		/*
		 * Read from standard input forever.
		 */
		for {
			scanner.Scan()
		}

	}

}

/*
 * Initialize user database.
 */
func (this *controllerStruct) initializeUserDB() error {
	config := this.config
	userDBPath := config.UserDB
	contentUserDB, err := ioutil.ReadFile(userDBPath)

	/*
	 * Check if file could be read.
	 */
	if err != nil {
		return fmt.Errorf("Failed to open user database: '%s'", userDBPath)
	} else {
		r := rand.SystemPRNG()
		seed := make([]byte, rand.SEED_SIZE)
		_, err = r.Read(seed)

		/*
		 * Check if seed could be read from system.
		 */
		if err != nil {
			return fmt.Errorf("Failed to obtain entropy from system.")
		} else {
			prng, err := rand.CreatePRNG(seed)

			/*
			 * Check if PRNG could be created.
			 */
			if err != nil {
				msg := err.Error()
				return fmt.Errorf("Failed to create pseudo-random number generator: %s", msg)
			} else {
				userManager, err := user.CreateManager(prng)

				/*
				 * Check if user manager could be created.
				 */
				if err != nil {
					msg := err.Error()
					return fmt.Errorf("Failed to create user manager: ", msg)
				} else {
					this.userManager = userManager
					this.userDBPath = userDBPath
					err := userManager.Import(contentUserDB)

					/*
					 * Check if user database could be imported.
					 */
					if err != nil {
						msg := err.Error()
						return fmt.Errorf("Failed to import user database: ", msg)
					} else {
						expiryString := config.SessionExpiry
						expiry, _ := time.ParseDuration(expiryString)

						/*
						 * Set default session expiry of one hour.
						 */
						if expiry <= 0 {
							expiry = time.Hour
						}

						sessionManager, err := session.CreateManager(userManager, prng, expiry)

						/*
						 * Check if user manager could be created.
						 */
						if err != nil {
							msg := err.Error()
							return fmt.Errorf("Failed to create session manager: ", msg)
						} else {
							this.sessionManager = sessionManager
							return nil
						}

					}

				}

			}

		}

	}

}

/*
 * Initialize geographical data.
 */
func (this *controllerStruct) initializeGeoData() error {
	config := this.config
	geoDataPath := config.GeoData
	contentGeo, err := ioutil.ReadFile(geoDataPath)

	/*
	 * Check if file could be read.
	 */
	if err != nil {
		return fmt.Errorf("Could not read geo data file '%s'.", geoDataPath)
	} else {
		dataSet, err := geo.FromBytes(contentGeo)

		/*
		 * Check if geo data could be decoded.
		 */
		if err != nil {
			msg := err.Error()
			return fmt.Errorf("Could not decode geo data file '%s': %s", geoDataPath, msg)
		} else {
			numLocs := dataSet.LocationCount()
			data := make([]geo.Location, numLocs)

			/*
			 * Iterate over the locations and project them.
			 */
			for i := 0; i < numLocs; i++ {
				loc, err := dataSet.LocationAt(i)

				/*
				 * Verify that location could be obtained.
				 */
				if err == nil {
					data[i] = loc.Optimize()
				}

			}

			this.data = data
			return nil
		}

	}

}

/*
 * Initialize tile source.
 */
func (this *controllerStruct) initializeTileSource() {
	config := this.config
	cachePath := config.MapCache
	uri := config.MapServer
	useMap := config.UseMap

	/*
	 * Create OSM tile source if map should be used
	 * and cache path is set.
	 */
	if useMap && cachePath != "" {
		tileSource := tile.CreateOSMSource(uri, cachePath)
		this.tileSource = tileSource
	} else {
		this.tileSource = nil
	}

}

/*
 * Initialize the controller.
 */
func (this *controllerStruct) initialize() error {
	content, err := ioutil.ReadFile(CONFIG_PATH)

	/*
	 * Check if file could be read.
	 */
	if err != nil {
		return fmt.Errorf("Could not open config file: '%s'", CONFIG_PATH)
	} else {
		config := configStruct{}
		err = json.Unmarshal(content, &config)
		this.config = config

		/*
		 * Check if file failed to unmarshal.
		 */
		if err != nil {
			return fmt.Errorf("Could not decode config file: '%s'", CONFIG_PATH)
		} else {
			err = this.initializeUserDB()

			/*
			 * Check if user database could be initialized.
			 */
			if err != nil {
				return err
			} else {
				return nil
			}

		}

	}

}

/*
 * Main routine of our controller. Performs initialization, then runs the message pump.
 */
func (this *controllerStruct) Operate(args []string) {
	err := this.initialize()

	/*
	 * Check if initialization was successful.
	 */
	if err != nil {
		msg := err.Error()
		msgNew := "Initialization failed: " + msg
		fmt.Printf("%s\n", msgNew)
	} else {
		numArgs := len(args)

		/*
		 * If no arguments are passed, run the server, otherwise interpret them.
		 */
		if numArgs == 0 {
			this.initializeGeoData()
			this.initializeTileSource()
			this.runServer()
		} else {
			this.interpret(args)
		}

	}

}

/*
 * Pre-fetch tile data from OSM.
 */
func (this *controllerStruct) Prefetch(zoomLevel uint8) {
	err := this.initialize()

	/*
	 * Check if initialization was successful.
	 */
	if err != nil {
		msg := err.Error()
		msgNew := "Initialization failed: " + msg
		fmt.Printf("%s\n", msgNew)
	} else {
		tileSource := this.tileSource
		tileSource.Prefetch(zoomLevel)
	}

}

/*
 * Creates a new controller.
 */
func CreateController() Controller {
	controller := controllerStruct{}
	return &controller
}
