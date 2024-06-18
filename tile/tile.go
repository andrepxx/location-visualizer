package tile

/*
 * An image - either fetched from a tile server or stored in cache.
 *
 * Implements io.ReadSeekCloser and io.ReaderAt.
 */
type Image interface {
	Close() error
	Read(buf []byte) (int, error)
	ReadAt(buf []byte, offset int64) (int, error)
	Seek(offset int64, whence int) (int64, error)
}

/*
 * Data structure representing a tile ID.
 */
type Id struct {
	x uint32
	y uint32
	z uint8
}

/*
 * Returns the X coordinate associated with this map tile.
 */
func (this *Id) X() uint32 {
	result := this.x
	return result
}

/*
 * Returns the Y coordinate associated with this map tile.
 */
func (this *Id) Y() uint32 {
	result := this.y
	return result
}

/*
 * Returns the zoom level associated with this map tile.
 */
func (this *Id) Z() uint8 {
	result := this.z
	return result
}

/*
 * Creates a tile ID based on zoom level, x and y coordinate.
 */
func CreateId(z uint8, x uint32, y uint32) Id {

	/*
	 * Create tile ID.
	 */
	id := Id{
		x: x,
		y: y,
		z: z,
	}

	return id
}
