package server

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/RossCampbellDev/go-ftp-simple/internal/types"
)

type FtpServer struct {
	port       int
	portString string
	clients    map[int]types.Client
}

func NewFtpServer() *FtpServer {
	return &FtpServer{}
}

/*
testing:

	our listener = net.Listen with protocol and port
	loop:  listener.Accept() - which returns a net.Conn
	on a net.Conn we can call Read, and supply a byte-slice as a buffer to read into
*/
func (f *FtpServer) Listen(port int) {
	f.port = port
	f.portString = fmt.Sprintf(":%d", port)

	ln, err := net.Listen("tcp", f.portString)
	if err != nil {
		panic(err)
	}
	fmt.Printf("listening on %d\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go f.readLoop(conn)
	}
}

/*
testing:

	naive reading of data - not streamed
*/
func (f *FtpServer) readLoop(conn net.Conn) {
	// readBuffer := make([]byte, 2048)
	readBuffer := new(bytes.Buffer)
	for {
		var size int64
		binary.Read(conn, binary.LittleEndian, &size)
		// n, err := conn.Read(readBuffer)
		n, err := io.CopyN(readBuffer, conn, size) // src, dst.  will endlessly copy until an EOF signal is reached.  that's a problem!  use CopyN instead
		if err != nil {
			panic(err)
		}
		// dataReceived := readBuffer[:n] // n is num bytes received
		// fmt.Println(dataReceived)
		fmt.Println(readBuffer.Bytes())
		fmt.Printf("received %d bytes\n", n)
	}
}

/*
testing:

	ReadFull() - for the length of 'file' which is our byte slice, read random numbers into the slice
	start a net connection
	write to that connection, the contents of file
*/
func SendFile(size int, portString string) error {
	file := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, file) // reads len(file) bytes from <io.Reader> into file.
	if err != nil {
		return err
	}

	conn, err := net.Dial("tcp", portString) // assumes localhost if no IP before portString
	if err != nil {
		panic(err)
	}

	// add this.  tell the server side the size of the file before streaming over
	binary.Write(conn, binary.LittleEndian, int64(size))

	n, err := io.CopyN(conn, bytes.NewReader(file), int64(size)) // copy TO the connection.  convert file to fit the io.Reader interface
	// n, err := conn.Write(file)
	if err != nil {
		panic(err)
	}

	fmt.Printf("written %d bytes over\n", n)
	return nil
}

func (f *FtpServer) NewClient() {
	id := 1
	f.clients[id] = types.Client{IpAddr: "123"}
}

func (f *FtpServer) ExitClient(id int) {
	delete(f.clients, id)
	fmt.Println("exiting server...")
}

func (f *FtpServer) ReadCommand(command string) {
	switch command {
	case "bye", "quit", "exit":
		f.ExitClient(1)
	case "delete":
		f.DeleteFile("test")
	case "get":
		f.SendFileToClient("test")
	case "ls":
		f.ListFiles()
	case "put":
		f.RetrieveFileFromClient("test")
	}
}

func (f *FtpServer) DeleteFile(filename string) {
	fmt.Println("delete", filename)
}

func (f *FtpServer) SendFileToClient(filename string) {
	fmt.Println("get", filename)
}

func (f *FtpServer) ListFiles() {
	fmt.Println("list")
}

func (f *FtpServer) RetrieveFileFromClient(filename string) {
	fmt.Println("put", filename)
}
