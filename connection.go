package ftptest

import (
	"bufio"
	"net"
	"path/filepath"
	"strconv"
	"strings"
)

type Connection struct {
	Connection       net.Conn
	DataSocket       DataSocket
	Buffer           *bufio.Reader
	Filesystem       *Filesystem
	Authenticated    bool
	WorkingDirectory string

	RenameFrom string
	RenameTo   string
}

func (c *Connection) WriteMessage(code int, message string) {
	// Format the message
	sCode := strconv.Itoa(code)
	c.Connection.Write([]byte(sCode + " " + message + "\r\n"))
}

func (c *Connection) ChangeWorkingDirectory(path string) error {
	new_directory := filepath.Clean(
		strings.Replace(
			filepath.Join(c.WorkingDirectory, path),
			"..",
			"",
			-1,
		),
	)

	if err := c.Filesystem.DirExists(new_directory); err != nil {
		return err
	}

	c.WorkingDirectory = new_directory
	return nil
}

func (c *Connection) BuildPath(path string) string {
	path = filepath.Join(c.WorkingDirectory, path)

	if path[0] != '/' {
		path = "/" + path
	}

	return filepath.Clean(path)
}

func (c *Connection) SendDataThroughSocket(data string) {
	bytes := len(data)

	c.DataSocket.Write([]byte(data))
	c.DataSocket.Close()

	message := "Closing data connection, sent " + strconv.Itoa(bytes) + " bytes"
	c.WriteMessage(226, message)
}

func (c *Connection) ParseIncoming() (string, string, error) {
	// Read the next line
	line, err := c.Buffer.ReadString('\n')
	if err != nil {
		// Might be EOF
		return "", "", err
	}

	// Split command and arguments
	parts := strings.SplitN(strings.Trim(line, "\r\n"), " ", 2)

	if len(parts) == 1 {
		return parts[0], "", nil
	}

	// Return first argument - the command and rest - the parameters
	return parts[0], strings.TrimSpace(parts[1]), nil
}

func (c *Connection) Serve() {
	// Send a welcome message
	c.WriteMessage(220, "Welcome to a Go FTPTest server!")

	// Read commands
	for {
		// Parse the incoming command
		cmd, args, err := c.ParseIncoming()
		if err != nil {
			return
		}

		// Find the command
		command, ok := commands[cmd]
		if !ok {
			c.WriteMessage(500, "Command not found")
			continue
		}

		// Perform basic ACL checks
		if command.RequiresAuth() && !c.Authenticated {
			c.WriteMessage(530, "Not logged in")
			continue
		}

		// Check if command is valid
		if command.RequiresParams() && args == "" {
			c.WriteMessage(553, "This command requires params")
			continue
		}

		// Execute the command
		err = command.Execute(c, args)
		if err == ErrQuitRequest {
			c.Connection.Close()
			break
		} else if err != nil {
			c.WriteMessage(550, err.Error())
		}
	}
}
