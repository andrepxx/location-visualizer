package session

import (
	"crypto/sha512"
	"crypto/subtle"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/andrepxx/location-visualizer/auth/publickey"
	"github.com/andrepxx/location-visualizer/auth/user"
)

/*
 * Global constants.
 */
const (
	CTC_EQUAL       = 1
	LENGTH          = 64
	SESSION_REFRESH = false
	SESSION_EXPIRE  = true
)

/*
 * Data structure representing a challenge for password-based authentication.
 */
type challengeStruct struct {
	nonce [LENGTH]byte
	salt  [LENGTH]byte
}

/*
 * A challenge for password-based authentication.
 */
type Challenge interface {
	Nonce() [LENGTH]byte
	Salt() [LENGTH]byte
}

/*
 * Data structure representing a session token.
 */
type tokenStruct struct {
	token [LENGTH]byte
}

/*
 * A session token.
 */
type Token interface {
	Token() [LENGTH]byte
}

/*
 * Data structure representing an authenticated session.
 */
type sessionStruct struct {
	token      [LENGTH]byte
	name       string
	mutex      sync.RWMutex
	lastAccess time.Time
}

/*
 * Data structure representing a session manager
 */
type managerStruct struct {
	expiry      time.Duration
	prng        io.Reader
	mutex       sync.RWMutex
	userManager user.Manager
	sessions    []*sessionStruct
}

/*
 * A session manager.
 */
type Manager interface {
	CreateToken(token []byte) Token
	Challenge(name string) (Challenge, error)
	ResponseHash(name string, response []byte) (Token, error)
	ResponseSignature(name string, response []byte) (Token, error)
	Terminate(token Token) error
	UserName(token Token) (string, error)
}

/*
 * Returns the nonce of an authentication challenge.
 */
func (this *challengeStruct) Nonce() [LENGTH]byte {
	return this.nonce
}

/*
 * Returns the salt of an authentication challenge.
 */
func (this *challengeStruct) Salt() [LENGTH]byte {
	return this.salt
}

/*
 * Returns the contents of a session token.
 */
func (this *tokenStruct) Token() [LENGTH]byte {
	return this.token
}

/*
 * Expire a session.
 *
 * The caller is expected to hold a write lock on the session list from when
 * he obtained the session ID.
 */
func (this *managerStruct) expire(id int64) {
	sessions := this.sessions
	idInc := id + 1
	sessions = append(sessions[:id], sessions[idInc:]...)
	this.sessions = sessions
}

/*
 * Returns the id of a session associated with a certain token.
 *
 * The caller is expected to hold at least a read lock on the session list.
 */
func (this *managerStruct) sessionIdFromToken(token [LENGTH]byte) int64 {
	tokenSlice := token[:]
	id := int64(-1)
	sessions := this.sessions

	/*
	 * Iterate over the sessions.
	 */
	for i, session := range sessions {
		other := session.token
		otherSlice := other[:]
		c := subtle.ConstantTimeCompare(otherSlice, tokenSlice)

		/*
		 * In case of a match, store session ID.
		 */
		if c == CTC_EQUAL {
			id = int64(i)
		}

	}

	return id
}

/*
 * Refresh a session.
 *
 * This function locks the session it refreshes for writing.
 *
 * The caller is expected to hold at least a read lock on the session list from
 * when he obtained the session ID.
 */
func (this *managerStruct) refresh(id int64, now time.Time) {
	sessions := this.sessions
	sessions[id].mutex.Lock()
	sessions[id].lastAccess = now
	sessions[id].mutex.Unlock()
}

/*
 * Check if a session should be refreshed or expired, and the point in time
 * when this was checked.
 *
 * This function locks the session it checks for reading.
 *
 * The caller is expected to hold at least a read lock on the session list from
 * when he obtained the session ID.
 */
func (this *managerStruct) refreshOrExpire(id int64) (bool, time.Time) {
	now := time.Now()
	result := SESSION_EXPIRE
	sessions := this.sessions
	sessions[id].mutex.RLock()
	session := sessions[id]
	lastAccess := session.lastAccess
	sessions[id].mutex.RUnlock()
	period := now.Sub(lastAccess)
	expiry := this.expiry

	/*
	 * Check if session should be refreshed.
	 */
	if period < expiry {
		result = SESSION_REFRESH
	}

	return result, now
}

/*
 * Creates a session token from a byte slice.
 */
func (this *managerStruct) CreateToken(token []byte) Token {
	buf := [LENGTH]byte{}
	copy(buf[:], token)

	/*
	 * Create session token.
	 */
	t := tokenStruct{
		token: buf,
	}

	return &t
}

/*
 * Generate an authentication challenge for a user, given his / her name.
 */
func (this *managerStruct) Challenge(name string) (Challenge, error) {
	this.mutex.RLock()
	mgr := this.userManager
	salt, errSalt := mgr.Salt(name)
	nonce, errNonce := mgr.Nonce(name)
	this.mutex.RUnlock()

	/*
	 * Check if salt and nonce could be obtained.
	 */
	if (errSalt != nil) || (errNonce != nil) {
		return nil, fmt.Errorf("User '%s' not found.", name)
	} else {

		/*
		 * Create authentication challenge.
		 */
		challenge := challengeStruct{
			nonce: nonce,
			salt:  salt,
		}

		return &challenge, nil
	}

}

/*
 * Verify an authentication response for a user, given his / her name and the response hash.
 */
func (this *managerStruct) ResponseHash(name string, reseponse []byte) (Token, error) {
	result := Token(nil)
	errResult := error(nil)
	this.mutex.RLock()
	mgr := this.userManager
	nonce, errNonce := mgr.Nonce(name)
	hash, errHash := mgr.Hash(name)
	this.mutex.RUnlock()
	hashSize := len(hash)

	/*
	 * If user does not exist or has no hash set, abort with failure.
	 */
	if (errNonce != nil) || (errHash != nil) {
		errResult = fmt.Errorf("User '%s' not found.", name)
	} else if hashSize == 0 {
		errResult = fmt.Errorf("%s", "Authentication failed.")
	} else {
		nonceSlice := nonce[:]
		nonceAndHash := append(nonceSlice, hash...)
		expected := sha512.Sum512(nonceAndHash)
		expectedSlice := expected[:]
		c := subtle.ConstantTimeCompare(reseponse, expectedSlice)

		/*
		 * Check if the response matches.
		 */
		if c != CTC_EQUAL {
			errResult = fmt.Errorf("%s", "Authentication failed.")
		} else {
			token := [LENGTH]byte{}
			tokenSlice := token[:]
			rng := this.prng
			numBytes, err := rng.Read(tokenSlice)

			/*
			 * Check if token was generated and associate it to session.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Failed to generate session token: %s", msg)
			} else if numBytes != LENGTH {
				errResult = fmt.Errorf("Failed to generate session token: Incorrect number of bytes read from PRNG: Expected %d, got %d.", LENGTH, numBytes)
			} else {
				now := time.Now()

				/*
				 * Create session.
				 */
				s := sessionStruct{
					token:      [LENGTH]byte{},
					name:       name,
					lastAccess: now,
				}

				sessionToken := s.token[:]
				copy(sessionToken, tokenSlice)
				this.mutex.Lock()
				mgr.RegenerateNonce(name)
				sessions := this.sessions
				sessions = append(sessions, &s)
				this.sessions = sessions
				this.mutex.Unlock()

				/*
				 * Create session token.
				 */
				t := tokenStruct{
					token: token,
				}

				result = &t
			}

		}

	}

	return result, errResult
}

/*
 * Verify an authentication response for a user, given his / her name and the response signature.
 */
func (this *managerStruct) ResponseSignature(name string, response []byte) (Token, error) {
	result := Token(nil)
	errResult := error(nil)
	this.mutex.RLock()
	mgr := this.userManager
	nonce, errNonce := mgr.Nonce(name)
	publicKeys, errPublicKeys := mgr.PublicKeys(name)
	this.mutex.RUnlock()

	/*
	 * If user does not exist, abort with failure.
	 */
	if (errNonce != nil) || (errPublicKeys != nil) {
		errResult = fmt.Errorf("User '%s' not found.", name)
	} else {
		nonceSlice := nonce[:]
		valid := false

		/*
		 * Verify RSA PSS signature against every public key.
		 */
		for _, publicKey := range publicKeys {
			keyData := publicKey.KeyData()
			representation := publicKey.Representation()
			rsaPublicKey, err := publickey.LoadRSAPublicKey(keyData, representation)

			/*
			 * Check if key could be loaded.
			 */
			if err == nil {
				valid = publickey.VerifyPSS(nonceSlice, response, rsaPublicKey) || valid
			}

		}

		/*
		 * Check if signature validated successfully against one of the public
		 * keys.
		 */
		if !valid {
			errResult = fmt.Errorf("%s", "Signature verification failed.")
		} else {
			token := [LENGTH]byte{}
			tokenSlice := token[:]
			rng := this.prng
			numBytes, err := rng.Read(tokenSlice)

			/*
			 * Check if token was generated and associate it to session.
			 */
			if err != nil {
				msg := err.Error()
				errResult = fmt.Errorf("Failed to generate session token: %s", msg)
			} else if numBytes != LENGTH {
				errResult = fmt.Errorf("Failed to generate session token: Incorrect number of bytes read from PRNG: Expected %d, got %d.", LENGTH, numBytes)
			} else {
				now := time.Now()

				/*
				 * Create session.
				 */
				s := sessionStruct{
					token:      [LENGTH]byte{},
					name:       name,
					lastAccess: now,
				}

				sessionToken := s.token[:]
				copy(sessionToken, tokenSlice)
				this.mutex.Lock()
				mgr.RegenerateNonce(name)
				sessions := this.sessions
				sessions = append(sessions, &s)
				this.sessions = sessions
				this.mutex.Unlock()

				/*
				 * Create session token.
				 */
				t := tokenStruct{
					token: token,
				}

				result = &t
			}

		}

	}

	return result, errResult
}

/*
 * Terminate a session given a session token, logging out the corresponding user.
 */
func (this *managerStruct) Terminate(token Token) error {
	errResult := error(nil)
	t := token.Token()
	this.mutex.Lock()
	sid := this.sessionIdFromToken(t)

	/*
	 * Refresh or expire is only applicable if the session exists.
	 */
	if sid >= 0 {
		roe, _ := this.refreshOrExpire(sid)

		/*
		 * Check if session shall be expired.
		 */
		if roe == SESSION_EXPIRE {
			this.expire(sid)
		}

	}

	sid = this.sessionIdFromToken(t)

	/*
	 * If a session with this token exists, terminate it.
	 */
	if sid < 0 {
		errResult = fmt.Errorf("%s", "No session with this token found.")
	} else {
		this.expire(sid)
	}

	this.mutex.Unlock()
	return errResult
}

/*
 * Returns the name of the user associated with a session token.
 */
func (this *managerStruct) UserName(token Token) (string, error) {
	result := ""
	errResult := error(nil)
	t := token.Token()
	this.mutex.RLock()
	sid := this.sessionIdFromToken(t)

	/*
	 * Check if session with this token exists.
	 */
	if sid < 0 {
		errResult = fmt.Errorf("%s", "No session with this token found.")
	} else {
		roe, now := this.refreshOrExpire(sid)

		/*
		 * Refresh or expire session.
		 */
		switch roe {
		case SESSION_REFRESH:
			this.refresh(sid, now)
			sessions := this.sessions
			s := sessions[sid]
			result = s.name
		case SESSION_EXPIRE:
			this.mutex.RUnlock()
			this.mutex.Lock()
			sid = this.sessionIdFromToken(t)

			/*
			 * Have to search again, since we re-acquired the lock!
			 */
			if sid >= 0 {
				this.expire(sid)
			}

			this.mutex.Unlock()
			this.mutex.RLock()
			errResult = fmt.Errorf("%s", "No session with this token found.")
		default:
			errResult = fmt.Errorf("%s", "Something unexpected happened.")
		}

	}

	this.mutex.RUnlock()
	return result, errResult
}

/*
 * Creates a new session manager.
 */
func CreateManager(userManager user.Manager, prng io.Reader, expiry time.Duration) (Manager, error) {

	/*
	 * Check if user manager and PRNG were provided.
	 */
	if userManager == nil {
		return nil, fmt.Errorf("%s", "User manager must not be nil!")
	} else if prng == nil {
		return nil, fmt.Errorf("%s", "PRNG must not be nil!")
	} else {
		sessions := []*sessionStruct{}

		/*
		 * Create session manager.
		 */
		ms := managerStruct{
			expiry:      expiry,
			prng:        prng,
			sessions:    sessions,
			userManager: userManager,
		}

		return &ms, nil
	}

}
