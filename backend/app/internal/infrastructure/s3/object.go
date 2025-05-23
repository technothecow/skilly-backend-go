package s3

type Object interface {
	Read(p []byte) (n int, err error)
	Close() error
}