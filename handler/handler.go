package handler

type Handler interface {
	Run(buf []byte) ([]byte, error)
}
