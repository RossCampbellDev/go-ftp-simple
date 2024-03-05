package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FtpServer struct {
	Port       int
	PortString string
	Clients    map[int]net.Conn
}

func NewFtpServer() *FtpServer {
	return &FtpServer{
		Clients: make(map[int]net.Conn),
	}
}

func (f *FtpServer) Listen(port int) {
	f.Port = port
	f.PortString = fmt.Sprintf(":%d", port)

	ln, err := net.Listen("tcp", f.PortString)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("listening on %d...\n", port)

	var wg sync.WaitGroup
	wg.Add(1)
	go f.quitter(&wg) // TODO: not sure how this works in conjunction with the below loop

	for {
		wg.Add(1)
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		id := f.newClient(conn)
		go f.parseCommand(&wg, id)
	}
	wg.Wait()
}

func (f *FtpServer) parseCommand(wg *sync.WaitGroup, id int) {
	defer wg.Done()
	sentinel := false

	for !sentinel {
		userInput := make([]byte, 512)          // TODO: not likely to break but still bad solution.  refactor.  use bufio.Scanner?
		n, err := f.Clients[id].Read(userInput) // blocks the program but we need waitgroups if more than one connection
		if err != nil {
			log.Fatal(err) // replace with channel?
		}

		command, args := splitCommand(userInput, n)

		var response string // TODO: change to use the binary write?  refactor.

		switch strings.ToUpper(command) {
		case "EXIT", "QUIT":
			defer f.exitClient(id)
			response = "goodbye"
			sentinel = true
		case "DEL": // TODO: hanging after successfully deleting the file
			response = f.deleteFile(args)
		case "GET":
			response = f.sendFileToClient(args, id)
		case "LS":
			response = f.listFiles()
		case "PUT":
			wg.Add(1)
			response = f.retrieveFileFromClient(args, id, wg)
		default:
			fmt.Printf("Invalid Command Received: '%s'\n", command)
			response = fmt.Sprintf("invalid command: '%s'\n", command)
		}

		f.sendResponse(response, id) // bodge? lets the client know something happened
	}
}

func splitCommand(userInput []byte, n int) (string, string) {
	var (
		firstSpace = bytes.IndexByte(userInput, ' ')
		command    = ""
		args       = ""
	)

	if firstSpace > -1 {
		command = string(userInput[:firstSpace])
		args = string(userInput[firstSpace+1 : n])
	} else {
		command = string(userInput[0:n])
	}
	return command, args
}

// maybe don't need this but whatever
func (f *FtpServer) sendResponse(response string, id int) {
	f.Clients[id].Write([]byte(response))
}

func (f *FtpServer) deleteFile(fileName string) string {
	err := os.Remove(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return "File doesn't exist anyway!"
		}
		return fmt.Sprintf("There was a problem deleting %s", fileName)
	}

	return fmt.Sprintf("%s Deleted", fileName)
}

func (f *FtpServer) sendFileToClient(fileName string, id int) string {
	fmt.Println("get", fileName)
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

	return response
}

func (f *FtpServer) retrieveFileFromClient(fileName string, id int, wg *sync.WaitGroup) string {
	defer wg.Done()
	var (
		readBuffer = new(bytes.Buffer)
		size       int64
	)
	binary.Read(f.Clients[id], binary.LittleEndian, &size)

	for {
		n, err := io.CopyN(readBuffer, f.Clients[id], size)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "error receiving file"
		}
		if n >= size {
			break
		}
	}

	if readBuffer.Len() == 0 {
		return "failed to receive data"
	}

	fileName = filepath.Base(fileName)

	err := os.WriteFile(fileName, readBuffer.Bytes(), 0777) // is 777 a security risk?
	if err != nil {
		log.Fatal(err)
		return "failed to write the file to disk"
	}

	return "file uploaded successfully!"
}

func (f *FtpServer) quitter(wg *sync.WaitGroup) {
	defer wg.Done()
	var input string
	for {
		fmt.Scanln(&input)
		if input == "quit" {
			for id := range f.Clients {
				f.sendResponse("goodbye", id) // TODO: not being listened for at the other end.  client only listens when it's sent a command
			}
			os.Exit(0)
		}
	}
}

func (f *FtpServer) newClient(conn net.Conn) int {
	// TODO: could result in over-writing connections.  refactor.
	id := rand.Intn(10000)
	f.Clients[id] = conn
	fmt.Printf("...connection accepted from %s\n", f.Clients[id].RemoteAddr().String())
	return id
}

func (f *FtpServer) exitClient(id int) {
	fmt.Printf("client disconnected: %s\n", f.Clients[id].RemoteAddr().String())
	f.Clients[id].Close()
	delete(f.Clients, id)
}
