package tftp

import (
	"io"
	"net/http"
	"os"
)

type fileHandler struct {
	root http.FileSystem
}

func ReadonlyFileServer(fs http.FileSystem) Handler {
	return &fileHandler{root: fs}
}

func (h *fileHandler) ServeTFTPReadRequest(w io.WriteCloser, req *Request) error {
	f, err := h.root.Open(req.Filename)
	if err != nil {
		return toTFTPError(err)
	}
	stat, err := f.Stat()
	if err != nil {
		return toTFTPError(err)
	}
	if stat.IsDir() {
		return ErrFileNotFound
	}
	if _, err := io.Copy(w, f); err != nil {
		return err
	}
	w.Close()
	return nil
}

func (h *fileHandler) ServeTFTPWriteRequest(r io.Reader, req *Request) error {
	return ErrAccessViolation
}

func toTFTPError(err error) *TFTPError {
	if os.IsNotExist(err) {
		return ErrFileNotFound
	}
	if os.IsPermission(err) {
		return ErrAccessViolation
	}
	return ErrNotDefined
}
