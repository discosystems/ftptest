package ftptest

import (
	"net"
	"strconv"
	"time"
)

type DataSocket interface {
	GetHost() string
	GetPort() int
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
}

type ActiveSocket struct {
	Connection net.Conn
	Host       string
	Port       int
}

func NewActiveSocket(host string, port int) (DataSocket, error) {
	addr, err := net.ResolveTCPAddr("tcp", host+":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}

	connection, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}

	socket := &ActiveSocket{}
	socket.Connection = connection
	socket.Host = host
	socket.Port = port

	return socket, nil
}

func (a *ActiveSocket) GetHost() string {
	return a.Host
}

func (a *ActiveSocket) GetPort() int {
	return a.Port
}

func (a *ActiveSocket) Read(p []byte) (int, error) {
	return a.Connection.Read(p)
}

func (a *ActiveSocket) Write(p []byte) (int, error) {
	return a.Connection.Write(p)
}

func (a *ActiveSocket) Close() error {
	return a.Connection.Close()
}

type PassiveSocket struct {
	Connection net.Conn
	Port       int
}

func NewPassiveSocket() (DataSocket, error) {
	socket := &PassiveSocket{}

	go socket.ListenAndServe()

	for {
		if socket.GetPort() > 0 {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return socket, nil
}

func (p *PassiveSocket) GetHost() string {
	return "127.0.0.1"
}

func (p *PassiveSocket) GetPort() int {
	return p.Port
}

func (p *PassiveSocket) Read(b []byte) (int, error) {
	if p.waitUntilOpen() == false {
		return 0, ErrDataSocketUnavailable
	}

	return p.Connection.Read(b)
}

func (p *PassiveSocket) Write(b []byte) (int, error) {
	if p.waitUntilOpen() == false {
		return 0, ErrDataSocketUnavailable
	}

	return p.Connection.Write(b)
}

func (p *PassiveSocket) Close() error {
	if p.Connection != nil {
		return p.Connection.Close()
	}

	return nil
}

func (p *PassiveSocket) ListenAndServe() {
	address, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return
	}

	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return
	}

	p.Port = listener.Addr().(*net.TCPAddr).Port

	connection, err := listener.AcceptTCP()
	if err != nil {
		return
	}

	p.Connection = connection
}

func (p *PassiveSocket) waitUntilOpen() bool {
	retries := 0

	for {
		if p.Connection != nil {
			break
		}

		if retries > 3 {
			return false
		}

		time.Sleep(500 * time.Millisecond)
		retries += 1
	}

	return true
}
