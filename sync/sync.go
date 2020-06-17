package sync

/*
 * An empty data structure does not occupy memory.
 */
type empty struct {
}

/*
 * Data structure representing a semaphore.
 */
type semaphoreStruct struct {
	c chan empty
}

/*
 * A Semaphore provides synchronized access to a constrained ressource.
 */
type Semaphore interface {
	Acquire()
	Release()
}

/*
 * Acquires a semaphore.
 */
func (this *semaphoreStruct) Acquire() {
	c := this.c
	e := empty{}
	c <- e
}

/*
 * Releases a semaphore.
 */
func (this *semaphoreStruct) Release() {
	c := this.c
	<-c
}

/*
 * Creates a new Semaphore with a specific limit for concurrent access.
 */
func CreateSemaphore(limit uint32) Semaphore {
	sc := make(chan empty, limit)

	/*
	 * Create semaphore.
	 */
	s := semaphoreStruct{
		c: sc,
	}

	return &s
}
