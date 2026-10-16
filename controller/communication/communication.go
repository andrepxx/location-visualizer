package communication

/*
 * Indicates whether a request was successful or not.
 */
type WebResponseStruct struct {
	Success bool
	Reason  string
}

/*
 * Represents some boolean result, for example whether a paricular user has a
 * particular device token or permission assigned.
 */
type WebBooleanResultStruct struct {
	Success bool
	Reason  string
	Value   bool
}

/*
 * Web representation of a device token.
 */
type WebDeviceTokenStruct struct {
	CreationTime string
	Description  string
	Token        string
}

/*
 * Web representation of a list of device tokens.
 */
type WebDeviceTokensStruct struct {
	WebResponseStruct
	DeviceTokens []WebDeviceTokenStruct
}

/*
 * Web representation of a list of permissions.
 */
type WebPermissionsStruct struct {
	WebResponseStruct
	Permissions []string
}

/*
 * Web representation of a public key.
 */
type WebPublicKeyStruct struct {
	CreationTime   string
	Description    string
	Hash           string
	Representation string
}

/*
 * Web representation of a list of public keys.
 */
type WebPublicKeysStruct struct {
	WebResponseStruct
	PublicKeys []WebPublicKeyStruct
}

/*
 * Web representation of a public key PEM data.
 */
type WebPublicKeyPemDataStruct struct {
	WebResponseStruct
	PemData string
}
