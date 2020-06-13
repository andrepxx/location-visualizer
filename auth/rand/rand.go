package rand

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
)

const (
	BITS_PER_BYTE    = 8
	BLOCK_SIZE_WORDS = 2
	WORD_SIZE        = 8
	BLOCK_SIZE       = BLOCK_SIZE_WORDS * WORD_SIZE
	KEY_SIZE         = 32
	SEED_SIZE        = KEY_SIZE + BLOCK_SIZE
)

var g_prng io.Reader = rand.Reader

/*
 * Data structure representing a cryptographically secure pseudo-random number
 * generator.
 *
 * All words are considered to be stored in network byte order - most
 * significant byte first.
 */
type prngStruct struct {
	blockCipher  cipher.Block
	counter      [BLOCK_SIZE_WORDS]uint64
	counterBytes [BLOCK_SIZE]byte
	cipherBlock  [BLOCK_SIZE]byte
	ptr          uint
	mutex        sync.Mutex
}

/*
 * Synchronizes byte counter to word counter.
 */
func (this *prngStruct) syncCounters() {
	counter := this.counter[:]
	counterBytes := this.counterBytes[:]
	counterBytesLength := len(counterBytes)

	/*
	 * Read counter bytes from words.
	 */
	for i := 0; i < counterBytesLength; i++ {
		wordNum := i / WORD_SIZE
		word := counter[wordNum]
		numByte := i % WORD_SIZE
		shiftBytes := WORD_SIZE - (numByte + 1)
		shiftBits := BITS_PER_BYTE * shiftBytes
		counterBytes[i] = byte(word >> shiftBits)
	}

}

/*
 * Increments the counter value of this PRNG.
 */
func (this *prngStruct) incrementCounter() {
	counter := this.counter[:]
	counterHigh := counter[0]
	counterLowOld := counter[1]
	counterLow := counterLowOld + 1

	/*
	 * In case low half of counter overflows, increment high half of
	 * counter as well.
	 */
	if counterLow < counterLowOld {
		counterHigh++
	}

	counter[0] = counterHigh
	counter[1] = counterLow
	this.syncCounters()
}

/*
 * Generates a new cipher block.
 */
func (this *prngStruct) generateCipherBlock() {
	this.incrementCounter()
	c := this.blockCipher
	in := this.counterBytes[:]
	out := this.cipherBlock[:]
	c.Encrypt(out, in)
}

/*
 * Read cryptographically secure pseudo-random numbers into a byte buffer.
 */
func (this *prngStruct) Read(target []byte) (int, error) {

	/*
	 * Make sure that target buffer is not nil.
	 */
	if target == nil {
		return 0, fmt.Errorf("%s", "Error reading from CSPRNG: Target buffer must not be nil.")
	} else {
		this.mutex.Lock()
		cipherBlock := this.cipherBlock[:]
		readPtr := this.ptr
		numBytesRead := int(0)

		/*
		 * Fill target buffer with pseudo-random numbers.
		 */
		for writePtr := range target {
			currentByte := cipherBlock[readPtr]
			target[writePtr] = currentByte
			readPtr++

			/*
			 * If we reached the end of a cipher block, generate a
			 * new one.
			 */
			if readPtr >= BLOCK_SIZE {
				this.generateCipherBlock()
				readPtr = 0
			}

			numBytesRead++
		}

		this.ptr = readPtr
		this.mutex.Unlock()
		return numBytesRead, nil
	}

}

/*
 * Creates a cryptographically secure pseudo-random number generator
 * initialized to a 384 bit seed.
 *
 * The provided seed must be exactly 48 bytes long.
 */
func CreatePRNG(seed []byte) (io.Reader, error) {
	seedSize := len(seed)

	/*
	 * Check if seed has the correct size.
	 */
	if seedSize != SEED_SIZE {
		return nil, fmt.Errorf("Seed size does not match. Expected %d, got %d.", SEED_SIZE, seedSize)
	} else {
		key := seed[0:KEY_SIZE]
		counter := seed[KEY_SIZE:SEED_SIZE]
		counterHigh := counter[0:WORD_SIZE]
		counterHighWord := uint64(0)

		/*
		 * Initialize higher (MSB) half of counter.
		 */
		for i := uint(0); i < WORD_SIZE; i++ {
			iInc := i + 1
			shiftBytes := WORD_SIZE - iInc
			shiftBits := BITS_PER_BYTE * shiftBytes
			currentByte := counterHigh[i]
			currentByteShifted := uint64(currentByte) << shiftBits
			counterHighWord |= currentByteShifted
		}

		counterLow := counter[WORD_SIZE:BLOCK_SIZE]
		counterLowWord := uint64(0)

		/*
		 * Initialize lower (LSB) half of counter.
		 */
		for i := uint(0); i < WORD_SIZE; i++ {
			iInc := i + 1
			shiftBytes := WORD_SIZE - iInc
			shiftBits := BITS_PER_BYTE * shiftBytes
			currentByte := counterLow[i]
			currentByteShifted := uint64(currentByte) << shiftBits
			counterLowWord |= currentByteShifted
		}

		c, err := aes.NewCipher(key)

		/*
		 * Check if cipher was created successfully.
		 */
		if err != nil {
			msg := err.Error()
			errNew := fmt.Errorf("Failed to initialize AES block cipher: %s", msg)
			return nil, errNew
		} else {

			/*
			 * Create CSPRNG.
			 */
			prng := &prngStruct{
				blockCipher: c,
				counter: [BLOCK_SIZE_WORDS]uint64{
					counterHighWord,
					counterLowWord,
				},
			}

			prng.syncCounters()
			prng.generateCipherBlock()
			return prng, nil
		}

	}

}

/*
 * Returns the operating system entropy source.
 */
func SystemPRNG() io.Reader {
	return g_prng
}
