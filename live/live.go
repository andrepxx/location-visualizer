package live

import (
	"fmt"
	"sync"
	"time"
)

const (
	RFC3339_MILLI     = "2006-01-02T15:04:05.000Z07:00"
	SCALING_FACTOR_E7 = 10000000
)

/*
 * Latitude in degrees, scaled by 10^7.
 */
type Latitude int32

/*
 * Longitude in degrees, scaled by 10^7.
 */
type Longitude int32

/*
 * Timestamp in milliseconds since the EPOCH.
 */
type Timestamp uint64

/*
 * Data structure representing a GNSS position.
 */
type positionStruct struct {
	timestamp Timestamp
	latitude  Latitude
	longitude Longitude
}

/*
 * Data structure representing a device.
 */
type deviceStruct struct {
	description string
	token       uint64
	mutex       sync.RWMutex
	hasPosition bool
	position    positionStruct
}

/*
 * Data structure representing a user.
 */
type userStruct struct {
	name        string
	mutex       sync.RWMutex
	deviceIndex map[uint64]int
	devices     []*deviceStruct
}

/*
 * Data structure representing the position manager storing live positions of
 * devices.
 */
type positionManagerStruct struct {
	mutex     sync.RWMutex
	userIndex map[string]int
	users     []*userStruct
}

/*
 * A geographic position, including a time stamp.
 */
type Position interface {
	Latitude() Latitude
	Longitude() Longitude
	Timestamp() Timestamp
}

/*
 * A device owned by a user.
 */
type Device interface {
	Description() string
	Position() Position
	SetPosition(timestampMS uint64, latitudeE7 int32, longitudeE7 int32)
	Token() uint64
}

/*
 * A user holding devices.
 */
type User interface {
	AddDevice(token uint64, description string) error
	Device(token uint64) (Device, error)
	Devices() []Device
	Name() string
	RemoveDevice(token uint64) error
}

/*
 * A position manager storing live positions of devices.
 */
type PositionManager interface {
	AddUser(name string) error
	RemoveUser(name string) error
	User(name string) (User, error)
	Users() []User
}

/*
 * Returns this latitude value in E7 degrees representation.
 */
func (this Latitude) DegreesE7() int64 {
	return int64(this)
}

/*
 * Returns a string representation of this latitude value.
 */
func (this Latitude) String() string {
	value := int64(this)
	suffix := "N"

	/*
	 * Check if value is negative.
	 */
	if value < 0 {
		suffix = "S"
		value = -value
	}

	integerPart := value / SCALING_FACTOR_E7
	fractionalPart := value % SCALING_FACTOR_E7
	result := fmt.Sprintf("%d.%07d%s", integerPart, fractionalPart, suffix)
	return result
}

/*
 * Returns this longitude value in E7 degrees representation.
 */
func (this Longitude) DegreesE7() int64 {
	return int64(this)
}

/*
 * Returns a string representation of this longitude value.
 */
func (this Longitude) String() string {
	value := int64(this)
	suffix := "E"

	/*
	 * Check if value is negative.
	 */
	if value < 0 {
		suffix = "W"
		value = -value
	}

	integerPart := value / SCALING_FACTOR_E7
	fractionalPart := value % SCALING_FACTOR_E7
	result := fmt.Sprintf("%d.%07d%s", integerPart, fractionalPart, suffix)
	return result
}

/*
 * Returns a string representation (in RFC 3339 representation) of this
 * timestamp.
 */
func (this Timestamp) String() string {
	value := int64(this)
	t := time.UnixMilli(value)
	utc := t.UTC()
	result := utc.Format(RFC3339_MILLI)
	return result
}

/*
 * Returns the latitude associated with this position.
 *
 * The int32 representation is a value in degrees, scaled by 10^7.
 */
func (this *positionStruct) Latitude() Latitude {
	latitude := this.latitude
	return latitude
}

/*
 * Returns the longitude, scaled by 10^7, associated with this position.
 *
 * The int32 representation is a value in degrees, scaled by 10^7.
 */
func (this *positionStruct) Longitude() Longitude {
	longitude := this.longitude
	return longitude
}

/*
 * Returns the timestamp associated with this position.
 *
 * The uint64 representation is in milliseconds since the Epoch,
 */
func (this *positionStruct) Timestamp() Timestamp {
	timestamp := this.timestamp
	return timestamp
}

/*
 * Return the description for this device.
 */
func (this *deviceStruct) Description() string {
	description := this.description
	return description
}

/*
 * Returns the current position of this device.
 */
func (this *deviceStruct) Position() Position {
	result := Position(nil)
	this.mutex.RLock()
	hasPosition := this.hasPosition

	/*
	 * Check whether we have a valid position.
	 */
	if hasPosition {
		position := this.position
		result = &position
	}

	this.mutex.RUnlock()
	return result
}

/*
 * Set the position of this device to the provided values.
 */
func (this *deviceStruct) SetPosition(timestampMS uint64, latitudeE7 int32, longitudeE7 int32) {
	timestamp := Timestamp(timestampMS)
	latitude := Latitude(latitudeE7)
	longitude := Longitude(longitudeE7)

	/*
	 * Create a new position.
	 */
	position := positionStruct{
		timestamp: timestamp,
		latitude:  latitude,
		longitude: longitude,
	}

	this.mutex.Lock()
	this.hasPosition = true
	this.position = position
	this.mutex.Unlock()
}

/*
 * Returns the device token associated with this device.
 */
func (this *deviceStruct) Token() uint64 {
	token := this.token
	return token
}

/*
 * Update the device index.
 *
 * Assumes that the caller holds a write-lock.
 */
func (this *userStruct) updateDeviceIndex() {
	deviceIndex := make(map[uint64]int)
	devices := this.devices

	/*
	 * Iterate over all devices.
	 */
	for i, device := range devices {
		token := device.token
		deviceIndex[token] = i
	}

	this.deviceIndex = deviceIndex
}

/*
 * Add the device with the provided token and description to the user.
 */
func (this *userStruct) AddDevice(token uint64, description string) error {
	result := error(nil)
	this.mutex.Lock()
	_, exists := this.deviceIndex[token]

	/*
	 * Device must not exist.
	 */
	if exists {
		result = fmt.Errorf("%s", "Device already exists")
	} else {

		/*
		 * Create a new device.
		 */
		device := &deviceStruct{
			description: description,
			token:       token,
		}

		devices := this.devices
		idx := len(devices)
		devices = append(devices, device)
		this.devices = devices
		this.deviceIndex[token] = idx
	}

	this.mutex.Unlock()
	return result
}

/*
 * Returns the device with the provided token or an error in case the device is
 * not registered to this user.
 */
func (this *userStruct) Device(token uint64) (Device, error) {
	result := (Device)(nil)
	errResult := error(nil)
	this.mutex.RLock()
	deviceIndex := this.deviceIndex
	idx, exists := deviceIndex[token]

	/*
	 * Check whether a device with this token exists.
	 */
	if exists {
		devices := this.devices
		result = devices[idx]
	} else {
		errResult = fmt.Errorf("%s", "Device does not exist.")
	}

	this.mutex.RUnlock()
	return result, errResult
}

/*
 * Returns all devices associated with this user.
 */
func (this *userStruct) Devices() []Device {
	result := []Device(nil)
	this.mutex.RLock()
	devices := this.devices
	numDevices := len(devices)
	result = make([]Device, numDevices)

	/*
	 * Can't use copy function here due to type conversion.
	 */
	for i, device := range devices {
		result[i] = device
	}

	this.mutex.RUnlock()
	return result
}

/*
 * Returns the name of this user.
 */
func (this *userStruct) Name() string {
	name := this.name
	return name
}

/*
 * Remove the device with the provided token from this user.
 */
func (this *userStruct) RemoveDevice(token uint64) error {
	result := error(nil)
	this.mutex.Lock()
	deviceIndex := this.deviceIndex
	idx, exists := deviceIndex[token]

	/*
	 * Check whether a device with this token exists.
	 */
	if exists {
		idxInc := idx + 1
		devices := this.devices
		devices = append(devices[:idx], devices[idxInc:]...)
		this.devices = devices
		this.updateDeviceIndex()
	} else {
		result = fmt.Errorf("%s", "Device does not exist.")
	}

	this.mutex.Unlock()
	return result
}

/*
 * Update the user index.
 *
 * Assumes that the caller holds a write-lock.
 */
func (this *positionManagerStruct) updateUserIndex() {
	userIndex := make(map[string]int)
	users := this.users

	/*
	 * Iterate over all users.
	 */
	for i, user := range users {
		name := user.name
		userIndex[name] = i
	}

	this.userIndex = userIndex
}

/*
 * Add the user with the provided name to enable live positions to be stored
 * for this user.
 */
func (this *positionManagerStruct) AddUser(name string) error {
	result := error(nil)
	this.mutex.Lock()
	_, exists := this.userIndex[name]

	/*
	 * User must not exist.
	 */
	if exists {
		result = fmt.Errorf("%s", "User already exists")
	} else {
		deviceIndex := make(map[uint64]int)

		/*
		 * Create a new user.
		 */
		user := &userStruct{
			name:        name,
			deviceIndex: deviceIndex,
		}

		users := this.users
		idx := len(users)
		users = append(users, user)
		this.users = users
		this.userIndex[name] = idx
	}

	this.mutex.Unlock()
	return result
}

/*
 * Remove the user with the provided name.
 */
func (this *positionManagerStruct) RemoveUser(name string) error {
	result := error(nil)
	this.mutex.Lock()
	userIndex := this.userIndex
	idx, exists := userIndex[name]

	/*
	 * Check whether a user with this name exists.
	 */
	if exists {
		idxInc := idx + 1
		users := this.users
		users = append(users[:idx], users[idxInc:]...)
		this.users = users
		this.updateUserIndex()
	} else {
		result = fmt.Errorf("%s", "User does not exist.")
	}

	this.mutex.Unlock()
	return result
}

/*
 * Returns the user with the provided name to retrieve or store live positions.
 */
func (this *positionManagerStruct) User(name string) (User, error) {
	result := (User)(nil)
	errResult := error(nil)
	this.mutex.RLock()
	userIndex := this.userIndex
	idx, exists := userIndex[name]

	/*
	 * Check whether a user with this name exists.
	 */
	if exists {
		users := this.users
		result = users[idx]
	} else {
		errResult = fmt.Errorf("%s", "User does not exist.")
	}

	this.mutex.RUnlock()
	return result, errResult
}

/*
 * Returns all users for which live position information can be stored.
 */
func (this *positionManagerStruct) Users() []User {
	result := []User(nil)
	this.mutex.RLock()
	users := this.users
	numUsers := len(users)
	result = make([]User, numUsers)

	/*
	 * Can't use copy function here due to type conversion.
	 */
	for i, user := range users {
		result[i] = user
	}

	this.mutex.RUnlock()
	return result
}

/*
 * Creates a new position manager.
 */
func CreatePositionManager() PositionManager {
	userIndex := make(map[string]int)

	/*
	 * Create position manager.
	 */
	p := positionManagerStruct{
		userIndex: userIndex,
	}

	return &p
}
