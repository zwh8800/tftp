# TFTP: a golang TFTP implementation

[![GoDoc][1]][2] [![MIT licensed][3]][4]

[1]: https://godoc.org/github.com/zwh8800/tftp?status.svg
[2]: https://godoc.org/github.com/zwh8800/tftp
[3]: https://img.shields.io/badge/license-MIT-blue.svg
[4]: LICENSE

This package provides TFTP server implementations.

## How to get this package

```
go get github.com/zwh8800/tftp
```

there is no other dependencies except golang standard library.

## Usage:

#### 1. Start a file server

```golang
log.Panic(tftp.ListenAndServe(":1024", tftp.ReadonlyFileServer(http.Dir("/Users/zzz/Downloads"))))
```

#### 2. handle some specific path

```golang
tftp.Handle("uboot.bin", someHandler)

// only handle "read" operation
tftp.HandleFunc("kernel.bin", func(w io.WriteCloser, req *tftp.Request) error {
    log.Println("incoming read operation for kernel.bin:", req)
    f, _ := os.Open("someFileToRead")
    io.Copy(w, f)
    f.Close()   // important

    return nil
}, nil)

// only handle "write" operation
tftp.HandleFunc("fs.bin", nil, func(r io.Reader, req *tftp.Request) nil {
    log.Println("incoming write operation for fs.bin:", req)
    f, _ := os.Create("someFileToWrite")
    io.Copy(f, r)

    return nil
})

// handle "fs.secure.bin" path and return a ErrAccessViolation error (operation not allowed)
tftp.HandleFunc("fs.secure.bin", nil, func(r io.Reader, req *tftp.Request) nil {
    log.Println("incoming write operation for fs.secure.bin, NOT ALLOWED:", req)

    return tftp.ErrAccessViolation
})

log.Panic(tftp.ListenAndServe(":1024", nil))
```

#### 3. more control

```golang
server := &tftp.Server{
    Addr: ":1234",
    Handler: someHandler,
    Timeout: 10 * time.Second,
}
log.Fatal(server.ListenAndServe())
```
