package iterator

type Iterator[T any] interface {
	Next() bool
	Item() T
	Err() error
}
