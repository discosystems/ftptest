package ftptest

import (
	"bufio"
	"net"
	"strconv"
)

type FTPServer struct {
	Listener   net.Listener
	Filesystem *Filesystem
}

// Creates a new FTP server on random port
func NewFTPServer() (*FTPServer, error) {
	server := &FTPServer{}

	address, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	server.Listener = net.ListenTCP("tcp", address)
	server.Filesystem = &Filesystem{}
	return server, nil
}

// Address of the server to use in tests
func (f *FTPServer) Address() string {
	port := f.Listener.Addr().(*net.TCPAddr).Port
	return "127.0.0.1:" + strconv.Itoa(port)
}

// Run it in a goroutine
func (f *FTPServer) Listen() error {
	for {
		connection, err := f.Listener.Accept()
		if err != nil {
			return err
		}

		handler := &Connection{
			Connection:       connection,
			Filesystem:       f.Filesystem,
			Buffer:           bufio.NewReader(connection),
			WorkingDirectory: "/",
		}
		go handler.Serve()
	}
}
