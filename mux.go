package tftp

import "io"

var (
	notFoundHandler = &funcHandler{
		rrqHandler: func(w io.WriteCloser, req *Request) error { return ErrFileNotFound },
		wrqHandler: func(r io.Reader, req *Request) error { return ErrFileNotFound },
	}
)

func NotFoundHandler() Handler {
	return notFoundHandler
}

type funcHandler struct {
	rrqHandler func(w io.WriteCloser, req *Request) error
	wrqHandler func(r io.Reader, req *Request) error
}

func (h *funcHandler) ServeTFTPReadRequest(w io.WriteCloser, req *Request) error {
	if h.rrqHandler != nil {
		return h.rrqHandler(w, req)
	}
	return notFoundHandler.rrqHandler(w, req)
}

func (h *funcHandler) ServeTFTPWriteRequest(r io.Reader, req *Request) error {
	if h.wrqHandler != nil {
		return h.wrqHandler(r, req)
	}
	return notFoundHandler.wrqHandler(r, req)
}

type ServeMux struct {
	handlers map[string]Handler
}

func NewServeMux() *ServeMux {
	return &ServeMux{handlers: make(map[string]Handler)}
}

// Handle registers the handler for given path
func (mux *ServeMux) Handle(path string, handler Handler) {
	mux.handlers[path] = handler
}

// HandleFunc registers the read request handler and write request handler
// for given path. both rrqHandler and wrqHandler could be nil(NotFoundHandler will be used)
func (mux *ServeMux) HandleFunc(path string,
	rrqHandler func(w io.WriteCloser, req *Request) error,
	wrqHandler func(r io.Reader, req *Request) error) {
	mux.handlers[path] = &funcHandler{rrqHandler: rrqHandler, wrqHandler: wrqHandler}
}

// ServeTFTPReadRequest implement Handler interface and
// dispatch TFTP read request to handler
func (mux *ServeMux) ServeTFTPReadRequest(w io.WriteCloser, req *Request) error {
	h, ok := mux.handlers[req.Filename]
	if ok {
		return h.ServeTFTPReadRequest(w, req)
	} else {
		return notFoundHandler.ServeTFTPReadRequest(w, req)
	}
}

// ServeTFTPReadRequest implement Handler interface and
// dispatch TFTP write request to handler
func (mux *ServeMux) ServeTFTPWriteRequest(r io.Reader, req *Request) error {
	h, ok := mux.handlers[req.Filename]
	if ok {
		return h.ServeTFTPWriteRequest(r, req)
	} else {
		return notFoundHandler.ServeTFTPWriteRequest(r, req)
	}
}
