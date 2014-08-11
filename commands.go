package ftptest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/jehiah/go-strftime"
)

// Command interface
type Command interface {
	RequiresParams() bool
	RequiresAuth() bool
	Execute(conn *Connection, args string) error
}

// Commands map
var commands = map[string]Command{
	"ALLO": &commandALLO{},
	"CDUP": &commandCDUP{},
	"CWD":  &commandCWD{},
	"DELE": &commandDELE{},
	"EPRT": &commandEPRT{},
	"EPSV": &commandEPSV{},
	"LIST": &commandLIST{},
	"NLST": &commandNLST{},
	"MDTM": &commandMDTM{},
	"MKD":  &commandMKD{},
	"MODE": &commandMODE{},
	"NOOP": &commandNOOP{},
	"PASS": &commandPASS{},
	"PASV": &commandPASV{},
	"PORT": &commandPORT{},
	"PWD":  &commandPWD{},
	"QUIT": &commandQUIT{},
	"RETR": &commandRETR{},
	"RNFR": &commandRNFR{},
	"RNTO": &commandRNTO{},
	"RMD":  &commandRMD{},
	"SIZE": &commandSIZE{},
	"STOR": &commandSTOR{},
	"STRU": &commandSTRU{},
	"SYST": &commandSYST{},
	"TYPE": &commandTYPE{},
	"USER": &commandUSER{},
	"XCUP": &commandCDUP{},
	"XCWD": &commandCWD{},
	"XPWD": &commandPWD{},
	"XRMD": &commandRMD{},
}

// ALLO FTP commmand is a ping
type commandALLO struct{}

func (c *commandALLO) RequiresParams() bool {
	return false
}

func (c *commandALLO) RequiresAuth() bool {
	return false
}

func (c *commandALLO) Execute(conn *Connection, args string) error {
	conn.WriteMessage(202, "Obsolete")
	return nil
}

// CDUP goes to the parent directory.
type commandCDUP struct{}

func (c *commandCDUP) RequiresParams() bool {
	return false
}

func (c *commandCDUP) RequiresAuth() bool {
	return true
}

func (c *commandCDUP) Execute(conn *Connection, args string) error {
	cmd := &commandCWD{}
	return cmd.Execute(conn, "..")
}

// CWD command changes the current working directory
type commandCWD struct{}

func (c *commandCWD) RequiresParams() bool {
	return true
}

func (c *commandCWD) RequiresAuth() bool {
	return true
}

func (c *commandCWD) Execute(conn *Connection, args string) error {
	err := conn.ChangeWorkingDirectory(args)
	if err != nil {
		return err
	}

	conn.WriteMessage(250, "Directory changed to "+conn.WorkingDirectory)
	return nil
}

// DELE allows client to remove a file
type commandDELE struct{}

func (c *commandDELE) RequiresParams() bool {
	return true
}

func (c *commandDELE) RequiresAuth() bool {
	return true
}

func (c *commandDELE) Execute(conn *Connection, args string) error {
	path := conn.BuildPath(args)
	err := conn.Filesystem.Remove(path)
	if err != nil {
		return err
	}

	conn.WriteMessage(250, "File deleted")
	return nil
}

// EPRT is an advanced PORT command with IPv6 support
type commandEPRT struct{}

func (c *commandEPRT) RequiresParams() bool {
	return true
}

func (c *commandEPRT) RequiresAuth() bool {
	return true
}

func (c *commandEPRT) Execute(conn *Connection, args string) error {
	delimiter := string(args[0:1])
	parts := strings.Split(args, delimiter)

	if len(parts) != 4 {
		return ErrInvalidMessage
	}

	addressFamily, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}

	if addressFamily != 1 && addressFamily != 2 {
		return ErrProtocolNotSupported
	}

	host := parts[2]
	port, err := strconv.Atoi(parts[3])
	if err != nil {
		return err
	}

	socket, err := NewActiveSocket(host, port)
	if err != nil {
		return err
	}

	conn.DataSocket = socket

	conn.WriteMessage(200, "Connection estabilished ("+parts[3]+")")
	return nil
}

// EPSV is a modern version of the PASV command that includes IPv6 support,
// which is not supported in FTPTest
type commandEPSV struct{}

func (c *commandEPSV) RequiresParams() bool {
	return false
}

func (c *commandEPSV) RequiresAuth() bool {
	return true
}

func (c *commandEPSV) Execute(conn *Connection, args string) error {
	socket, err := NewPassiveSocket()
	if err != nil {
		return err
	}

	conn.DataSocket = socket

	conn.WriteMessage(200, "Entering Extended Passive Mode (|||"+strconv.Itoa(socket.GetPort())+"|)")
	return nil
}

// LIST returns a listing of the directory contents
type commandLIST struct{}

func (c *commandLIST) RequiresParams() bool {
	return false
}

func (c *commandLIST) RequiresAuth() bool {
	return true
}

func (c *commandLIST) Execute(conn *Connection, args string) error {
	conn.WriteMessage(150, "Opening ASCII mode data connection for file list")

	path := conn.BuildPath(args)

	contents, err := conn.Filesystem.DirContents(path)
	if err != nil {
		contents = make([]*File, 0)
	}

	var buffer bytes.Buffer
	for _, file := range contents {
		buffer.WriteString(
			file.ModeString() + " 1 owner group" +
				lpad(strconv.Itoa(int(file.Size)), 12) + " " +
				strftime.Format("%b %d %H:%M", file.TimeModified) + " " +
				file.Name + "\r\n",
		)
	}

	buffer.WriteString("\r\n")

	conn.SendDataThroughSocket(buffer.String())
	return nil
}

// NLST returns a list of filenames in CWD
type commandNLST struct{}

func (c *commandNLST) RequiresParams() bool {
	return false
}

func (c *commandNLST) RequiresAuth() bool {
	return true
}

func (c *commandNLST) Execute(conn *Connection, args string) error {
	conn.WriteMessage(150, "Opening ASCII mode data connection for file list")

	path := conn.BuildPath(args)

	contents, err := conn.Filesystem.DirContents(path)
	if err != nil {
		contents = make([]*File, 0)
	}

	var buffer bytes.Buffer
	for _, file := range contents {
		buffer.WriteString(
			file.Name + "\r\n",
		)
	}

	buffer.WriteString("\r\n")

	conn.SendDataThroughSocket(buffer.String())
	return nil
}

// MDTM returns time when the file was last modified
type commandMDTM struct{}

func (c *commandMDTM) RequiresParams() bool {
	return true
}

func (c *commandMDTM) RequiresAuth() bool {
	return true
}

func (c *commandMDTM) Execute(conn *Connection, args string) error {
	path := conn.BuildPath(args)

	time, err := conn.Filesystem.LastModified(path)
	if err == nil {
		conn.WriteMessage(213, strftime.Format("%Y%m%d%H%M%S", time))
	} else {
		conn.WriteMessage(450, "File not available")
	}

	return nil
}

// MKD creates a new directory
type commandMKD struct{}

func (c *commandMKD) RequiresParams() bool {
	return false
}

func (c *commandMKD) RequiresAuth() bool {
	return true
}

func (c *commandMKD) Execute(conn *Connection, args string) error {
	path := conn.BuildPath(args)

	if err := conn.Filesystem.MkDir(path); err == nil {
		conn.WriteMessage(257, "Directory created")
	} else {
		conn.WriteMessage(550, "Action not taken")
	}

	return nil
}

// MODE sets the socket transmission mode
type commandMODE struct{}

func (c commandMODE) RequiresParams() bool {
	return true
}

func (c commandMODE) RequiresAuth() bool {
	return true
}

func (c commandMODE) Execute(conn *Connection, args string) error {
	if strings.ToUpper(args) == "S" {
		conn.WriteMessage(200, "OK")
	} else {
		conn.WriteMessage(504, "MODE is an obsolete command")
	}

	return nil
}

// NOOP is an another ping command
type commandNOOP struct{}

func (c *commandNOOP) RequiresParams() bool {
	return false
}

func (c *commandNOOP) RequiresAuth() bool {
	return false
}

func (c *commandNOOP) Execute(conn *Connection, args string) error {
	conn.WriteMessage(200, "OK")
	return nil
}

// PASS placeholder command. Processes the password.
type commandPASS struct{}

func (c *commandPASS) RequiresParams() bool {
	return true
}

func (c *commandPASS) RequiresAuth() bool {
	return false
}

func (c *commandPASS) Execute(conn *Connection, args string) error {
	conn.Authenticated = true
	conn.WriteMessage(230, "Password ok, continue")
	return nil
}

// PASV starts a new passive mode connection
type commandPASV struct{}

func (c *commandPASV) RequiresParams() bool {
	return false
}

func (c *commandPASV) RequiresAuth() bool {
	return true
}

func (c *commandPASV) Execute(conn *Connection, args string) error {
	socket, err := NewPassiveSocket()
	if err != nil {
		conn.WriteMessage(425, "Data connection failed")
		return nil
	}

	conn.DataSocket = socket

	p1 := socket.GetPort() / 256
	p2 := socket.GetPort() - (p1 * 256)

	quads := strings.Split(socket.GetHost(), ".")
	target := fmt.Sprintf("(%s,%s,%s,%s,%d,%d)", quads[0], quads[1], quads[2], quads[3], p1, p2)
	msg := "Entering Passive Mode " + target

	conn.WriteMessage(227, msg)
	return nil
}

// PORT starts a new active mode connection
type commandPORT struct{}

func (c commandPORT) RequiresParams() bool {
	return true
}

func (c commandPORT) RequiresAuth() bool {
	return true
}

func (c commandPORT) Execute(conn *Connection, args string) error {
	nums := strings.Split(args, ",")

	portOne, _ := strconv.Atoi(nums[4])
	portTwo, _ := strconv.Atoi(nums[5])
	port := (portOne * 256) + portTwo

	host := nums[0] + "." + nums[1] + "." + nums[2] + "." + nums[3]

	if host == "127.0.0.1" {
		host = strings.Split(conn.Connection.RemoteAddr().String(), ":")[0]
	}

	socket, err := NewActiveSocket(host, port)
	if err != nil {
		conn.WriteMessage(425, "Data connection failed")
		return nil
	}

	conn.DataSocket = socket
	conn.WriteMessage(200, "Connection established ("+strconv.Itoa(port)+")")
	return nil
}

// PWD tells the client current working directory
type commandPWD struct{}

func (c *commandPWD) RequiresParams() bool {
	return false
}

func (c *commandPWD) RequiresAuth() bool {
	return true
}

func (c *commandPWD) Execute(conn *Connection, args string) error {
	conn.WriteMessage(257, "\""+conn.WorkingDirectory+"\" is the current directory")
	return nil
}

// QUIT closes the connection
type commandQUIT struct{}

func (c *commandQUIT) RequiresParams() bool {
	return false
}

func (c *commandQUIT) RequiresAuth() bool {
	return false
}

func (c *commandQUIT) Execute(conn *Connection, args string) error {
	return ErrQuitRequest
}

// RETR downloads a file
type commandRETR struct{}

func (c *commandRETR) RequiresParams() bool {
	return true
}

func (c *commandRETR) RequiresAuth() bool {
	return true
}

func (c *commandRETR) Execute(conn *Connection, args string) error {
	path := conn.BuildPath(args)

	file, err := conn.Filesystem.ReadFile(path)
	if err != nil {
		conn.WriteMessage(551, "File not available")
		return nil
	}

	conn.WriteMessage(150, "Data transfer starting "+strconv.Itoa(file.Size)+" bytes")

	conn.DataSocket.Write(file.Content)
	conn.DataSocket.Close()

	conn.WriteMessage(226, "Closing data connection, sent "+strconv.Itoa(file.Size)+" bytes")
	return nil
}

// RNFR is a request for a file rename
type commandRNFR struct{}

func (c *commandRNFR) RequiresParams() bool {
	return false
}

func (c *commandRNFR) RequiresAuth() bool {
	return true
}

func (c *commandRNFR) Execute(conn *Connection, args string) error {
	conn.RenameFrom = conn.BuildPath(args)

	conn.WriteMessage(350, "Requested file action pending further information.")
	return nil
}

// Second part of the rename action
type commandRNTO struct{}

func (c *commandRNTO) RequiresParams() bool {
	return false
}

func (c *commandRNTO) RequiresAuth() bool {
	return true
}

func (c *commandRNTO) Execute(conn *Connection, args string) error {
	path := conn.BuildPath(args)

	if err := conn.Filesystem.Rename(conn.RenameFrom, path); err == nil {
		conn.WriteMessage(250, "File renamed")
	} else {
		conn.WriteMessage(550, "Action not taken")
	}

	return nil
}

// RMD removes a directory
type commandRMD struct{}

func (c *commandRMD) RequiresParams() bool {
	return false
}

func (c *commandRMD) RequiresAuth() bool {
	return true
}

func (c *commandRMD) Execute(conn *Connection, args string) error {
	path := conn.BuildPath(args)

	if err := conn.Filesystem.RmDir(path); err == nil {
		conn.WriteMessage(250, "Directory deleted")
	} else {
		conn.WriteMessage(550, "Action not taken")
	}

	return nil
}

// SIZE returns bytes count of a file
type commandSIZE struct{}

func (c *commandSIZE) RequiresParams() bool {
	return true
}

func (c *commandSIZE) RequiresAuth() bool {
	return true
}

func (c *commandSIZE) Execute(conn *Connection, args string) error {
	path := conn.BuildPath(args)

	bytes, err := conn.Filesystem.Size(path)
	if err != nil {
		conn.WriteMessage(450, "file not available")
		return nil
	}

	conn.WriteMessage(213, strconv.Itoa(bytes))
	return nil
}

// STOR allows uploading new files
type commandSTOR struct{}

func (c *commandSTOR) RequiresParams() bool {
	return true
}

func (c *commandSTOR) RequiresAuth() bool {
	return true
}

func (c *commandSTOR) Execute(conn *Connection, args string) error {
	path := conn.BuildPath(args)

	conn.WriteMessage(150, "Data transfer starting")

	data, err := ioutil.ReadAll(conn.DataSocket)
	if err != nil {
		conn.WriteMessage(450, "Error during transfer")
		return nil
	}

	err = conn.Filesystem.WriteFile(path, data)
	if err != nil {
		conn.WriteMessage(550, "Action not taken")
		return nil
	}

	conn.WriteMessage(226, "OK, received "+strconv.Itoa(len(data))+" bytes")
	return nil
}

// STRU is an obsolete command, only one parameter is used nowadays.
type commandSTRU struct{}

func (c *commandSTRU) RequiresParams() bool {
	return true
}

func (c *commandSTRU) RequiresAuth() bool {
	return true
}

func (c *commandSTRU) Execute(conn *Connection, args string) error {
	if strings.ToUpper(args) == "F" {
		conn.WriteMessage(200, "OK")
	} else {
		conn.WriteMessage(504, "STRU is an obsolete command")
	}

	return nil
}

// SYST always responds with same message
type commandSYST struct{}

func (c *commandSYST) RequiresParams() bool {
	return true
}

func (c *commandSYST) RequiresAuth() bool {
	return true
}

func (c *commandSYST) Execute(conn *Connection, args string) error {
	conn.WriteMessage(215, "UNIX Type: L8")
	return nil
}

// We only care about binary more though.
type commandTYPE struct{}

func (c *commandTYPE) RequiresParams() bool {
	return false
}

func (c *commandTYPE) RequiresAuth() bool {
	return true
}

func (c *commandTYPE) Execute(conn *Connection, args string) error {
	if strings.ToUpper(args) == "A" {
		conn.WriteMessage(200, "Type set to ASCII")
	} else if strings.ToUpper(args) == "I" {
		conn.WriteMessage(200, "Type set to binary")
	} else {
		conn.WriteMessage(500, "Invalid type")
	}

	return nil
}

// First part of the authentication
type commandUSER struct{}

func (c *commandUSER) RequiresParams() bool {
	return true
}

func (c *commandUSER) RequiresAuth() bool {
	return false
}

func (c *commandUSER) Execute(conn *Connection, args string) error {
	conn.WriteMessage(331, "User name ok, password required")
	return nil
}
