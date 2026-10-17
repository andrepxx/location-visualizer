package client

import (
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/andrepxx/location-visualizer/auth/publickey"
	"github.com/andrepxx/location-visualizer/controller/communication"
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

		/*
		 * The block may be nil on decoding errors.
		 */
		if block == nil {
			err = fmt.Errorf("%s", "Failed to decode PEM block")
		} else {
			result = block.Bytes
		}

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
 * Check whether a user on a server has a specific permission.
 *
 * This action is invoked by instance administrators via the CLI client.
 */
func (this *controllerStruct) adminHasDeviceToken(args []string, useKeyFile bool) {
	const EXPECTED_NUMBER_OF_ARGS = 9
	numArgs := len(args)

	/*
	 * Check if we have the expected number of arguments.
	 */
	if numArgs != EXPECTED_NUMBER_OF_ARGS {
		stderr := os.Stderr
		fmt.Fprintf(stderr, "Expected %d arguments\n", EXPECTED_NUMBER_OF_ARGS)
	} else {
		host := args[2]
		portString := args[3]
		certificatePath := args[4]
		user := args[5]
		passwordOrKeyFilePath := args[6]
		name := args[7]
		deviceTokenString := args[8]
		port, errPort := strconv.ParseUint(portString, 10, 16)
		certificate, errCertificate := this.loadCertificate(certificatePath)
		deviceToken, errDeviceToken := strconv.ParseUint(deviceTokenString, 16, 64)

		/*
		 * Check if port number could be parsed and certificate could be read.
		 */
		if errPort != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to parse port number")
		} else if errCertificate != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to load certificate")
		} else if errDeviceToken != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to parse device token")
		} else {
			portValue := uint16(port)
			userAgent := this.userAgent
			conn, err := remote.CreateConnection(host, portValue, userAgent, certificate)

			/*
			 * Check if connection could be established.
			 */
			if err != nil {
				msg := err.Error()
				stderr := os.Stderr
				fmt.Fprintf(stderr, "Failed to establish connection: %s\n", msg)
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
					stderr := os.Stderr
					fmt.Fprintf(stderr, "Failed to establish session: %s\n", msg)
				} else {
					fdRemote, err := sess.AdminHasDeviceToken(name, deviceToken)

					/*
					 * Check if error occured during retrieval.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to check for device token: %s\n", msg)
					} else {
						content, err := io.ReadAll(fdRemote)

						/*
						 * Check if error occured during retrieval.
						 */
						if err != nil {
							msg := err.Error()
							stderr := os.Stderr
							fmt.Fprintf(stderr, "Failed to check for device token: %s\n", msg)
						} else {
							webBooleanResult := communication.WebBooleanResultStruct{}
							err := json.Unmarshal(content, &webBooleanResult)

							/*
							 * Check if JSON structure could be decoded.
							 */
							if err != nil {
								msg := err.Error()
								stderr := os.Stderr
								fmt.Fprintf(stderr, "Failed to decode boolean result: %s\n", msg)
							} else {
								success := webBooleanResult.Success

								/*
								 * Check if a remote error occurred.
								 */
								if !success {
									reason := webBooleanResult.Reason
									stderr := os.Stderr
									fmt.Fprintf(stderr, "Failed to retrieve boolean result: %s\n", reason)
								} else {
									value := webBooleanResult.Value
									fmt.Printf("%t\n", value)
								}

							}

						}

					}

					err = sess.Logout()

					/*
					 * Check if session could be terminated.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to terminate session: %s\n", msg)
					}

				}

			}

		}

	}

}

/*
 * Check whether a user on a server has a specific permission.
 *
 * This action is invoked by instance administrators via the CLI client.
 */
func (this *controllerStruct) adminHasPermission(args []string, useKeyFile bool) {
	const EXPECTED_NUMBER_OF_ARGS = 9
	numArgs := len(args)

	/*
	 * Check if we have the expected number of arguments.
	 */
	if numArgs != EXPECTED_NUMBER_OF_ARGS {
		stderr := os.Stderr
		fmt.Fprintf(stderr, "Expected %d arguments\n", EXPECTED_NUMBER_OF_ARGS)
	} else {
		host := args[2]
		portString := args[3]
		certificatePath := args[4]
		user := args[5]
		passwordOrKeyFilePath := args[6]
		name := args[7]
		permission := args[8]
		port, errPort := strconv.ParseUint(portString, 10, 16)
		certificate, errCertificate := this.loadCertificate(certificatePath)

		/*
		 * Check if port number could be parsed and certificate could be read.
		 */
		if errPort != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to parse port number")
		} else if errCertificate != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to load certificate")
		} else {
			portValue := uint16(port)
			userAgent := this.userAgent
			conn, err := remote.CreateConnection(host, portValue, userAgent, certificate)

			/*
			 * Check if connection could be established.
			 */
			if err != nil {
				msg := err.Error()
				stderr := os.Stderr
				fmt.Fprintf(stderr, "Failed to establish connection: %s\n", msg)
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
					stderr := os.Stderr
					fmt.Fprintf(stderr, "Failed to establish session: %s\n", msg)
				} else {
					fdRemote, err := sess.AdminHasPermission(name, permission)

					/*
					 * Check if error occured during retrieval.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to check for permission: %s\n", msg)
					} else {
						content, err := io.ReadAll(fdRemote)

						/*
						 * Check if error occured during retrieval.
						 */
						if err != nil {
							msg := err.Error()
							stderr := os.Stderr
							fmt.Fprintf(stderr, "Failed to check for permission: %s\n", msg)
						} else {
							webBooleanResult := communication.WebBooleanResultStruct{}
							err := json.Unmarshal(content, &webBooleanResult)

							/*
							 * Check if JSON structure could be decoded.
							 */
							if err != nil {
								msg := err.Error()
								stderr := os.Stderr
								fmt.Fprintf(stderr, "Failed to decode boolean result: %s\n", msg)
							} else {
								success := webBooleanResult.Success

								/*
								 * Check if a remote error occurred.
								 */
								if !success {
									reason := webBooleanResult.Reason
									stderr := os.Stderr
									fmt.Fprintf(stderr, "Failed to retrieve boolean result: %s\n", reason)
								} else {
									value := webBooleanResult.Value
									fmt.Printf("%t\n", value)
								}

							}

						}

					}

					err = sess.Logout()

					/*
					 * Check if session could be terminated.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to terminate session: %s\n", msg)
					}

				}

			}

		}

	}

}

/*
 * Retrieve a list of device tokens of a user on a server.
 *
 * This action is invoked by instance administrators via the CLI client.
 */
func (this *controllerStruct) adminListDeviceTokens(args []string, useKeyFile bool) {
	const EXPECTED_NUMBER_OF_ARGS = 8
	numArgs := len(args)

	/*
	 * Check if we have the expected number of arguments.
	 */
	if numArgs != EXPECTED_NUMBER_OF_ARGS {
		stderr := os.Stderr
		fmt.Fprintf(stderr, "Expected %d arguments\n", EXPECTED_NUMBER_OF_ARGS)
	} else {
		host := args[2]
		portString := args[3]
		certificatePath := args[4]
		user := args[5]
		passwordOrKeyFilePath := args[6]
		name := args[7]
		port, errPort := strconv.ParseUint(portString, 10, 16)
		certificate, errCertificate := this.loadCertificate(certificatePath)

		/*
		 * Check if port number could be parsed and certificate could be read.
		 */
		if errPort != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to parse port number")
		} else if errCertificate != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to load certificate")
		} else {
			portValue := uint16(port)
			userAgent := this.userAgent
			conn, err := remote.CreateConnection(host, portValue, userAgent, certificate)

			/*
			 * Check if connection could be established.
			 */
			if err != nil {
				msg := err.Error()
				stderr := os.Stderr
				fmt.Fprintf(stderr, "Failed to establish connection: %s\n", msg)
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
					stderr := os.Stderr
					fmt.Fprintf(stderr, "Failed to establish session: %s\n", msg)
				} else {
					fdRemote, err := sess.AdminGetDeviceTokens(name)

					/*
					 * Check if error occured during retrieval.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to retrieve device tokens: %s\n", msg)
					} else {
						content, err := io.ReadAll(fdRemote)

						/*
						 * Check if error occured during retrieval.
						 */
						if err != nil {
							msg := err.Error()
							stderr := os.Stderr
							fmt.Fprintf(stderr, "Failed to retrieve device tokens: %s\n", msg)
						} else {
							webDeviceTokens := communication.WebDeviceTokensStruct{}
							err := json.Unmarshal(content, &webDeviceTokens)

							/*
							 * Check if JSON structure could be decoded.
							 */
							if err != nil {
								msg := err.Error()
								stderr := os.Stderr
								fmt.Fprintf(stderr, "Failed to decode device tokens: %s\n", msg)
							} else {
								success := webDeviceTokens.Success

								/*
								 * Check if a remote error occurred.
								 */
								if !success {
									reason := webDeviceTokens.Reason
									stderr := os.Stderr
									fmt.Fprintf(stderr, "Failed to retrieve device tokens: %s\n", reason)
								} else {
									deviceTokens := webDeviceTokens.DeviceTokens

									/*
									* Iterate over all device tokens to list them.
									 */
									for _, deviceToken := range deviceTokens {
										creationTime := deviceToken.CreationTime
										description := deviceToken.Description
										token := deviceToken.Token
										fmt.Printf("%s %s %s\n", token, creationTime, description)
									}

								}

							}

						}

					}

					err = sess.Logout()

					/*
					 * Check if session could be terminated.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to terminate session: %s\n", msg)
					}

				}

			}

		}

	}

}

/*
 * Retrieve a list of permissions of a user on a server.
 *
 * This action is invoked by instance administrators via the CLI client.
 */
func (this *controllerStruct) adminListPermissions(args []string, useKeyFile bool) {
	const EXPECTED_NUMBER_OF_ARGS = 8
	numArgs := len(args)

	/*
	 * Check if we have the expected number of arguments.
	 */
	if numArgs != EXPECTED_NUMBER_OF_ARGS {
		stderr := os.Stderr
		fmt.Fprintf(stderr, "Expected %d arguments\n", EXPECTED_NUMBER_OF_ARGS)
	} else {
		host := args[2]
		portString := args[3]
		certificatePath := args[4]
		user := args[5]
		passwordOrKeyFilePath := args[6]
		name := args[7]
		port, errPort := strconv.ParseUint(portString, 10, 16)
		certificate, errCertificate := this.loadCertificate(certificatePath)

		/*
		 * Check if port number could be parsed and certificate could be read.
		 */
		if errPort != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to parse port number")
		} else if errCertificate != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to load certificate")
		} else {
			portValue := uint16(port)
			userAgent := this.userAgent
			conn, err := remote.CreateConnection(host, portValue, userAgent, certificate)

			/*
			 * Check if connection could be established.
			 */
			if err != nil {
				msg := err.Error()
				stderr := os.Stderr
				fmt.Fprintf(stderr, "Failed to establish connection: %s\n", msg)
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
					stderr := os.Stderr
					fmt.Fprintf(stderr, "Failed to establish session: %s\n", msg)
				} else {
					fdRemote, err := sess.AdminGetPermissions(name)

					/*
					 * Check if error occured during retrieval.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to retrieve permissions list: %s\n", msg)
					} else {
						content, err := io.ReadAll(fdRemote)

						/*
						 * Check if error occured during retrieval.
						 */
						if err != nil {
							msg := err.Error()
							stderr := os.Stderr
							fmt.Fprintf(stderr, "Failed to retrieve permissions list: %s\n", msg)
						} else {
							webPermissions := communication.WebPermissionsStruct{}
							err := json.Unmarshal(content, &webPermissions)

							/*
							 * Check if JSON structure could be decoded.
							 */
							if err != nil {
								msg := err.Error()
								stderr := os.Stderr
								fmt.Fprintf(stderr, "Failed to decode permissions list: %s\n", msg)
							} else {
								success := webPermissions.Success

								/*
								 * Check if a remote error occurred.
								 */
								if !success {
									reason := webPermissions.Reason
									stderr := os.Stderr
									fmt.Fprintf(stderr, "Failed to retrieve permissions list: %s\n", reason)
								} else {
									permissions := webPermissions.Permissions

									/*
									 * Iterate over all permissions to list them.
									 */
									for _, permission := range permissions {
										fmt.Printf("%s\n", permission)
									}

								}

							}

						}

					}

					err = sess.Logout()

					/*
					 * Check if session could be terminated.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to terminate session: %s\n", msg)
					}

				}

			}

		}

	}

}

/*
 * Retrieve a list of public keys of a user on a server.
 *
 * This action is invoked by instance administrators via the CLI client.
 */
func (this *controllerStruct) adminListPublicKeys(args []string, useKeyFile bool) {
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
		name := args[7]
		port, errPort := strconv.ParseUint(portString, 10, 16)
		certificate, errCertificate := this.loadCertificate(certificatePath)

		/*
		 * Check if port number could be parsed and certificate could be read.
		 */
		if errPort != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to parse port number")
		} else if errCertificate != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to load certificate")
		} else {
			portValue := uint16(port)
			userAgent := this.userAgent
			conn, err := remote.CreateConnection(host, portValue, userAgent, certificate)

			/*
			 * Check if connection could be established.
			 */
			if err != nil {
				msg := err.Error()
				stderr := os.Stderr
				fmt.Fprintf(stderr, "Failed to establish connection: %s\n", msg)
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
					stderr := os.Stderr
					fmt.Fprintf(stderr, "Failed to establish session: %s\n", msg)
				} else {
					fdRemote, err := sess.AdminGetPublicKeys(name)

					/*
					 * Check if error occured during retrieval.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to retrieve public keys: %s\n", msg)
					} else {
						content, err := io.ReadAll(fdRemote)

						/*
						 * Check if error occured during retrieval.
						 */
						if err != nil {
							msg := err.Error()
							stderr := os.Stderr
							fmt.Fprintf(stderr, "Failed to retrieve public keys: %s\n", msg)
						} else {
							webPublicKeys := communication.WebPublicKeysStruct{}
							err := json.Unmarshal(content, &webPublicKeys)

							/*
							 * Check if JSON structure could be decoded.
							 */
							if err != nil {
								msg := err.Error()
								stderr := os.Stderr
								fmt.Fprintf(stderr, "Failed to decode public keys: %s\n", msg)
							} else {
								success := webPublicKeys.Success

								/*
								 * Check if a remote error occurred.
								 */
								if !success {
									reason := webPublicKeys.Reason
									stderr := os.Stderr
									fmt.Fprintf(stderr, "Failed to retrieve public keys: %s\n", reason)
								} else {
									publicKeys := webPublicKeys.PublicKeys

									/*
									* Iterate over all public keys to list them.
									 */
									for i, publicKey := range publicKeys {
										creationTime := publicKey.CreationTime
										description := publicKey.Description
										hash := publicKey.Hash
										representation := publicKey.Representation
										fmt.Printf("%d: %s [%s] %s %s\n", i, creationTime, representation, hash, description)
									}

								}

							}

						}

					}

					err = sess.Logout()

					/*
					 * Check if session could be terminated.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to terminate session: %s\n", msg)
					}

				}

			}

		}

	}

}

/*
 * Retrieve a list of users on a server.
 *
 * This action is invoked by instance administrators via the CLI client.
 */
func (this *controllerStruct) adminListUsers(args []string, useKeyFile bool) {
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
		certificatePath := args[4]
		user := args[5]
		passwordOrKeyFilePath := args[6]
		port, errPort := strconv.ParseUint(portString, 10, 16)
		certificate, errCertificate := this.loadCertificate(certificatePath)

		/*
		 * Check if port number could be parsed and certificate could be read.
		 */
		if errPort != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to parse port number")
		} else if errCertificate != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to load certificate")
		} else {
			portValue := uint16(port)
			userAgent := this.userAgent
			conn, err := remote.CreateConnection(host, portValue, userAgent, certificate)

			/*
			 * Check if connection could be established.
			 */
			if err != nil {
				msg := err.Error()
				stderr := os.Stderr
				fmt.Fprintf(stderr, "Failed to establish connection: %s\n", msg)
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
					stderr := os.Stderr
					fmt.Fprintf(stderr, "Failed to establish session: %s\n", msg)
				} else {
					fdRemote, err := sess.AdminGetUsers()

					/*
					 * Check if error occured during retrieval.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to retrieve user list: %s\n", msg)
					} else {
						content, err := io.ReadAll(fdRemote)

						/*
						 * Check if error occured during retrieval.
						 */
						if err != nil {
							msg := err.Error()
							stderr := os.Stderr
							fmt.Fprintf(stderr, "Failed to retrieve user list: %s\n", msg)
						} else {
							users := []string{}
							err := json.Unmarshal(content, &users)

							/*
							 * Check if JSON structure could be decoded.
							 */
							if err != nil {
								msg := err.Error()
								stderr := os.Stderr
								fmt.Fprintf(stderr, "Failed to decode user list: %s", msg)
							} else {

								/*
								 * Iterate over all users to list them.
								 */
								for _, user := range users {
									fmt.Printf("%s\n", user)
								}

							}

						}

					}

					err = sess.Logout()

					/*
					 * Check if session could be terminated.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to terminate session: %s\n", msg)
					}

				}

			}

		}

	}

}

/*
 * Retrieve a list of public keys of a user on a server.
 *
 * This action is invoked by instance administrators via the CLI client.
 */
func (this *controllerStruct) adminShowPublicKey(args []string, useKeyFile bool) {
	const EXPECTED_NUMBER_OF_ARGS = 9
	numArgs := len(args)

	/*
	 * Check if we have the expected number of arguments.
	 */
	if numArgs != EXPECTED_NUMBER_OF_ARGS {
		stderr := os.Stderr
		fmt.Fprintf(stderr, "Expected %d arguments\n", EXPECTED_NUMBER_OF_ARGS)
	} else {
		host := args[2]
		portString := args[3]
		certificatePath := args[4]
		user := args[5]
		passwordOrKeyFilePath := args[6]
		name := args[7]
		keyIdString := args[8]
		port, errPort := strconv.ParseUint(portString, 10, 16)
		keyId, errKeyId := strconv.ParseUint(keyIdString, 10, 16)
		certificate, errCertificate := this.loadCertificate(certificatePath)

		/*
		 * Check if port number could be parsed and certificate could be read.
		 */
		if errPort != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to parse port number")
		} else if errKeyId != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to parse key id")
		} else if errCertificate != nil {
			stderr := os.Stderr
			fmt.Fprintf(stderr, "%s\n", "Failed to load certificate")
		} else {
			portValue := uint16(port)
			userAgent := this.userAgent
			conn, err := remote.CreateConnection(host, portValue, userAgent, certificate)

			/*
			 * Check if connection could be established.
			 */
			if err != nil {
				msg := err.Error()
				stderr := os.Stderr
				fmt.Fprintf(stderr, "Failed to establish connection: %s\n", msg)
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
					stderr := os.Stderr
					fmt.Fprintf(stderr, "Failed to establish session: %s\n", msg)
				} else {
					fdRemote, err := sess.AdminGetPublicKey(name, keyId)

					/*
					 * Check if error occured during retrieval.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to retrieve public key: %s\n", msg)
					} else {
						content, err := io.ReadAll(fdRemote)

						/*
						 * Check if error occured during retrieval.
						 */
						if err != nil {
							msg := err.Error()
							stderr := os.Stderr
							fmt.Fprintf(stderr, "Failed to retrieve public key: %s\n", msg)
						} else {
							webPublicKeyPemData := communication.WebPublicKeyPemDataStruct{}
							err := json.Unmarshal(content, &webPublicKeyPemData)

							/*
							 * Check if JSON structure could be decoded.
							 */
							if err != nil {
								msg := err.Error()
								stderr := os.Stderr
								fmt.Fprintf(stderr, "Failed to decode public keys: %s\n", msg)
							} else {
								success := webPublicKeyPemData.Success

								/*
								 * Check if a remote error occurred.
								 */
								if !success {
									reason := webPublicKeyPemData.Reason
									stderr := os.Stderr
									fmt.Fprintf(stderr, "Failed to retrieve public key: %s\n", reason)
								} else {
									pemData := webPublicKeyPemData.PemData
									fmt.Printf("%s", pemData)
								}

							}

						}

					}

					err = sess.Logout()

					/*
					 * Check if session could be terminated.
					 */
					if err != nil {
						msg := err.Error()
						stderr := os.Stderr
						fmt.Fprintf(stderr, "Failed to terminate session: %s\n", msg)
					}

				}

			}

		}

	}

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
					 * Check if file could be opened.
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
		case "admin-has-device-token":
			this.adminHasDeviceToken(args, false)
		case "admin-has-device-token-pk":
			this.adminHasDeviceToken(args, true)
		case "admin-has-permission":
			this.adminHasPermission(args, false)
		case "admin-has-permission-pk":
			this.adminHasPermission(args, true)
		case "admin-list-device-tokens":
			this.adminListDeviceTokens(args, false)
		case "admin-list-device-tokens-pk":
			this.adminListDeviceTokens(args, true)
		case "admin-list-permissions":
			this.adminListPermissions(args, false)
		case "admin-list-permissions-pk":
			this.adminListPermissions(args, true)
		case "admin-list-public-keys":
			this.adminListPublicKeys(args, false)
		case "admin-list-public-keys-pk":
			this.adminListPublicKeys(args, true)
		case "admin-list-users":
			this.adminListUsers(args, false)
		case "admin-list-users-pk":
			this.adminListUsers(args, true)
		case "admin-show-public-key":
			this.adminShowPublicKey(args, false)
		case "admin-show-public-key-pk":
			this.adminShowPublicKey(args, true)
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
