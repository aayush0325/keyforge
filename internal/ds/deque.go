package ds

// Deque is a generic ring-buffer-based double-ended queue.
type Deque[T comparable] struct {
	Buf []T
}

// NewDeque creates a deque with an initial capacity.
func NewDeque[T comparable]() *Deque[T] {
	return &Deque[T]{
		Buf: make([]T, 0),
	}
}

func (d *Deque[T]) Len() int {
	return len(d.Buf)
}

func (d *Deque[T]) empty() bool {
	return len(d.Buf) == 0
}

func (d *Deque[T]) PushBack(v T) {
	d.Buf = append(d.Buf, v)
}

func (d *Deque[T]) PushFront(v T) {
	d.Buf = append([]T{v}, d.Buf...)
}

func (d *Deque[T]) PopFront() (T, bool) {
	if d.empty() {
		var noop T
		return noop, false
	}
	val := d.Buf[0]
	d.Buf = d.Buf[1:]
	return val, true
}

func (d *Deque[T]) PopBack() (T, bool) {
	if d.empty() {
		var zero T
		return zero, false
	}

	v := d.Buf[len(d.Buf)-1]
	d.Buf = d.Buf[:len(d.Buf)-1]
	return v, true
}

func (d *Deque[T]) Front() (T, bool) {
	if d.empty() {
		var zero T
		return zero, false
	}
	return d.Buf[0], true
}

func (d *Deque[T]) Back() (T, bool) {
	if d.empty() {
		var zero T
		return zero, false
	}
	return d.Buf[len(d.Buf)-1], true
}

// Note that this expects that the start and stop is already validated
func (d *Deque[T]) GetSlice(start int64, stop int64) []T {
	return d.Buf[start : stop+1]
}

func (d *Deque[T]) Remove(el T) bool {
	for i, ch := range d.Buf {
		if ch == el {
			// Found the element to remove.
			// Return a new slice that concatenates the part before the index
			// and the part after the index.
			d.Buf = append(d.Buf[:i], d.Buf[i+1:]...)
			return true
		}
	}
	// If the element was not found, return false.
	return false
}
