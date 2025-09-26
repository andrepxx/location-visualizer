package multipart

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
)

const (
	SIZE_BUFFER        = 4096
	SIZE_FOLDED_HASH   = 35
	MIME_TYPE_TEMPLATE = "multipart/form-data; boundary=%s"
)

/*
 * A key value pair.
 */
type KeyValuePair interface {
	Key() string
	Value() string
}

/*
 * Data structure representing a key value pair.
 */
type keyValuePairStruct struct {
	key   string
	value string
}

/*
 * Returns the key associated with this key value pair.
 */
func (this *keyValuePairStruct) Key() string {
	key := this.key
	return key
}

/*
 * Returns the value associated with this key value pair.
 */
func (this *keyValuePairStruct) Value() string {
	value := this.value
	return value
}

/*
 * Creates a key value pair.
 */
func CreateKeyValuePair(key string, value string) KeyValuePair {

	/*
	 * Create key value pair.
	 */
	kv := keyValuePairStruct{
		key:   key,
		value: value,
	}

	return &kv
}

/*
 * A file entry.
 */
type FileEntry interface {
	Key() string
	Name() string
	Value() io.ReadSeekCloser
}

/*
 * Data structure representing a file entry.
 */
type fileEntryStruct struct {
	key   string
	name  string
	value io.ReadSeekCloser
}

/*
 * Returns the key associated with this file entry.
 */
func (this *fileEntryStruct) Key() string {
	key := this.key
	return key
}

/*
 * Returns the name associated with this file entry.
 */
func (this *fileEntryStruct) Name() string {
	name := this.name
	return name
}

/*
 * Returns the value associated with this file entry.
 */
func (this *fileEntryStruct) Value() io.ReadSeekCloser {
	value := this.value
	return value
}

/*
 * Creates a file entry.
 */
func CreateFileEntry(key string, name string, value io.ReadSeekCloser) FileEntry {

	/*
	 * Create file entry.
	 */
	fe := fileEntryStruct{
		key:   key,
		name:  name,
		value: value,
	}

	return &fe
}

/*
 * Data structure representing a multipart provider.
 */
type multipartProviderStruct struct {
	buf            *bytes.Buffer
	fileBuffer     []byte
	fileEntries    []fileEntryStruct
	fw             io.Writer
	keyValuePairs  []keyValuePairStruct
	mimeType       string
	trailerWritten bool
	w              *multipart.Writer
}

/*
 * Generates the next part of the multipart message.
 */
func (this *multipartProviderStruct) generateNextPart() bool {
	keyValuePairs := this.keyValuePairs
	numKeyValuePairs := len(keyValuePairs)
	fileEntries := this.fileEntries
	numFileEntries := len(fileEntries)
	trailerWritten := this.trailerWritten
	w := this.w

	/*
	 * Check if we can generate a key value pair, a file entry or a trailer.
	 */
	if numKeyValuePairs > 0 {
		keyValuePair := keyValuePairs[0]
		key := keyValuePair.key
		value := keyValuePair.value
		err := w.WriteField(key, value)

		/*
		 * Check if error occured writing the field.
		 */
		if err != nil {
			msg := err.Error()
			panic("Failed to generate message part: " + msg)
		}

		this.keyValuePairs = keyValuePairs[1:]
		return true
	} else if numFileEntries > 0 {
		fileEntry := fileEntries[0]
		fw := this.fw

		/*
		 * If file header has not been written, then do so.
		 *
		 * Otherwise, write file content.
		 */
		if fw == nil {
			key := fileEntry.key
			name := fileEntry.name
			err := error(nil)
			fw, err = w.CreateFormFile(key, name)

			/*
			 * Check if error occured creating the file part.
			 */
			if err != nil {
				msg := err.Error()
				panic("Failed to generate message part: " + msg)
			}

			this.fw = fw
		} else {
			fw := this.fw
			value := fileEntry.value
			_, err := io.CopyN(fw, value, SIZE_BUFFER)

			/*
			 * Advance to next file.
			 */
			if err != nil {
				numFileEntries = len(fileEntries)
				this.fileEntries = fileEntries[1:]
				this.fw = nil
			}

		}

		return true
	} else if trailerWritten == false {
		err := w.Close()

		/*
		 * Check if error occured writing trailer.
		 */
		if err != nil {
			msg := err.Error()
			panic("Failed to write multipart trailer: " + msg)
		}

		this.trailerWritten = true
		return true
	} else {
		return false
	}

}

/*
 * Provides the Close method of io.Closer, closing all underlying file
 * descriptors.
 */
func (this *multipartProviderStruct) Close() error {
	fileEntries := this.fileEntries
	errResult := error(nil)

	/*
	 * Iterate over all file entries.
	 */
	for _, entry := range fileEntries {
		file := entry.value

		/*
		 * Close file.
		 */
		if file != nil {
			err := file.Close()

			/*
			 * Store first error.
			 */
			if (err != nil) && (errResult == nil) {
				errResult = err
			}

		}

	}

	return errResult
}

/*
 * Provides the Read method of io.Reader.
 */
func (this *multipartProviderStruct) Read(p []byte) (int, error) {
	bytesRequested := len(p)
	bytesRead := int(0)
	buf := this.buf
	n := int(0)
	moreParts := true
	bytesAvailable := buf.Len()

	/*
	 * Do this until there is nothing more to read.
	 */
	for (bytesRead < bytesRequested) && ((bytesAvailable > 0) || moreParts) {

		/*
		 * If the buffer still has content, read from the buffer.
		 *
		 * Otherwise, generate a new part.
		 */
		if bytesAvailable > 0 {
			q := p[bytesRead:]
			n, _ = buf.Read(q)
			bytesRead += n
		} else {
			moreParts = this.generateNextPart()
		}

		bytesAvailable = buf.Len()
	}

	errResult := error(nil)

	/*
	 * If there are no more bytes to read and there are no more parts, we've
	 * reached end-of-file.
	 */
	if (bytesAvailable <= 0) && !moreParts {
		errResult = io.EOF
	}

	return bytesRead, errResult
}

/*
 * Creates a multipart provider returning the key value pairs and file entries.
 */
func CreateMultipartProvider(keyValuePairs []KeyValuePair, fileEntries []FileEntry) (io.ReadCloser, string) {
	sha := sha512.New()
	buf := &bytes.Buffer{}
	numKeyValuePairs := len(keyValuePairs)
	keyValuePairsInternal := make([]keyValuePairStruct, numKeyValuePairs)

	/*
	 * Iterate over all key value pairs.
	 */
	for i, keyValuePair := range keyValuePairs {
		key := keyValuePair.Key()
		value := keyValuePair.Value()

		/*
		 * Create new key value pair.
		 */
		kv := keyValuePairStruct{
			key:   key,
			value: value,
		}

		keyValuePairsInternal[i] = kv
	}

	numFileEntries := len(fileEntries)
	fileEntriesInternal := make([]fileEntryStruct, numFileEntries)

	/*
	 * Iterate over all file entries.
	 */
	for i, fileEntry := range fileEntries {
		key := fileEntry.Key()
		name := fileEntry.Name()
		value := fileEntry.Value()

		/*
		 * Create new file entry.
		 */
		f := fileEntryStruct{
			key:   key,
			name:  name,
			value: value,
		}

		_, err := value.Seek(0, io.SeekStart)

		/*
		 * Check if error occured.
		 */
		if err != nil {
			msg := err.Error()
			panic("Failed to seek to start of file: " + msg)
		}

		_, err = io.Copy(sha, value)

		/*
		 * Check if error occured.
		 */
		if err != nil {
			msg := err.Error()
			panic("Failed to read from file: " + msg)
		}

		_, err = value.Seek(0, io.SeekStart)

		/*
		 * Check if error occured.
		 */
		if err != nil {
			msg := err.Error()
			panic("Failed to seek to start of file: " + msg)
		}

		fileEntriesInternal[i] = f
	}

	hash := make([]byte, sha512.Size)
	hash = sha.Sum(hash[:0])
	foldedHash := make([]byte, SIZE_FOLDED_HASH)

	/*
	 * Fold the hash into a size of 35 bytes.
	 */
	for i, b := range hash {
		idx := i % SIZE_FOLDED_HASH
		foldedHash[idx] ^= b
	}

	boundary := hex.EncodeToString(foldedHash)
	w := multipart.NewWriter(buf)
	w.SetBoundary(boundary)
	mimeType := fmt.Sprintf(MIME_TYPE_TEMPLATE, boundary)

	/*
	 * Create multipart provider.
	 */
	prov := multipartProviderStruct{
		buf:            buf,
		fileEntries:    fileEntriesInternal,
		fw:             nil,
		keyValuePairs:  keyValuePairsInternal,
		trailerWritten: false,
		mimeType:       mimeType,
		w:              w,
	}

	return &prov, mimeType
}
