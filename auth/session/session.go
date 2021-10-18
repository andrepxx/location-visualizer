package session

import (
	"crypto/sha512"
	"crypto/subtle"
	"fmt"
	"github.com/andrepxx/location-visualizer/auth/user"
	"io"
	"sync"
	"time"
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
	sessions    []sessionStruct
}

/*
 * A session manager.
 */
type Manager interface {
	CreateToken(token []byte) Token
	Challenge(name string) (Challenge, error)
	Response(name string, hash []byte) (Token, error)
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
	id := int64(-1)
	sessions := this.sessions

	/*
	 * Iterate over the sessions.
	 */
	for i, session := range sessions {
		other := session.token
		c := subtle.ConstantTimeCompare(other[:], token[:])

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
	sessions := this.sessions
	sessions[id].mutex.RLock()
	lastAccess := sessions[id].lastAccess
	sessions[id].mutex.RUnlock()
	period := now.Sub(lastAccess)
	expiry := this.expiry

	/*
	 * Check if session should be expired or refreshed.
	 */
	if period < expiry {
		return SESSION_REFRESH, now
	} else {
		return SESSION_EXPIRE, now
	}
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
func (this *managerStruct) Response(name string, response []byte) (Token, error) {
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
		return nil, fmt.Errorf("User '%s' not found.", name)
	} else if hashSize == 0 {
		return nil, fmt.Errorf("%s", "Authentication failed.")
	} else {
		nonceAndHash := append(nonce[:], hash...)
		expected := sha512.Sum512(nonceAndHash)
		c := subtle.ConstantTimeCompare(response, expected[:])

		/*
		 * Check if the response matches.
		 */
		if c != CTC_EQUAL {
			return nil, fmt.Errorf("%s", "Authentication failed.")
		} else {
			token := [LENGTH]byte{}
			rng := this.prng
			numBytes, err := rng.Read(token[:])

			/*
			 * Check if token was generated and associate it to session.
			 */
			if err != nil {
				msg := err.Error()
				return nil, fmt.Errorf("Failed to generate session token: %s", msg)
			} else if numBytes != LENGTH {
				return nil, fmt.Errorf("Failed to generate session token: Incorrect number of bytes read from PRNG: Expected %d, got %d.", LENGTH, numBytes)
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

				copy(s.token[:], token[:])
				this.mutex.Lock()
				mgr.RegenerateNonce(name)
				sessions := this.sessions
				sessions = append(sessions, s)
				this.sessions = sessions
				this.mutex.Unlock()

				/*
				 * Create session token.
				 */
				t := tokenStruct{
					token: token,
				}

				return &t, nil
			}

		}

	}

}

/*
 * Terminate a session given a session token, logging out the corresponding user.
 */
func (this *managerStruct) Terminate(token Token) error {
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
		this.mutex.Unlock()
		return fmt.Errorf("%s", "No session with this token found.")
	} else {
		this.expire(sid)
		this.mutex.Unlock()
		return nil
	}

}

/*
 * Returns the name of the user associated with a session token.
 */
func (this *managerStruct) UserName(token Token) (string, error) {
	t := token.Token()
	this.mutex.RLock()
	sid := this.sessionIdFromToken(t)

	/*
	 * Check if session with this token exists.
	 */
	if sid < 0 {
		this.mutex.RUnlock()
		return "", fmt.Errorf("%s", "No session with this token found.")
	} else {
		roe, now := this.refreshOrExpire(sid)

		/*
		 * Refresh or expire session.
		 */
		switch roe {
		case SESSION_REFRESH:
			this.refresh(sid, now)
			sessions := this.sessions
			s := &sessions[sid]
			name := s.name
			this.mutex.RUnlock()
			return name, nil
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
			return "", fmt.Errorf("%s", "No session with this token found.")
		default:
			this.mutex.RUnlock()
			return "", fmt.Errorf("%s", "Something unexpected happened.")
		}

	}

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
		sessions := []sessionStruct{}

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
