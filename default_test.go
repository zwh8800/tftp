package tftp_test

import (
	"bytes"
	"io"
	"log"
	"os"
	"testing"

	"github.com/zwh8800/tftp"
)

func TestDefault(t *testing.T) {
	tftp.HandleFunc("1", func(w io.WriteCloser, req *tftp.Request) error {
		log.Println(req)
		w.Write([]byte("Hello world\nnihao"))
		f, err := os.Open("/Users/zzz/66.go")
		if err != nil {
			log.Panic(err)
		}
		io.Copy(w, f)

		w.Close()

		return nil
	}, nil)
	tftp.HandleFunc("2", nil, func(r io.Reader, req *tftp.Request) error {
		log.Println(req)
		var buf bytes.Buffer
		io.Copy(&buf, r)
		log.Println(buf.Len(), " bytes received:", buf.String())

		return nil
	})
	tftp.HandleFunc("3", nil, nil)

	tftp.ListenAndServe(":1024", nil)
}
