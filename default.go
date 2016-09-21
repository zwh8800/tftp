package tftp

import "io"

var (
	DefaultServer   *Server
	DefaultServeMux *ServeMux
)

func init() {
	DefaultServeMux = NewServeMux()
	DefaultServer = &Server{
		Handler: DefaultServeMux,
	}
}

// Handle registers the handler for given path to the DefaultServerMux
func Handle(path string, handler Handler) {
	DefaultServeMux.Handle(path, handler)
}

// HandleFunc registers the read request handler and write request handler
// for given path to the DefaultServerMux.
// both rrqHandler and wrqHandler could be nil (NotFoundHandler will be used)
func HandleFunc(path string,
	rrqHandler func(w io.WriteCloser, req *Request) error,
	wrqHandler func(r io.Reader, req *Request) error) {
	DefaultServeMux.HandleFunc(path, rrqHandler, wrqHandler)
}

// ListenAndServe listen on the UDP network address addr and then
// Serve with handler. Handler is typically nil, in which case the
// DefaultServerMux is used
func ListenAndServe(addr string, handler Handler) error {
	DefaultServer.Addr = addr
	if handler == nil {
		DefaultServer.Handler = DefaultServeMux
	}
	return DefaultServer.ListenAndServe()
}
