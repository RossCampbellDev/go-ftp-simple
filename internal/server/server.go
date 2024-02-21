package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"
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
	wg.Add(1)
	go f.quitter(&wg)
	for {
		wg.Add(1)
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		id := f.newClient(conn)
		go f.parseCommand(conn, &wg, id)
	}
	// wg.Wait()
}

func (f *FtpServer) quitter(wg *sync.WaitGroup) {
	defer wg.Done()
	var input string
	for {
		fmt.Scanln(&input)
		if input == "quit" {
			for id := range f.clients {
				f.sendResponse("goodbye", id) // TODO: not being listened for at the other end.  client only listens when it's sent a command
			}
			os.Exit(0)
		}
	}
}

func (f *FtpServer) parseCommand(conn net.Conn, wg *sync.WaitGroup, id int) {
	defer wg.Done()
	sentinel := false

	for !sentinel {
		userInput := make([]byte, 512) // TODO: not likely to break but still bad solution.  refactor.
		n, err := conn.Read(userInput) // blocks the program but we need waitgroups if more than one connection
		if err != nil {
			panic(err) // replace with channel?
		}

		command, args := splitCommand(userInput, n)

		var response string // TODO: change to use the binary write?  refactor.

		switch strings.ToUpper(command) {
		case "EXIT", "QUIT":
			defer f.exitClient(id)
			response = "goodbye"
			sentinel = true
		case "DEL":
			response = f.deleteFile(args) // TODO: get filename from client
		case "GET":
			response = f.sendFileToClient(args) // TODO: get filename
		case "LS":
			response = f.listFiles()
		case "PUT":
			wg.Add(1)
			response = f.retrieveFileFromClient(args, id, wg)
			wg.Wait()
		default:
			fmt.Println("Invalid Command Received")
			response = "invalid command"
		}

		f.sendResponse(response, id) // bodge? lets the client know something happened
	}
}

func splitCommand(userInput []byte, n int) (string, string) {
	firstSpace := bytes.IndexByte(userInput, ' ')
	command := string(userInput[:firstSpace])
	args := string(userInput[firstSpace+1 : n])
	return command, args
}

// maybe don't need this but whatever
func (f *FtpServer) sendResponse(response string, id int) {
	f.clients[id].Write([]byte(response))
}

func (f *FtpServer) newClient(conn net.Conn) int {
	// TODO: could result in over-writing connections.  refactor.
	id := rand.Intn(10000)
	f.clients[id] = conn
	fmt.Printf("...connection accepted from %s\n", f.clients[id].RemoteAddr().String())
	return id
}

func (f *FtpServer) exitClient(id int) {
	fmt.Printf("client disconnected: %s\n", f.clients[id].RemoteAddr().String())
	f.clients[id].Close()
	delete(f.clients, id)
}

func (f *FtpServer) deleteFile(filename string) string {
	fmt.Println("delete", filename)
	return "deleted"
}

func (f *FtpServer) sendFileToClient(filename string) string {
	fmt.Println("get", filename)
	return "file sent"
}

func (f *FtpServer) listFiles() string {
	var response string
	cwd, err := os.Getwd()
	if err != nil {
		return "can't find out current directory"
	}
	response = fmt.Sprintf("%s\n", cwd)

	files, err := os.ReadDir(cwd)
	if err != nil {
		return "can't list files"
	}

	for _, f := range files {
		if len(f.Name()) > 0 {
			response += fmt.Sprintf("|- %s\n", f.Name())
		}
	}

	fmt.Println(response)

	return response
}

func (f *FtpServer) retrieveFileFromClient(fileName string, id int, wg *sync.WaitGroup) string {
	defer wg.Done()
	readBuffer := new(bytes.Buffer)

	var size int64
	binary.Read(f.clients[id], binary.LittleEndian, &size)
	fmt.Println("size", size)

	for {
		_, err := io.CopyN(readBuffer, f.clients[id], size)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "error receiving file"
		}
	}

	if readBuffer.Len() == 0 {
		return "failed to receive data"
	}

	fileName = filepath.Base(fileName)

	err := os.WriteFile(fileName, readBuffer.Bytes(), 0777) // is 777 a security risk?
	if err != nil {
		return "failed to write the file to disk"
	}

	return "file received"
}
