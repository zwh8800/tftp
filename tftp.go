package tftp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

const (
	ModeNetascii = "netascii"
	ModeOctet    = "octet"
)

const (
	opReadRequest uint16 = iota + 1
	opWriteRequest
	opData
	opAck
	opError
)

const (
	errCodeNotDefined uint16 = iota
	errCodeFileNotFound
	errCodeAccessViolation
	errCodeDiskFull
	errCodeIllegalTFTPOperation
	errCodeUnknownTID
	errCodeFileAlreadyExists
	errCodeNoSuckUser

	errCodeFileNameTooLong
	errCodeFormatError
	errCodeUnknownMode
)

type TFTPError struct {
	code    uint16
	message string
}

func (e *TFTPError) Error() string {
	return fmt.Sprintf("TFTP error with code %d, message: %s", e.code, e.message)
}

var (
	ErrNotDefined           = &TFTPError{code: errCodeNotDefined}
	ErrFileNotFound         = &TFTPError{code: errCodeFileNotFound, message: "file not found"}
	ErrAccessViolation      = &TFTPError{code: errCodeAccessViolation, message: "access violation"}
	ErrDiskFull             = &TFTPError{code: errCodeDiskFull, message: "disk full"}
	errIllegalTFTPOperation = &TFTPError{code: errCodeIllegalTFTPOperation, message: "illegal TFTP operation"}
	errUnknownTID           = &TFTPError{code: errCodeUnknownTID, message: "unknown transfer ID"}
	ErrFileAlreadyExists    = &TFTPError{code: errCodeFileAlreadyExists, message: "file already exists"}

	errFileNameTooLong = &TFTPError{code: errCodeFileNameTooLong, message: "file name too long"}
	errFormatError     = &TFTPError{code: errCodeFormatError, message: "format error"}
	errUnknownMode     = &TFTPError{code: errCodeUnknownMode, message: "unknown mode"}
)

type Request struct {
	Mode     string
	Filename string
}

type Handler interface {
	ServeTFTPReadRequest(w io.WriteCloser, req *Request) error
	ServeTFTPWriteRequest(r io.Reader, req *Request) error
}

type Server struct {
	Addr    string        // TCP address to listen on, ":tftp" if empty
	Handler Handler       // handler to invoke
	Timeout time.Duration // maximum duration before timing out ack of the request

	// ErrorLog specifies an optional logger for errors accepting
	// connections and unexpected behavior from handlers.
	// If nil, logging goes to os.Stderr via the log package's
	// standard logger.
	ErrorLog *log.Logger
}

func (s *Server) ListenAndServe() error {
	addr := s.Addr
	if s.Addr == "" {
		addr = ":tftp"
	}
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	return s.Serve(conn)
}

func (s *Server) Serve(l *net.UDPConn) error {
	defer l.Close()

	var tempDelay time.Duration
	buf := make([]byte, 4096)

	for {
		_, peerAddr, err := l.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				s.logf("tftp: ReadFromUDP error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		op := binary.BigEndian.Uint16(buf[0:2])
		if op == opReadRequest || op == opWriteRequest {
			pos1 := bytes.IndexByte(buf[2:], 0)
			if pos1 == -1 {
				s.writeErrorPkt(l, peerAddr, errFormatError)
				continue
			}
			pos1 += 2
			filename := string(buf[2:pos1]) // assume utf-8

			pos2 := bytes.IndexByte(buf[pos1+1:], 0)
			if pos2 == -1 {
				s.writeErrorPkt(l, peerAddr, errFormatError)
				continue
			}
			pos2 += pos1 + 1

			mode := string(buf[pos1+1 : pos2])
			if mode != ModeNetascii && mode != ModeOctet {
				s.writeErrorPkt(l, peerAddr, errUnknownMode)
				continue
			}
			req := &Request{
				Filename: filename,
				Mode:     mode,
			}
			go s.serve(op, peerAddr, req)
		} else {
			s.writeErrorPkt(l, peerAddr, errIllegalTFTPOperation)
		}
	}
}

func (s *Server) writeErrorPkt(conn *net.UDPConn, addr *net.UDPAddr, err *TFTPError) {
	buf := make([]byte, 5+len(err.message))
	binary.BigEndian.PutUint16(buf[0:2], opError)
	binary.BigEndian.PutUint16(buf[2:4], err.code)
	copy(buf[4:], []byte(err.message))
	buf[len(buf)-1] = 0

	var n int
	var e error
	if addr == nil {
		n, e = conn.Write(buf)
	} else {
		n, e = conn.WriteToUDP(buf, addr)
	}
	if e != nil {
		s.logf("tftp: writeErrorPkt error: %v", e)
		return
	}
	if n != len(buf) {
		s.logf("tftp: writeErrorPkt error: write len != len(buf)")
	}
}

func (s *Server) logf(format string, v ...interface{}) {
	if s.ErrorLog == nil {
		s.ErrorLog = log.New(os.Stderr, "", log.LstdFlags)
	}
	s.ErrorLog.Output(2, fmt.Sprintf(format, v...))
}

func (s *Server) serve(op uint16, peerAddr *net.UDPAddr, req *Request) {
	conn, err := net.DialUDP("udp", nil, peerAddr)
	if err != nil {
		s.writeErrorPkt(conn, nil, ErrNotDefined)
		return
	}

	if op == opReadRequest {
		err = s.Handler.ServeTFTPReadRequest(newReadRequestWriter(s, conn, req.Mode), req)
	} else {
		err = s.Handler.ServeTFTPWriteRequest(newWriteRequestReader(s, conn, req.Mode), req)
	}
	if err != nil {
		tftpErr, ok := err.(*TFTPError)
		if !ok {
			*tftpErr = *ErrNotDefined
			tftpErr.message = err.Error()
		}

		s.writeErrorPkt(conn, nil, tftpErr)
		return
	}
}

type readWriterBase struct {
	conn *net.UDPConn
	s    *Server
	mode string
}

func newReadWriterBase(s *Server, conn *net.UDPConn, mode string) *readWriterBase {
	return &readWriterBase{
		conn: conn,
		s:    s,
		mode: mode,
	}
}

type readRequestWriter struct {
	*readWriterBase
	buf     bytes.Buffer
	blockNo uint16
}

func newReadRequestWriter(s *Server, conn *net.UDPConn, mode string) *readRequestWriter {
	base := newReadWriterBase(s, conn, mode)
	return &readRequestWriter{
		readWriterBase: base,
		blockNo:        1,
	}
}

func (r *readRequestWriter) Write(p []byte) (int, error) {
	if r.mode == ModeNetascii {
		p = bytes.Replace(p, []byte("\r\n"), []byte("\n"), -1)
		p = bytes.Replace(p, []byte("\n"), []byte("\r\n"), -1)
	}

	i, n := 0, len(p)
	for ; n-i > 512; i += 512 {
		if err := r.writeDataPacket(p[i : i+512]); err != nil {
			return i, err
		}
	}
	r.buf.Write(p[i:n])

	return n, nil
}

func (r *readRequestWriter) Close() error {
	data := r.buf.Bytes()
	r.buf.Reset()

	if err := r.writeDataPacket(data); err != nil {
		return err
	}

	return r.conn.Close()
}

func (r *readRequestWriter) writeDataPacket(data []byte) error {
	buf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(buf[0:2], opData)
	binary.BigEndian.PutUint16(buf[2:4], r.blockNo)
	copy(buf[4:], data)

	_, err := r.conn.Write(buf)
	if err != nil {
		r.s.writeErrorPkt(r.conn, nil, ErrNotDefined)
		return err
	}

	for {
		n, err := r.conn.Read(buf)
		if err != nil {
			r.s.writeErrorPkt(r.conn, nil, ErrNotDefined)
			return err
		}
		if n < 4 {
			r.s.writeErrorPkt(r.conn, nil, errFormatError)
			return errFormatError
		}
		op := binary.BigEndian.Uint16(buf[0:2])
		if op != opAck {
			r.s.writeErrorPkt(r.conn, nil, errIllegalTFTPOperation)
			return errIllegalTFTPOperation
		}
		blockNo := binary.BigEndian.Uint16(buf[2:4])
		if blockNo == r.blockNo {
			break
		}
	}

	r.blockNo++
	return nil
}

type writeRequestReader struct {
	*readWriterBase
	zeroAckSent bool
	closed      bool
}

func newWriteRequestReader(s *Server, conn *net.UDPConn, mode string) *writeRequestReader {
	base := newReadWriterBase(s, conn, mode)
	return &writeRequestReader{
		readWriterBase: base,
		zeroAckSent:    false,
		closed:         false,
	}
}

func (w *writeRequestReader) ack(n uint16) error {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint16(buf[0:2], opAck)
	binary.BigEndian.PutUint16(buf[2:4], n)
	if _, err := w.conn.Write(buf); err != nil {
		w.close()
		return err
	}
	return nil
}

func (w *writeRequestReader) Read(p []byte) (int, error) {
	if w.closed {
		return 0, io.EOF
	}

	if !w.zeroAckSent {
		if err := w.ack(0); err != nil {
			return 0, err
		}
		w.zeroAckSent = true
	}

	// 516 = max TFTP data packet length
	buf := make([]byte, 516)
	n, err := w.conn.Read(buf)
	if err != nil {
		w.s.writeErrorPkt(w.conn, nil, ErrNotDefined)
		w.close()
		return 0, err
	}
	op := binary.BigEndian.Uint16(buf[0:2])
	blockNo := binary.BigEndian.Uint16(buf[2:4])
	if op != opData {
		w.s.writeErrorPkt(w.conn, nil, errIllegalTFTPOperation)
		w.close()
		return 0, errIllegalTFTPOperation
	}

	if err := w.ack(blockNo); err != nil {
		return 0, err
	}

	copy(p, buf[4:])

	if n < 516 {
		w.close()
	}
	return n - 4, nil
}

func (w *writeRequestReader) close() {
	w.conn.Close()
	w.closed = true
}
