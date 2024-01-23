package geo

/*
 * A geographic location.
 */
type Location interface {
	Latitude() int32
	Longitude() int32
	Timestamp() uint64
}

/*
 * A location database.
 */
type Database interface {
	LocationAt(idx int) (Location, error)
	LocationCount() int
}
