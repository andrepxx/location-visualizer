package client

import (
	"fmt"
	"io"
	"os"
	"strconv"

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
 * Export activities from remote server into a CSV file.
 */
func (this *controllerStruct) exportActivityCsv(args []string) {
	const EXPECTED_NUMBER_OF_ARGS = 7
	numArgs := len(args)

	/*
	 * Check if we have the expected number of arguments.
	 */
	if numArgs != EXPECTED_NUMBER_OF_ARGS {
		fmt.Printf("Expected %d arguments\n", EXPECTED_NUMBER_OF_ARGS)
	} else {
		host := args[2]
		portString := args[3]
		user := args[4]
		password := args[5]
		path := args[6]
		port, err := strconv.ParseUint(portString, 10, 16)

		/*
		 * Check if port number could be parsed.
		 */
		if err != nil {
			fmt.Printf("%s\n", "Failed to parse port number")
		} else {
			portValue := uint16(port)
			userAgent := this.userAgent
			conn := remote.CreateConnection(host, portValue, userAgent)
			sess, err := conn.Login(user, password)

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

/*
 * Export geo data from remote server into a file of the selected format.
 */
func (this *controllerStruct) exportGeodata(args []string) {
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
		user := args[4]
		password := args[5]
		format := args[6]
		path := args[7]
		port, err := strconv.ParseUint(portString, 10, 16)

		/*
		 * Check if port number could be parsed.
		 */
		if err != nil {
			fmt.Printf("%s\n", "Failed to parse port number")
		} else {
			portValue := uint16(port)
			userAgent := this.userAgent
			conn := remote.CreateConnection(host, portValue, userAgent)
			sess, err := conn.Login(user, password)

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

/*
 * Import geo data to remote server from a file of the selected format.
 */
func (this *controllerStruct) importGeodata(args []string) {
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
		user := args[4]
		password := args[5]
		format := args[6]
		strategy := args[7]
		path := args[8]
		port, err := strconv.ParseUint(portString, 10, 16)

		/*
		 * Check if port number could be parsed.
		 */
		if err != nil {
			fmt.Printf("%s\n", "Failed to parse port number")
		} else {
			portValue := uint16(port)
			userAgent := this.userAgent
			conn := remote.CreateConnection(host, portValue, userAgent)
			sess, err := conn.Login(user, password)

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
			this.exportActivityCsv(args)
		case "export-geodata":
			this.exportGeodata(args)
		case "import-geodata":
			this.importGeodata(args)
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
