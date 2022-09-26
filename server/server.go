package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	maxConn         = 10
	logoFilePath    = "static/logo.txt"
	PathMessageLogs = "logs/"
	UsernameLimit   = 32
)

type Server struct {
	Addr            string
	OpenConnections map[net.Conn]string
	NewConnections  chan net.Conn
	DeadConnections chan net.Conn
	MsgHistoryFile  string
	ClientCount     int
	mu              sync.Mutex
}

func NewServer(addr string) *Server {
	return &Server{
		Addr:            addr,
		OpenConnections: make(map[net.Conn]string),
		NewConnections:  make(chan net.Conn, 1),
		DeadConnections: make(chan net.Conn, 1),
	}
}

func (s *Server) HandleConnection() {
	listener, err := net.Listen("tcp", ":"+s.Addr)
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}

	fmt.Printf("Listening on the port: %s\n", s.Addr)
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("error on accepting the connection: %s\n", err)
				conn.Close()
				continue
			}
			if s.ClientCount == maxConn {
				conn.Write([]byte(fmt.Sprintf("Maximum %v connections. Sorry\n", maxConn)))
				conn.Close()
			} else {
				s.NewConnections <- conn
				s.mu.Lock()
				s.ClientCount++
				s.mu.Unlock()
			}
		}
	}()

	s.MsgHistoryFile = PathMessageLogs + time.Now().Format("2006-01-02 15:04:05") + ".log"
	file, err := os.OpenFile(s.MsgHistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("file for message logs: %s\n", err)
	}
	defer file.Close()

	for {
		select {
		case conn := <-s.NewConnections:
			go s.broadcastMsg(conn, file)
		case conn := <-s.DeadConnections:
			left := s.OpenConnections[conn]
			s.logMessages(file, left+" has left the chat")
			s.deleteUser(conn)
			for item, name := range s.OpenConnections {
				if _, err := item.Write([]byte(fmt.Sprintf("%s\r%s has left our chat...\n%s", s.clearLine(s.getMsgInfo(name)), left, s.getMsgInfo(name)))); err != nil {
					s.deleteUser(item)
				}
			}
		}
	}
}

func (s *Server) broadcastMsg(conn net.Conn, file io.Writer) {
	username := s.signUser(conn)
	s.logMessages(file, username+" has joined the chat")

	msgHistory, err := os.ReadFile(s.MsgHistoryFile)
	if err != nil {
		log.Printf("can't get the message history: %s\n", err)

		if _, err := conn.Write([]byte("Message history hasn't been loaded. Sorry\n")); err != nil {
			s.deleteUser(conn)
		}
	} else {
		conn.Write([]byte(msgHistory))
	}

	for item, name := range s.OpenConnections {
		if item != conn {
			if _, err := item.Write([]byte(fmt.Sprintf(s.clearLine(s.getMsgInfo(name))+"\r%s has joined our chat\n%s", username, s.getMsgInfo(name)))); err != nil {
				s.deleteUser(item)
			}
		}
	}

	reader := bufio.NewReader(conn)
	for {
		msgInfo := s.getMsgInfo(username)
		conn.Write([]byte(msgInfo))

		msg, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if msg == "\n" {
			continue
		}

		s.logMessages(file, msgInfo+msg[:len(msg)-1])

		for item, name := range s.OpenConnections {
			if item != conn {
				if _, err := item.Write([]byte(s.clearLine(s.getMsgInfo(name)) + msgInfo + msg + s.getMsgInfo(name))); err != nil {
					s.deleteUser(item)
				}
			}
		}
	}
	s.DeadConnections <- conn
}

func (s *Server) getMsgInfo(username string) string {
	return fmt.Sprintf("\r[%s][%s]: ", time.Now().Format("2006-01-02 15:04:05"), username)
}

func (s *Server) clearLine(line string) string {
	return "\r" + strings.Repeat(" ", len(line)) + "\r"
}

func (s *Server) logMessages(file io.Writer, msg string) {
	logger := log.New(file, "", log.LstdFlags)
	logger.Println(msg)
}

func (s *Server) deleteUser(conn net.Conn) {
	s.mu.Lock()
	delete(s.OpenConnections, conn)
	s.mu.Unlock()
}
