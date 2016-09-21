package tftp_test

import (
	"bytes"
	"io"
	"log"
	"os"
	"testing"

	"github.com/zwh8800/tftp"
)

type TestHandler struct{}

func (t *TestHandler) ServeTFTPReadRequest(w io.WriteCloser, req *tftp.Request) error {
	log.Println(req)
	w.Write([]byte("Hello world\nnihao"))
	f, err := os.Open("/Users/zzz/66.go")
	if err != nil {
		log.Panic(err)
	}
	io.Copy(w, f)

	w.Close()

	return nil
}

func (t *TestHandler) ServeTFTPWriteRequest(r io.Reader, req *tftp.Request) error {
	log.Println(req)
	var buf bytes.Buffer
	io.Copy(&buf, r)
	log.Println(buf.Len(), " bytes received:", buf.String())

	return nil
}

func TestServer(t *testing.T) {
	s := &tftp.Server{
		Addr:    ":1024",
		Handler: &TestHandler{},
	}
	if err := s.ListenAndServe(); err != nil {
		t.Error(err)
	}
}
