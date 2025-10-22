package client

import (
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/andrepxx/location-visualizer/auth/publickey"
	"github.com/andrepxx/location-visualizer/remote"
)

const (
	DEFAULT_BUFFER_SIZE = 8196
	DEFAULT_FILE_MODE   = 0666
)

/*
 * Interface representing the controller for the command-line client.
 */
type Controller interface {
	Interpret(args []string)
}

/*
 * Data structure representing the controller for the command-line client.
 */
type controllerStruct struct {
	userAgent string
}

/*
 * Loads a certificate from a path.
 */
func (this *controllerStruct) loadCertificate(path string) ([]byte, error) {
	result := []byte(nil)
	certificateBytes, err := os.ReadFile(path)

	/*
	 * If certificate could be loaded, decode it.
	 */
	if err == nil {
		block, _ := pem.Decode(certificateBytes)
		result = block.Bytes
	}

	return result, err
}

/*
 * Login to a remote server using RSA authentication.
 */
func (this *controllerStruct) loginPrivateKey(conn remote.Connection, user string, keyFilePath string) (remote.Session, error) {
	result := remote.Session(nil)
	errResult := error(nil)
	pemData, err := os.ReadFile(keyFilePath)

	/*
	 * Check if private key could be loaded.
	 */
	if err != nil {
		msg := err.Error()
		errResult = fmt.Errorf("Failed to load private key: %s", msg)
	} else {
		keyData, representation, err := publickey.DecodePEM(pemData)

		/*
		 * Check if private key could be decoded.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to decode private key: %s", msg)
		} else {
			rsaPrivateKey, err := publickey.LoadRSAPrivateKey(keyData, representation)

			/*
			 * Check if private key could be loaded.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Failed to load private key: %s", msg)
			} else {
				result, errResult = conn.LoginPrivateKey(user, rsaPrivateKey)
			}

		}

	}

	return result, errResult
}

/*
 * Export activities from remote server into a CSV file.
 */
func (this *controllerStruct) exportActivityCsv(args []string, useKeyFile bool) {
	const EXPECTED_NUMBER_OF_ARGS = 8
	numArgs := len(args)

	/*
	 * Check if we have the expected number of arguments.
	 */
	if numArgs != EXPECTED_NUMBER_OF_ARGS {
		fmt.Printf("Expected %d arguments\n", EXPECTED_NUMBER_OF_ARGS)
	} else {
		host := args[2]
		portString := args[3]
		certificatePath := args[4]
		user := args[5]
		passwordOrKeyFilePath := args[6]
		path := args[7]
		port, errPort := strconv.ParseUint(portString, 10, 16)
		certificate, errCertificate := this.loadCertificate(certificatePath)

		/*
		 * Check if port number could be parsed and certificate could be read.
		 */
		if errPort != nil {
			fmt.Printf("%s\n", "Failed to parse port number")
		} else if errCertificate != nil {
			fmt.Printf("%s\n", "Failed to load certificate")
		} else {
			portValue := uint16(port)
			userAgent := this.userAgent
			conn, err := remote.CreateConnection(host, portValue, userAgent, certificate)

			/*
			 * Check if connection could be established.
			 */
			if err != nil {
				msg := err.Error()
				fmt.Printf("Failed to establish connection: %s\n", msg)
			} else {
				sess := remote.Session(nil)

				/*
				 * Login using password or private key.
				 */
				if useKeyFile {
					sess, err = this.loginPrivateKey(conn, user, passwordOrKeyFilePath)
				} else {
					sess, err = conn.Login(user, passwordOrKeyFilePath)
				}

				/*
				 * Check if session could be established.
				 */
				if err != nil {
					msg := err.Error()
					fmt.Printf("Failed to establish session: %s\n", msg)
				} else {
					flags := int(os.O_CREATE | os.O_EXCL | os.O_WRONLY)
					fd, err := os.OpenFile(path, flags, DEFAULT_FILE_MODE)

					/*
					 * Check if file could be created.
					 */
					if err != nil {
						msg := err.Error()
						fmt.Printf("Failed to create output file: %s\n", msg)
					} else {
						fdRemote, err := sess.ExportActivityCsv()

						/*
						 * Check if error occured during export call.
						 */
						if err != nil {
							msg := err.Error()
							fmt.Printf("Failed to export activity data: %s\n", msg)
						} else {
							buf := make([]byte, DEFAULT_BUFFER_SIZE)
							_, err := io.CopyBuffer(fd, fdRemote, buf)

							/*
							 * Check if error occured during export process.
							 */
							if err != nil {
								msg := err.Error()
								fmt.Printf("Error reading from remote connection: %s\n", msg)
							}

						}

					}

					err = sess.Logout()

					/*
					* Check if session could be terminated.
					 */
					if err != nil {
						msg := err.Error()
						fmt.Printf("Failed to terminate session: %s\n", msg)
					}

				}

			}

		}

	}

}

/*
 * Export geo data from remote server into a file of the selected format.
 */
func (this *controllerStruct) exportGeodata(args []string, useKeyFile bool) {
	const EXPECTED_NUMBER_OF_ARGS = 9
	numArgs := len(args)

	/*
	 * Check if we have the expected number of arguments.
	 */
	if numArgs != EXPECTED_NUMBER_OF_ARGS {
		fmt.Printf("Expected %d arguments\n", EXPECTED_NUMBER_OF_ARGS)
	} else {
		host := args[2]
		portString := args[3]
		certificatePath := args[4]
		user := args[5]
		passwordOrKeyFilePath := args[6]
		format := args[7]
		path := args[8]
		port, errPort := strconv.ParseUint(portString, 10, 16)
		certificate, errCertificate := this.loadCertificate(certificatePath)

		/*
		 * Check if port number could be parsed and certificate could be read.
		 */
		if errPort != nil {
			fmt.Printf("%s\n", "Failed to parse port number")
		} else if errCertificate != nil {
			fmt.Printf("%s\n", "Failed to load certificate")
		} else {
			portValue := uint16(port)
			userAgent := this.userAgent
			conn, err := remote.CreateConnection(host, portValue, userAgent, certificate)

			/*
			 * Check if connection could be established.
			 */
			if err != nil {
				msg := err.Error()
				fmt.Printf("Failed to establish connection: %s\n", msg)
			} else {
				sess := remote.Session(nil)

				/*
				 * Login using password or private key.
				 */
				if useKeyFile {
					sess, err = this.loginPrivateKey(conn, user, passwordOrKeyFilePath)
				} else {
					sess, err = conn.Login(user, passwordOrKeyFilePath)
				}

				/*
				 * Check if session could be established.
				 */
				if err != nil {
					msg := err.Error()
					fmt.Printf("Failed to establish session: %s\n", msg)
				} else {
					flags := int(os.O_CREATE | os.O_EXCL | os.O_WRONLY)
					fd, err := os.OpenFile(path, flags, DEFAULT_FILE_MODE)

					/*
					 * Check if file could be created.
					 */
					if err != nil {
						msg := err.Error()
						fmt.Printf("Failed to create output file: %s\n", msg)
					} else {
						fdRemote, err := sess.ExportGeodata(format)

						/*
						 * Check if error occured during export call.
						 */
						if err != nil {
							msg := err.Error()
							fmt.Printf("Failed to export geo data: %s\n", msg)
						} else {
							buf := make([]byte, DEFAULT_BUFFER_SIZE)
							_, err := io.CopyBuffer(fd, fdRemote, buf)

							/*
							* Check if error occured during export process.
							 */
							if err != nil {
								msg := err.Error()
								fmt.Printf("Error reading from remote connection: %s\n", msg)
							}

						}

					}

					err = sess.Logout()

					/*
					 * Check if session could be terminated.
					 */
					if err != nil {
						msg := err.Error()
						fmt.Printf("Failed to terminate session: %s\n", msg)
					}

				}

			}

		}

	}

}

/*
 * Import geo data to remote server from a file of the selected format.
 */
func (this *controllerStruct) importGeodata(args []string, useKeyFile bool) {
	const EXPECTED_NUMBER_OF_ARGS = 10
	numArgs := len(args)

	/*
	 * Check if we have the expected number of arguments.
	 */
	if numArgs != EXPECTED_NUMBER_OF_ARGS {
		fmt.Printf("Expected %d arguments\n", EXPECTED_NUMBER_OF_ARGS)
	} else {
		host := args[2]
		portString := args[3]
		certificatePath := args[4]
		user := args[5]
		passwordOrKeyFilePath := args[6]
		format := args[7]
		strategy := args[8]
		path := args[9]
		port, errPort := strconv.ParseUint(portString, 10, 16)
		certificate, errCertificate := this.loadCertificate(certificatePath)

		/*
		 * Check if port number could be parsed and certificate could be read.
		 */
		if errPort != nil {
			fmt.Printf("%s\n", "Failed to parse port number")
		} else if errCertificate != nil {
			fmt.Printf("%s\n", "Failed to load certificate")
		} else {
			portValue := uint16(port)
			userAgent := this.userAgent
			conn, err := remote.CreateConnection(host, portValue, userAgent, certificate)

			/*
			 * Check if connection could be established.
			 */
			if err != nil {
				msg := err.Error()
				fmt.Printf("Failed to establish connection: %s\n", msg)
			} else {
				sess := remote.Session(nil)

				/*
				 * Login using password or private key.
				 */
				if useKeyFile {
					sess, err = this.loginPrivateKey(conn, user, passwordOrKeyFilePath)
				} else {
					sess, err = conn.Login(user, passwordOrKeyFilePath)
				}

				/*
				 * Check if session could be established.
				 */
				if err != nil {
					msg := err.Error()
					fmt.Printf("Failed to establish session: %s\n", msg)
				} else {
					fd, err := os.Open(path)

					/*
					 * Check if file could be created.
					 */
					if err != nil {
						msg := err.Error()
						fmt.Printf("Failed to open input file: %s\n", msg)
					} else {
						r, err := sess.ImportGeodata(format, strategy, fd)

						/*
						 * Check if error occured during import call.
						 */
						if err != nil {
							msg := err.Error()
							fmt.Printf("Failed to import geo data: %s\n", msg)
						} else if r != nil {
							_, err := io.Copy(os.Stdout, r)

							/*
							 * Check if error occured reading response.
							 */
							if err != nil {
								msg := err.Error()
								fmt.Printf("Failed to read response: %s\n", msg)
							}

							fmt.Printf("%s\n", "")
						}

					}

					err = sess.Logout()

					/*
					 * Check if session could be terminated.
					 */
					if err != nil {
						msg := err.Error()
						fmt.Printf("Failed to terminate session: %s\n", msg)
					}

				}

			}

		}

	}

}

/*
 * Interpret user commands entered into shell.
 */
func (this *controllerStruct) Interpret(args []string) {
	numArgs := len(args)

	/*
	 * Ensure that there is at least one argument.
	 */
	if numArgs < 2 {
		fmt.Printf("%s\n", "Missing argument / command")
	} else {
		cmd := args[1]

		/*
		 * Perform action based on command.
		 */
		switch cmd {
		case "export-activity-csv":
			this.exportActivityCsv(args, false)
		case "export-activity-csv-pk":
			this.exportActivityCsv(args, true)
		case "export-geodata":
			this.exportGeodata(args, false)
		case "export-geodata-pk":
			this.exportGeodata(args, true)
		case "import-geodata":
			this.importGeodata(args, false)
		case "import-geodata-pk":
			this.importGeodata(args, true)
		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}

	}

}

/*
 * Creates a client controller.
 */
func CreateController(userAgent string) Controller {

	/*
	 * Create client controller.
	 */
	controller := controllerStruct{
		userAgent: userAgent,
	}

	return &controller
}
