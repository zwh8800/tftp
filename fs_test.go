package tftp_test

import (
	"net/http"
	"testing"

	"github.com/zwh8800/tftp"
)

func TestReadonlyFileServer(t *testing.T) {
	s := tftp.Server{
		Addr:    ":1024",
		Handler: tftp.ReadonlyFileServer(http.Dir("/Users/zzz/Downloads")),
	}
	if err := s.ListenAndServe(); err != nil {
		t.Error(err)
	}
}
