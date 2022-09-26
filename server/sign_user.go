package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func (s *Server) signUser(conn net.Conn) string {
	content, err := os.ReadFile(logoFilePath)
	if err != nil {
		log.Printf("Error while getting a logo: %s", err)
	}
	welcomeMsg := fmt.Sprintf("Welcome to TCP-Chat!\n%s\n", string(content))
	nameMsg := "[ENTER YOUR NAME]: "

	conn.Write([]byte(welcomeMsg))

	reader := bufio.NewReader(conn)
	var ok bool
	var username, nameTips string

	for !ok {
		conn.Write([]byte(nameTips + nameMsg))
		username, err = reader.ReadString('\n')
		if err != nil {
			conn.Write([]byte("something went wrong while getting the username\n"))
		}
		username = strings.Trim(username, " \r\n")
		nameTips, ok = s.checkUsername(username)
	}
	s.mu.Lock()
	s.OpenConnections[conn] = username
	s.mu.Unlock()
	return username
}

func (s *Server) checkUsername(username string) (string, bool) {
	if username == "" {
		return "Username is required\n", false
	}
	if len(username) > UsernameLimit {
		return "Username length limit is 32 characters\n", false
	}
	for _, r := range username {
		if r >= 0 && r <= 32 || r == 127 {
			return "Username shouldn't contain not printable characters\n", false
		}
	}
	for item := range s.OpenConnections {
		if s.OpenConnections[item] == username {
			return "This username is taken. Choose another one please\n", false
		}
	}
	return "", true
}
