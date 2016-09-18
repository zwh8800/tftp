package tftp

import (
	"net/http"
	"testing"
)

func TestReadonlyFileServer(t *testing.T) {
	s := Server{
		Addr:    ":1024",
		Handler: ReadonlyFileServer(http.Dir("/Users/zzz/Downloads")),
	}
	if err := s.ListenAndServe(); err != nil {
		t.Error(err)
	}
}
