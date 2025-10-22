package publickey

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
)

const (
	BLOCK_TYPE_INVALID           = "INVALID"
	BLOCK_TYPE_PRIVATE_KEY_PKCS1 = "RSA PRIVATE KEY"
	BLOCK_TYPE_PRIVATE_KEY_PKCS8 = "PRIVATE KEY"
	BLOCK_TYPE_PUBLIC_KEY_PKCS1  = "RSA PUBLIC KEY"
	BLOCK_TYPE_PUBLIC_KEY_PKIX   = "PUBLIC KEY"
)

const (
	REPRESENTATION_INVALID = Representation(iota - 1)
	REPRESENTATION_RSA_PRIVATE_KEY_PKCS1
	REPRESENTATION_RSA_PRIVATE_KEY_PKCS8
	REPRESENTATION_RSA_PUBLIC_KEY_PKCS1
	REPRESENTATION_RSA_PUBLIC_KEY_PKIX
)

type Representation int8

/*
 * Determines the representation of an RSA key from the PEM block type.
 */
func CreateRepresentation(blockType string) Representation {
	result := REPRESENTATION_INVALID

	/*
	 * Decide on PEM block type.
	 */
	switch blockType {
	case BLOCK_TYPE_PRIVATE_KEY_PKCS1:
		result = REPRESENTATION_RSA_PRIVATE_KEY_PKCS1
	case BLOCK_TYPE_PRIVATE_KEY_PKCS8:
		result = REPRESENTATION_RSA_PRIVATE_KEY_PKCS8
	case BLOCK_TYPE_PUBLIC_KEY_PKCS1:
		result = REPRESENTATION_RSA_PUBLIC_KEY_PKCS1
	case BLOCK_TYPE_PUBLIC_KEY_PKIX:
		result = REPRESENTATION_RSA_PUBLIC_KEY_PKIX
	}

	return result
}

/*
 * Returns the PEM block type corresponding to this representation.
 */
func (this *Representation) String() string {
	result := BLOCK_TYPE_INVALID

	/*
	 * Decide on representation.
	 */
	switch *this {
	case REPRESENTATION_RSA_PRIVATE_KEY_PKCS1:
		result = BLOCK_TYPE_PRIVATE_KEY_PKCS1
	case REPRESENTATION_RSA_PRIVATE_KEY_PKCS8:
		result = BLOCK_TYPE_PRIVATE_KEY_PKCS8
	case REPRESENTATION_RSA_PUBLIC_KEY_PKCS1:
		result = BLOCK_TYPE_PUBLIC_KEY_PKCS1
	case REPRESENTATION_RSA_PUBLIC_KEY_PKIX:
		result = BLOCK_TYPE_PUBLIC_KEY_PKIX
	}

	return result
}

/*
 * Decode a PEM-encoded RSA key and return the decoded key material, the
 * representation and, potentially, an error.
 */
func DecodePEM(pemData []byte) ([]byte, Representation, error) {
	block, rest := pem.Decode(pemData)
	sizeRest := len(rest)
	resultData := []byte(nil)
	resultRepresentation := REPRESENTATION_INVALID
	errResult := error(nil)

	/*
	 * Check whether PEM decoding was successful.
	 */
	if (block == nil) || (sizeRest != 0) {
		errResult = fmt.Errorf("%s", "Failed to decode PEM block")
	} else {
		t := block.Type
		representation := CreateRepresentation(t)

		/*
		 * Check if representation is valid.
		 */
		if representation == REPRESENTATION_INVALID {
			errResult = fmt.Errorf("Unknown PEM block type: %s", t)
		} else {
			resultData = block.Bytes
			resultRepresentation = representation
		}

	}

	return resultData, resultRepresentation, errResult
}

/*
 * Encode an RSA key in a certain representation as PEM.
 */
func EncodePEM(key []byte, representation Representation) []byte {
	t := representation.String()

	/*
	 * Create PEM block.
	 */
	block := pem.Block{
		Type:  t,
		Bytes: key,
	}

	result := pem.EncodeToMemory(&block)
	return result
}

/*
 * Loads an RSA private key in ASN.1 encoding and either PKCS1 or PKCS8
 * representation.
 */
func LoadRSAPrivateKey(keyData []byte, representation Representation) (*rsa.PrivateKey, error) {
	result := (*rsa.PrivateKey)(nil)
	errResult := error(nil)

	/*
	 * Decode either PKCS1 or PKCS8 representation.
	 */
	switch representation {
	case REPRESENTATION_RSA_PRIVATE_KEY_PKCS1:
		privateKey, err := x509.ParsePKCS1PrivateKey(keyData)

		/*
		 * Check if an error occurred decoding the key.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to decode RSA private key in PKCS1 representation: %s", msg)
		} else {
			result = privateKey
		}

	case REPRESENTATION_RSA_PRIVATE_KEY_PKCS8:
		privateKey, err := x509.ParsePKCS8PrivateKey(keyData)
		rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)

		/*
		 * Check if an error occurred decoding the key.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to decode RSA private key in PKCS8 representation: %s", msg)
		} else if !ok {
			errResult = fmt.Errorf("Key is not an RSA private key.")
		} else {
			result = rsaPrivateKey
		}

	default:
		representationString := representation.String()
		errResult = fmt.Errorf("Illegal representation for RSA private key: %s", representationString)
	}

	return result, errResult
}

/*
 * Loads an RSA public key in ASN.1 encoding and either PKCS1 or PKIX
 * representation.
 */
func LoadRSAPublicKey(keyData []byte, representation Representation) (*rsa.PublicKey, error) {
	result := (*rsa.PublicKey)(nil)
	errResult := error(nil)

	/*
	 * Decode either PKCS1 or PKIX representation.
	 */
	switch representation {
	case REPRESENTATION_RSA_PUBLIC_KEY_PKCS1:
		publicKey, err := x509.ParsePKCS1PublicKey(keyData)

		/*
		 * Check if an error occurred decoding the key.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to decode RSA public key in PKCS1 representation: %s", msg)
		} else {
			result = publicKey
		}

	case REPRESENTATION_RSA_PUBLIC_KEY_PKIX:
		publicKey, err := x509.ParsePKIXPublicKey(keyData)
		rsaPublicKey, ok := publicKey.(*rsa.PublicKey)

		/*
		 * Check if an error occurred decoding the key.
		 */
		if err != nil {
			msg := err.Error()
			errResult = fmt.Errorf("Failed to decode RSA public key in PKIX representation: %s", msg)
		} else if !ok {
			errResult = fmt.Errorf("Key is not an RSA public key.")
		} else {
			result = rsaPublicKey
		}

	default:
		representationString := representation.String()
		errResult = fmt.Errorf("Illegal representation for RSA public key: %s", representationString)
	}

	return result, errResult
}

/*
 * Signs a message using RSA PSS.
 */
func SignPSS(message []byte, key *rsa.PrivateKey, csprng io.Reader) ([]byte, error) {
	hashAlgorithm := crypto.SHA512
	hash := sha512.Sum512(message)
	hashBuf := hash[:]

	/*
	 * Signature options.
	 */
	opts := rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthAuto,
		Hash:       hashAlgorithm,
	}

	result, err := rsa.SignPSS(csprng, key, hashAlgorithm, hashBuf, &opts)
	return result, err
}

/*
 * Verifies a signature using RSA PSS.
 */
func VerifyPSS(message []byte, signature []byte, key *rsa.PublicKey) bool {
	hashAlgorithm := crypto.SHA512
	hash := sha512.Sum512(message)
	hashBuf := hash[:]

	/*
	 * Signature options.
	 */
	opts := rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthAuto,
		Hash:       hashAlgorithm,
	}

	err := rsa.VerifyPSS(key, hashAlgorithm, hashBuf, signature, &opts)
	result := (err == nil)
	return result
}
