package user

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"
)

/*
 * Global constants.
 */
const (
	BASE_HEX      = 16
	LENGTH        = 64
	SIZE_TOKEN    = 8
	UNAME_L_LIMIT = 3
	UNAME_U_LIMIT = 16
	UNAME_REX     = "^[A-Za-z0-9\\-_\\.]+$"
)

/*
 * A device token, as represented in memory,
 */
type deviceTokenStruct struct {
	creationTime time.Time
	description  string
	token        uint64
}

/*
 * A device token, as represented on disk.
 */
type persistedDeviceTokenStruct struct {
	CreationTime string
	Description  string
	Token        string
}

/*
 * A user, as represented in memory.
 */
type userStruct struct {
	name         string
	salt         [LENGTH]byte
	hash         []byte
	nonce        [LENGTH]byte
	permissions  []string
	deviceTokens []deviceTokenStruct
}

/*
 * A user, as represented on disk.
 */
type persistedUserStruct struct {
	Name         string
	Salt         string
	Hash         string
	Permissions  []string
	DeviceTokens []persistedDeviceTokenStruct
}

/*
 * Data structure representing a user manager.
 */
type managerStruct struct {
	prng  io.Reader
	rex   *regexp.Regexp
	mutex sync.RWMutex
	users []userStruct
}

/*
 * A device token.
 */
type DeviceToken interface {
	CreationTime() time.Time
	Description() string
	Token() uint64
}

/*
 * A user manager.
 */
type Manager interface {
	AddPermission(name string, permission string) error
	CreateDeviceToken(name string, creationTime time.Time, description string) (DeviceToken, error)
	CreateUser(name string) error
	DeviceTokens(name string) ([]DeviceToken, error)
	Export() ([]byte, error)
	HasDeviceToken(name string, token uint64) (bool, error)
	Hash(name string) ([]byte, error)
	HasPermission(name string, permission string) (bool, error)
	Import(buf []byte) error
	Nonce(name string) ([LENGTH]byte, error)
	Permissions(name string) ([]string, error)
	RegenerateNonce(name string) error
	RemoveDeviceToken(name string, token uint64) error
	RemovePermission(name string, permission string) error
	RemoveUser(name string) error
	Salt(name string) ([LENGTH]byte, error)
	SetPassword(name string, password string) error
	UserExists(name string) bool
	Users() []string
}

/*
 * Returns the creation time of this device token.
 */
func (this *deviceTokenStruct) CreationTime() time.Time {
	t := this.creationTime
	return t
}

/*
 * Returns the description of this device token.
 */
func (this *deviceTokenStruct) Description() string {
	desc := this.description
	return desc
}

/*
 * Returns the token value of this device token.
 */
func (this *deviceTokenStruct) Token() uint64 {
	token := this.token
	return token
}

/*
 * Determines whether a user has a specific device token and returns its index.
 */
func (this *managerStruct) findDeviceToken(userId int, value uint64) int {
	users := this.users
	numUsers := len(users)
	idx := int(-1)

	/*
	 * Check whether user ID is in range.
	 */
	if userId < numUsers {
		user := users[userId]
		deviceTokens := user.deviceTokens

		/*
		 * Iterate over all device fokens.
		 */
		for i, token := range deviceTokens {
			t := token.token

			/*
			 * If value was found, save its index.
			 */
			if t == value {
				idx = i
			}

		}

	}

	return idx
}

/*
 * Determines the user id of a user, i. e. its position in the user slice.
 */
func (this *managerStruct) getUserId(name string) int {
	users := this.users
	foundId := -1

	/*
	 * Iterate over all users.
	 */
	for id, user := range users {
		userName := user.name

		/*
		 * Check if we have a user with the given name.
		 */
		if userName == name {
			foundId = id
		}

	}

	return foundId
}

/*
 * Determines whether a user has a specific device token.
 */
func (this *managerStruct) hasDeviceToken(userId int, value uint64) bool {
	idx := this.findDeviceToken(userId, value)
	found := (idx != -1)
	return found
}

/*
 * Adds a permission to a user.
 */
func (this *managerStruct) AddPermission(name string, permission string) error {
	errResult := error(nil)
	this.mutex.Lock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		errResult = fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		permissions := user.permissions
		exists := false

		/*
		 * Check if user already has permission.
		 */
		for _, currentPermission := range permissions {

			/*
			 * Check for permission.
			 */
			if currentPermission == permission {
				exists = true
			}

		}

		/*
		 * Add permission to user if he / she does not already have it.
		 */
		if !exists {
			user.permissions = append(permissions, permission)
		}

		users[id] = user
	}

	this.mutex.Unlock()
	return errResult
}

/*
 * Creates a new device token for a user.
 */
func (this *managerStruct) CreateDeviceToken(name string, creationTime time.Time, description string) (DeviceToken, error) {
	result := DeviceToken(nil)
	errResult := error(nil)
	this.mutex.Lock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		errResult = fmt.Errorf("User '%s' does not exist.", name)
	} else {
		token := make([]byte, SIZE_TOKEN)
		prng := this.prng
		numBytes, err := prng.Read(token)

		/*
		 * Check if token was successfully created.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to create device token for user '%s': %s", name, msg)
		} else if numBytes != SIZE_TOKEN {
			errResult = fmt.Errorf("Failed to create device token for user '%s': Incorrect number of bytes read from PRNG: Expected %d, got %d.", name, SIZE_TOKEN, numBytes)
		} else {
			endian := binary.BigEndian
			tokenValue := endian.Uint64(token)
			collision := this.hasDeviceToken(id, tokenValue)

			/*
			 * Keep generating until collision is resolved or an error occurs.
			 */
			for collision && (errResult == nil) {
				numBytes, err = prng.Read(token)

				/*
				 * Check if token was successfully created.
				 */
				if err != nil {
					msg := err.Error()
					errResult = fmt.Errorf("Failed to create device token for user '%s': %s", name, msg)
				} else if numBytes != SIZE_TOKEN {
					errResult = fmt.Errorf("Failed to create device token for user '%s': Incorrect number of bytes read from PRNG: Expected %d, got %d.", name, SIZE_TOKEN, numBytes)
				}

				collision = this.hasDeviceToken(id, tokenValue)
			}

			/*
			 * Create device token.
			 */
			deviceToken := deviceTokenStruct{
				creationTime: creationTime,
				description:  description,
				token:        tokenValue,
			}

			deviceTokens := this.users[id].deviceTokens
			deviceTokens = append(deviceTokens, deviceToken)
			this.users[id].deviceTokens = deviceTokens
			result = &deviceToken
		}

	}

	this.mutex.Unlock()
	return result, errResult
}

/*
 * Creates a new user.
 */
func (this *managerStruct) CreateUser(name string) error {
	errResult := error(nil)
	length := utf8.RuneCountInString(name)

	/*
	 * Check if username is of invalid length.
	 */
	if (length < UNAME_L_LIMIT) || (length > UNAME_U_LIMIT) {
		errResult = fmt.Errorf("Username must be at least %d characters and at most %d characters long.", UNAME_L_LIMIT, UNAME_U_LIMIT)
	} else {
		rex := this.rex
		match := rex.MatchString(name)

		/*
		 * Check if username matches regular expression.
		 */
		if !match {
			rexString := rex.String()
			errResult = fmt.Errorf("Username '%s' does not match regular expression '%s'.", name, rexString)
		} else {
			this.mutex.Lock()
			id := this.getUserId(name)

			/*
			 * Check if we have a user with the name provided to us.
			 */
			if id >= 0 {
				errResult = fmt.Errorf("User '%s' already exists.", name)
			} else {
				permissions := []string{}

				/*
				 * Create new user.
				 */
				userNew := userStruct{
					name:        name,
					permissions: permissions,
				}

				users := this.users
				users = append(users, userNew)
				this.users = users
			}

			this.mutex.Unlock()
		}

	}

	return errResult
}

/*
 * Returns all device tokens associated with a user.
 */
func (this *managerStruct) DeviceTokens(name string) ([]DeviceToken, error) {
	result := []DeviceToken(nil)
	errResult := error(nil)
	this.mutex.RLock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		errResult = fmt.Errorf("User '%s' does not exist.", name)
	} else {
		user := this.users[id]
		deviceTokens := user.deviceTokens
		numDeviceTokens := len(deviceTokens)
		result = make([]DeviceToken, numDeviceTokens)

		/*
		 * Copy the device tokens.
		 */
		for i, deviceToken := range deviceTokens {
			// Without this assignment, all pointers point to the same value.
			deviceTokenCopy := deviceToken
			result[i] = &deviceTokenCopy
		}

	}

	this.mutex.RUnlock()
	return result, errResult
}

/*
 * Export all users to JSON representation.
 */
func (this *managerStruct) Export() ([]byte, error) {
	this.mutex.RLock()
	users := this.users
	numUsers := len(users)
	persistedUsers := make([]persistedUserStruct, numUsers)
	encoding := base64.StdEncoding

	/*
	 * Iterate over all users and persist them.
	 */
	for i, user := range users {
		userName := user.name
		salt := user.salt[:]
		saltString := encoding.EncodeToString(salt)
		hash := user.hash
		hashString := ""

		/*
		 * An unset hash is encoded into an empty string.
		 */
		if hash != nil {
			hashString = encoding.EncodeToString(hash)
		}

		permissions := user.permissions
		numPermissions := len(permissions)
		permissionCopy := make([]string, numPermissions)
		copy(permissionCopy, permissions)
		deviceTokens := user.deviceTokens
		numDeviceTokens := len(deviceTokens)
		persistedDeviceTokens := make([]persistedDeviceTokenStruct, numDeviceTokens)

		/*
		 * Iterate over all device tokens and persist them.
		 */
		for j, deviceToken := range deviceTokens {
			creationTime := deviceToken.creationTime
			timeFormat := time.RFC3339
			creationTimeString := creationTime.Format(timeFormat)
			descriptionString := deviceToken.description
			token := deviceToken.token
			tokenString := fmt.Sprintf("%016x", token)

			/*
			 * Create persisted device token.
			 */
			persistedDeviceToken := persistedDeviceTokenStruct{
				CreationTime: creationTimeString,
				Description:  descriptionString,
				Token:        tokenString,
			}

			persistedDeviceTokens[j] = persistedDeviceToken
		}

		/*
		 * Create persisted user.
		 */
		persistedUser := persistedUserStruct{
			Name:         userName,
			Salt:         saltString,
			Hash:         hashString,
			Permissions:  permissionCopy,
			DeviceTokens: persistedDeviceTokens,
		}

		persistedUsers[i] = persistedUser

	}

	this.mutex.RUnlock()
	buf, err := json.MarshalIndent(persistedUsers, "", "\t")

	/*
	 * Check if serialization failed.
	 */
	if err != nil {
		msg := err.Error()
		return nil, fmt.Errorf("Failed to export users: %s", msg)
	} else {
		return buf, nil
	}

}

/*
 * Returns whether a user has a certain device token associated.
 */
func (this *managerStruct) HasDeviceToken(name string, token uint64) (bool, error) {
	result := false
	errResult := error(nil)
	this.mutex.RLock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		errResult = fmt.Errorf("User '%s' does not exist.", name)
	} else {
		result = this.hasDeviceToken(id, token)
	}

	this.mutex.RUnlock()
	return result, errResult
}

/*
 * Returns the password hash of a user.
 *
 * The hash may be nil when no password is set.
 */
func (this *managerStruct) Hash(name string) ([]byte, error) {
	result := []byte(nil)
	errResult := error(nil)
	this.mutex.RLock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		errResult = fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		hash := user.hash
		hashSize := len(hash)
		result = make([]byte, hashSize)
		copy(result, hash)
	}

	this.mutex.RUnlock()
	return result, errResult
}

/*
 * Returns whether a user has a certain permission.
 */
func (this *managerStruct) HasPermission(name string, permission string) (bool, error) {
	result := false
	errResult := error(nil)
	this.mutex.RLock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		errResult = fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		permissions := user.permissions

		/*
		 * Iterate over all permissions of the user.
		 */
		for _, currentPermission := range permissions {

			/*
			 * Check for requested permission.
			 */
			if currentPermission == permission {
				result = true
			}

		}

	}

	return result, errResult
}

/*
 * Imports all users from JSON representation.
 */
func (this *managerStruct) Import(buf []byte) error {
	persistedUsers := []persistedUserStruct{}
	encoding := base64.StdEncoding
	err := json.Unmarshal(buf, &persistedUsers)

	/*
	 * Check if unmarshalling was succesful.
	 */
	if err != nil {
		return fmt.Errorf("%s", "Failed to import users.")
	} else {
		numUsers := len(persistedUsers)
		users := make([]userStruct, numUsers)

		/*
		 * Iterate over all users and make them usable from their persisted state.
		 */
		for i, persistedUser := range persistedUsers {
			userName := persistedUser.Name
			userNameLength := utf8.RuneCountInString(userName)
			saltPersisted := persistedUser.Salt
			hashPersisted := persistedUser.Hash
			salt, errSalt := encoding.DecodeString(saltPersisted)
			saltSize := len(salt)
			hash, errHash := encoding.DecodeString(hashPersisted)
			hashSize := len(hash)
			persistedPermissions := persistedUser.Permissions
			persistedDeviceTokens := persistedUser.DeviceTokens

			/*
			 * Check for pathological cases.
			 */
			if (userNameLength < UNAME_L_LIMIT) || (userNameLength > UNAME_U_LIMIT) {
				return fmt.Errorf("User name '%s' is %d characters long, which is not in the interval [%d, %d].", userName, userNameLength, UNAME_L_LIMIT, UNAME_U_LIMIT)
			} else if errSalt != nil {
				return fmt.Errorf("Failed to decode password salt for user '%s'.", userName)
			} else if saltSize != LENGTH {
				return fmt.Errorf("Password salt of user '%s' has incorrect size. Expected %d bytes, found %d bytes.", userName, LENGTH, saltSize)
			} else if errHash != nil {
				return fmt.Errorf("Failed to decode password hash for user '%s'.", userName)
			} else if hashSize != 0 && hashSize != LENGTH {
				return fmt.Errorf("Password hash of user '%s' has incorrect size. Expected either 0 or %d bytes, found %d bytes.", userName, LENGTH, hashSize)
			} else {
				numPermissions := len(persistedPermissions)
				permissionsImported := make([]string, numPermissions)
				copy(permissionsImported, persistedPermissions)
				numDeviceTokens := len(persistedDeviceTokens)
				deviceTokensImported := make([]deviceTokenStruct, numDeviceTokens)

				/*
				 * Iterate over all device tokens and make them usable from their persisted state.
				 */
				for j, persistedDeviceToken := range persistedDeviceTokens {
					creationTimeString := persistedDeviceToken.CreationTime
					creationTimeValue, errCreationTime := time.ParseInLocation(time.RFC3339, creationTimeString, time.UTC)
					descriptionValue := persistedDeviceToken.Description
					tokenString := persistedDeviceToken.Token
					tokenValue, errToken := strconv.ParseUint(tokenString, BASE_HEX, 64)

					/*
					 * Check if creation time and token could be parsed.
					 */
					if errCreationTime != nil {
						return fmt.Errorf("Failed to parse creation time of device token %d for user '%s'.", j, userName)
					} else if errToken != nil {
						return fmt.Errorf("Failed to parse token value of device token %d for user '%s'.", j, userName)
					} else {

						/*
						 * Create device token.
						 */
						deviceToken := deviceTokenStruct{
							creationTime: creationTimeValue,
							description:  descriptionValue,
							token:        tokenValue,
						}

						deviceTokensImported[j] = deviceToken
					}

				}

				/*
				 * Create imported user.
				 */
				user := userStruct{
					name:         userName,
					permissions:  permissionsImported,
					deviceTokens: deviceTokensImported,
				}

				copy(user.salt[:], salt)

				/*
				 * If hash is not of zero length, initialize user hash.
				 */
				if hashSize != 0 {
					hashCopy := make([]byte, hashSize)
					copy(hashCopy, hash)
					user.hash = hashCopy
				}

				prng := this.prng
				numBytes, err := prng.Read(user.nonce[:])

				/*
				 * Check if nonce could be generated.
				 */
				if err != nil {
					msg := err.Error()
					return fmt.Errorf("Failed to generate nonce for user '%s': %s", userName, msg)
				} else if numBytes != LENGTH {
					return fmt.Errorf("Failed to generate nonce for user '%s': Incorrect number of bytes read from PRNG: Expected %d, got %d.", userName, LENGTH, numBytes)
				}

				users[i] = user
			}

		}

		this.mutex.Lock()
		this.users = users
		this.mutex.Unlock()
		return nil
	}

}

/*
 * Returns a nonce for a user.
 */
func (this *managerStruct) Nonce(name string) ([LENGTH]byte, error) {
	result := [LENGTH]byte{}
	errResult := error(nil)
	this.mutex.RLock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		errResult = fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		result = user.nonce
	}

	this.mutex.RUnlock()
	return result, errResult
}

/*
 * Returns permissions of a user.
 */
func (this *managerStruct) Permissions(name string) ([]string, error) {
	result := []string(nil)
	errResult := error(nil)
	this.mutex.RLock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		errResult = fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		permissions := user.permissions
		numPermissions := len(permissions)
		result := make([]string, numPermissions)
		copy(result, permissions)
	}

	this.mutex.RUnlock()
	return result, errResult
}

/*
 * Generates a new nonce for a user.
 *
 * This shall be invoked after successful authentication.
 */
func (this *managerStruct) RegenerateNonce(name string) error {
	errResult := error(nil)
	this.mutex.Lock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		errResult = fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		prng := this.prng
		numBytes, err := prng.Read(user.nonce[:])

		/*
		 * Check if nonce was successfully created.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to update nonce for user '%s': %s", name, msg)
		} else if numBytes != LENGTH {
			errResult = fmt.Errorf("Failed to update nonce for user '%s': Incorrect number of bytes read from PRNG: Expected %d, got %d.", name, LENGTH, numBytes)
		} else {
			users[id] = user
		}

	}

	this.mutex.Unlock()
	return errResult
}

/*
 * Removes a device token from a user.
 */
func (this *managerStruct) RemoveDeviceToken(name string, token uint64) error {
	errResult := error(nil)
	this.mutex.Lock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		errResult = fmt.Errorf("User '%s' does not exist.", name)
	} else {
		idx := this.findDeviceToken(id, token)

		/*
		 * Check if that user has the provided token associated.
		 */
		if idx < 0 {
			errResult = fmt.Errorf("User '%s' does not have token %016x.", name, token)
		} else {
			idxInc := idx + 1
			user := this.users[id]
			deviceTokens := user.deviceTokens
			deviceTokens = append(deviceTokens[:idx], deviceTokens[idxInc:]...)
			user.deviceTokens = deviceTokens
			this.users[id] = user
		}

	}

	this.mutex.Unlock()
	return errResult
}

/*
 * Revokes a permission from a user.
 */
func (this *managerStruct) RemovePermission(name string, permission string) error {
	errResult := error(nil)
	this.mutex.Lock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		errResult = fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		permissions := user.permissions
		idx := -1

		/*
		 * Iterate over all permissions of the user.
		 */
		for i, currentPermission := range permissions {

			/*
			 * Check if we found the right permission.
			 */
			if currentPermission == permission {
				idx = i
			}

		}

		/*
		 * Check if we found the permission.
		 */
		if idx < 0 {
			errResult = fmt.Errorf("User '%s' does not have permission '%s'.", name, permission)
		} else {
			idxInc := idx + 1
			permissions = append(permissions[:idx], permissions[idxInc:]...)
			user.permissions = permissions
			users[id] = user
		}

	}

	this.mutex.Unlock()
	return errResult
}

/*
 * Removes an existing user.
 */
func (this *managerStruct) RemoveUser(name string) error {
	errResult := error(nil)
	this.mutex.Lock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		errResult = fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		idInc := id + 1
		users = append(users[:id], users[idInc:]...)
		this.users = users
	}

	this.mutex.Unlock()
	return errResult
}

/*
 * Returns the salt of a user.
 */
func (this *managerStruct) Salt(name string) ([LENGTH]byte, error) {
	result := [LENGTH]byte{}
	errResult := error(nil)
	this.mutex.RLock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		errResult = fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		result = user.salt
	}

	this.mutex.RUnlock()
	return result, errResult
}

/*
 * Changes the password of a user.
 */
func (this *managerStruct) SetPassword(name string, password string) error {
	this.mutex.Lock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with this ID.
	 */
	if id < 0 {
		this.mutex.Unlock()
		return fmt.Errorf("User '%s' does not exist.", name)
	} else {
		prng := this.prng
		salt := make([]byte, LENGTH)
		numBytes, err := prng.Read(salt)

		/*
		 * Check if salt was generated.
		 */
		if err != nil {
			this.mutex.Unlock()
			return fmt.Errorf("Failed to generate salt for user '%s' password change.", name)
		} else if numBytes != LENGTH {
			this.mutex.Unlock()
			return fmt.Errorf("Failed to generate salt for user '%s': Incorrect number of bytes read from PRNG: Expected %d, got %d.", name, LENGTH, numBytes)
		} else {
			pwdBytes := []byte(password)
			pwdHash := sha512.Sum512(pwdBytes)
			saltAndHash := append(salt[:], pwdHash[:]...)
			users := this.users
			finalHash := sha512.Sum512(saltAndHash)
			users[id].hash = finalHash[:]
			copy(users[id].salt[:], salt)
			this.mutex.Unlock()
			return nil
		}

	}

}

/*
 * Finds out, if a user exists.
 */
func (this *managerStruct) UserExists(name string) bool {
	this.mutex.RLock()
	id := this.getUserId(name)
	this.mutex.RUnlock()
	exists := id >= 0
	return exists
}

/*
 * Returns the names of all registered users.
 */
func (this *managerStruct) Users() []string {
	this.mutex.RLock()
	users := this.users
	numUsers := len(users)
	userNames := make([]string, numUsers)

	/*
	 * Iterate over all users.
	 */
	for i, user := range users {
		userName := user.name
		userNames[i] = userName
	}

	this.mutex.RUnlock()
	return userNames
}

/*
 * Creates a new user manager.
 */
func CreateManager(prng io.Reader) (Manager, error) {

	/*
	 * Check if random number generator was provided.
	 */
	if prng == nil {
		return nil, fmt.Errorf("%s", "PRNG must not be nil!")
	} else {
		users := []userStruct{}
		rex, err := regexp.Compile(UNAME_REX)

		/*
		 * Check if regular expression could be compiled.
		 */
		if err != nil {
			return nil, fmt.Errorf("Regular expression '%s' failed to compile.", UNAME_REX)
		} else {

			/*
			 * Create user manager.
			 */
			ms := managerStruct{
				prng:  prng,
				users: users,
				rex:   rex,
			}

			return &ms, nil
		}

	}

}
