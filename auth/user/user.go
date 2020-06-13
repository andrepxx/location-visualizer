package user

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sync"
	"unicode/utf8"
)

/*
 * Global constants.
 */
const (
	LENGTH        = 64
	UNAME_L_LIMIT = 3
	UNAME_U_LIMIT = 16
	UNAME_REX     = "^[A-Za-z0-9\\-_\\.]+$"
)

/*
 * A user, as represented in memory.
 */
type userStruct struct {
	name        string
	salt        [LENGTH]byte
	hash        []byte
	nonce       [LENGTH]byte
	permissions []string
}

/*
 * A user, as represented on disk.
 */
type persistedUserStruct struct {
	Name        string
	Salt        string
	Hash        string
	Permissions []string
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
 * A user manager.
 */
type Manager interface {
	AddPermission(name string, permission string) error
	CreateUser(name string) error
	Export() ([]byte, error)
	Hash(name string) ([]byte, error)
	HasPermission(name string, permission string) (bool, error)
	Import(buf []byte) error
	Nonce(name string) ([LENGTH]byte, error)
	Permissions(name string) ([]string, error)
	RegenerateNonce(name string) error
	RemovePermission(name string, permission string) error
	RemoveUser(name string) error
	Salt(name string) ([LENGTH]byte, error)
	SetPassword(name string, password string) error
	UserExists(name string) bool
	Users() []string
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
 * Adds a permission to a user.
 */
func (this *managerStruct) AddPermission(name string, permission string) error {
	this.mutex.Lock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		this.mutex.Unlock()
		return fmt.Errorf("User '%s' does not exist.", name)
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
		this.mutex.Unlock()
		return nil
	}

}

/*
 * Creates a new user.
 */
func (this *managerStruct) CreateUser(name string) error {
	length := utf8.RuneCountInString(name)

	/*
	 * Check if username is of invalid length.
	 */
	if (length < UNAME_L_LIMIT) || (length > UNAME_U_LIMIT) {
		return fmt.Errorf("Username must be at least %d characters and at most %d characters long.", UNAME_L_LIMIT, UNAME_U_LIMIT)
	} else {
		rex := this.rex
		match := rex.MatchString(name)

		/*
		 * Check if username matches regular expression.
		 */
		if !match {
			rexString := rex.String()
			return fmt.Errorf("Username '%s' does not match regular expression '%s'.", name, rexString)
		} else {
			this.mutex.RLock()
			exists := this.UserExists(name)
			this.mutex.RUnlock()

			/*
			 * Check if we have a user with the name provided to us.
			 */
			if exists {
				return fmt.Errorf("User '%s' already exists.", name)
			} else {
				permissions := []string{}

				/*
				 * Create new user.
				 */
				userNew := userStruct{
					name:        name,
					permissions: permissions,
				}

				this.mutex.Lock()
				users := this.users
				users = append(users, userNew)
				this.users = users
				this.mutex.Unlock()
				return nil
			}

		}

	}

}

/*
 * Export all users to JSON representation.
 */
func (this *managerStruct) Export() ([]byte, error) {
	this.mutex.RLock()
	users := this.users
	p_users := []persistedUserStruct{}
	encoding := base64.StdEncoding

	/*
	 * Iterate over all users and persist them.
	 */
	for _, user := range users {
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
		permissionCount := len(permissions)
		permissionCopy := make([]string, permissionCount)
		copy(permissionCopy, permissions)

		/*
		 * Create persisted user.
		 */
		p_user := persistedUserStruct{
			Name:        userName,
			Salt:        saltString,
			Hash:        hashString,
			Permissions: permissionCopy,
		}

		p_users = append(p_users, p_user)
	}

	this.mutex.RUnlock()
	buf, err := json.MarshalIndent(p_users, "", "\t")

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
 * Returns the password hash of a user.
 *
 * The hash may be nil when no password is set.
 */
func (this *managerStruct) Hash(name string) ([]byte, error) {
	this.mutex.RLock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		this.mutex.RUnlock()
		return nil, fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		hash := user.hash
		hashSize := len(hash)
		hashCopy := make([]byte, hashSize)
		copy(hashCopy, hash)
		this.mutex.RUnlock()
		return hashCopy, nil
	}

}

/*
 * Returns whether a user has a certain permission.
 */
func (this *managerStruct) HasPermission(name string, permission string) (bool, error) {
	this.mutex.RLock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		this.mutex.RUnlock()
		return false, fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		permissions := user.permissions
		exists := false

		/*
		 * Iterate over all permissions of the user.
		 */
		for _, currentPermission := range permissions {

			/*
			 * Check for requested permission.
			 */
			if currentPermission == permission {
				exists = true
			}

		}

		this.mutex.RUnlock()
		return exists, nil
	}

}

/*
 * Imports all users from JSON representation.
 */
func (this *managerStruct) Import(buf []byte) error {
	persistentUsers := []persistedUserStruct{}
	encoding := base64.StdEncoding
	err := json.Unmarshal(buf, &persistentUsers)

	/*
	 * Check if unmarshalling was succesful.
	 */
	if err != nil {
		return fmt.Errorf("%s", "Failed to import users.")
	} else {
		users := []userStruct{}

		/*
		 * Iterate over all users and make them usable from their persisted state.
		 */
		for _, persistentUser := range persistentUsers {
			userName := persistentUser.Name
			userNameLength := utf8.RuneCountInString(userName)
			saltPersistent := persistentUser.Salt
			hashPersistent := persistentUser.Hash
			permissionsPersistent := persistentUser.Permissions
			salt, errSalt := encoding.DecodeString(saltPersistent)
			saltSize := len(salt)
			hash, errHash := encoding.DecodeString(hashPersistent)
			hashSize := len(hash)

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
				numPermissions := len(permissionsPersistent)
				permissionsCopy := make([]string, numPermissions)
				copy(permissionsCopy, permissionsPersistent)

				/*
				 * Create imported user.
				 */
				user := userStruct{
					name:        userName,
					permissions: permissionsCopy,
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

				users = append(users, user)
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
	this.mutex.RLock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		this.mutex.RUnlock()
		return [LENGTH]byte{}, fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		nonce := user.nonce
		this.mutex.RUnlock()
		return nonce, nil
	}

}

/*
 * Returns permissions of a user.
 */
func (this *managerStruct) Permissions(name string) ([]string, error) {
	this.mutex.RLock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		this.mutex.RUnlock()
		return nil, fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		permissions := user.permissions
		numPermissions := len(permissions)
		permissionsCopy := make([]string, numPermissions)
		copy(permissionsCopy, permissions)
		this.mutex.RUnlock()
		return permissionsCopy, nil
	}

}

/*
 * Generates a new nonce for a user.
 *
 * This shall be invoked after successful authentication.
 */
func (this *managerStruct) RegenerateNonce(name string) error {
	this.mutex.Lock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		this.mutex.Unlock()
		return fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		prng := this.prng
		numBytes, err := prng.Read(user.nonce[:])

		/*
		 * Check if nonce was successfully created.
		 */
		if err != nil {
			this.mutex.Unlock()
			msg := err.Error()
			return fmt.Errorf("Failed to update nonce for user '%s': %s", name, msg)
		} else if numBytes != LENGTH {
			this.mutex.Unlock()
			return fmt.Errorf("Failed to update nonce for user '%s': Incorrect number of bytes read from PRNG: Expected %d, got %d.", name, LENGTH, numBytes)
		} else {
			users[id] = user
			this.mutex.Unlock()
			return nil
		}

	}

}

/*
 * Revokes a permission from a user.
 */
func (this *managerStruct) RemovePermission(name string, permission string) error {
	this.mutex.Lock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		this.mutex.Unlock()
		return fmt.Errorf("User '%s' does not exist.", name)
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
			this.mutex.Unlock()
			return fmt.Errorf("User '%s' does not have permission '%s'.", name, permission)
		} else {
			idxInc := idx + 1
			permissions = append(permissions[:idx], permissions[idxInc:]...)
			user.permissions = permissions
			users[id] = user
			this.mutex.Unlock()
			return nil
		}

	}

}

/*
 * Removes an existing user.
 */
func (this *managerStruct) RemoveUser(name string) error {
	this.mutex.Lock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		this.mutex.Unlock()
		return fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		idInc := id + 1
		users = append(users[:id], users[idInc:]...)
		this.users = users
		this.mutex.Unlock()
		return nil
	}

}

/*
 * Returns the salt of a user.
 */
func (this *managerStruct) Salt(name string) ([LENGTH]byte, error) {
	this.mutex.RLock()
	id := this.getUserId(name)

	/*
	 * Check if we have a user with the name provided to us.
	 */
	if id < 0 {
		this.mutex.RUnlock()
		return [LENGTH]byte{}, fmt.Errorf("User '%s' does not exist.", name)
	} else {
		users := this.users
		user := users[id]
		salt := user.salt
		this.mutex.RUnlock()
		return salt, nil
	}

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
