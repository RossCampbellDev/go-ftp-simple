package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strings"
	"sync"
)

type FtpServer struct {
	port       int
	portString string
	clients    map[int]net.Conn
}

func NewFtpServer() *FtpServer {
	return &FtpServer{clients: make(map[int]net.Conn)}
}

func (f *FtpServer) Listen(port int) {
	f.port = port
	f.portString = fmt.Sprintf(":%d", port)

	ln, err := net.Listen("tcp", f.portString)
	if err != nil {
		panic(err)
	}
	fmt.Printf("listening on %d...\n", port)

	var wg sync.WaitGroup
	for {
		wg.Add(1)
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		id := f.newClient(conn)
		go f.parseCommand(conn, &wg, id)
		wg.Wait()
	}
}

func (f *FtpServer) parseCommand(conn net.Conn, wg *sync.WaitGroup, id int) {
	defer wg.Done()
	userInput := make([]byte, 512) // TODO: not likely to break but still bad solution.  refactor.
	_, err := conn.Read(userInput) // blocks the program but we need waitgroups if more than one connection
	if err != nil {
		panic(err) // replace with channel?
	}

	// TODO: bad bodge.  refactor.
	endOfWord := bytes.IndexByte(userInput, 0)

	command := string(userInput[:endOfWord])

	fmt.Printf("command was: %s\n", command)
	switch strings.ToUpper(command) {
	case "QUIT", "EXIT":
		f.exitClient(id)
	case "DEL":
		f.deleteFile("test")
	case "GET":
		f.sendFileToClient("test")
	case "LS":
		f.listFiles()
	case "PUT":
		f.retrieveFileFromClient("test")
	default:
		fmt.Println("Invalid Command")
	}

	f.sendResponse([]byte{0}, id) // bodge?
}

func (f *FtpServer) sendResponse(b []byte, id int) {
	f.clients[id].Write(b)
}

func (f *FtpServer) readFile(conn net.Conn) {
	readBuffer := new(bytes.Buffer)
	for {
		var size int64
		binary.Read(conn, binary.LittleEndian, &size) // retrieve the size of incoming stream

		_, err := io.CopyN(readBuffer, conn, size)
		if err != nil {
			panic(err) // replace with sending error to a channel?
		}

		fmt.Println("File Received")
	}
}

func (f *FtpServer) newClient(conn net.Conn) int {
	// TODO: could result in over-writing connecting.  refactor.
	id := rand.Intn(10000)
	f.clients[id] = conn
	fmt.Printf("...connection accepted from %s\n", f.clients[id].RemoteAddr().String())
	return id
}

func (f *FtpServer) exitClient(id int) {
	f.clients[id].Close()
	delete(f.clients, id)
	fmt.Println("exiting server...")
}

func (f *FtpServer) deleteFile(filename string) {
	fmt.Println("delete", filename)
}

func (f *FtpServer) sendFileToClient(filename string) {
	fmt.Println("get", filename)
}

func (f *FtpServer) listFiles() {
	fmt.Println("list")
}

func (f *FtpServer) retrieveFileFromClient(filename string) {
	fmt.Println("put", filename)
}
