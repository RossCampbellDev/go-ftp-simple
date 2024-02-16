package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
)

type myConn struct {
	net.Conn // embed in the struct so we inherit from net.Conn
}

var (
	server = ":10021"
)

func main() {
	var wg sync.WaitGroup
	// connect to server
	wg.Add(1)
	myConn := myConn{Conn: connect(server, &wg)}
	wg.Wait()
	defer myConn.Close()

	for {
		cmd := getUserInput()

		wg.Add(1)
		go myConn.parseCommand(cmd, &wg)
		wg.Wait()

		// TODO: come up with better way of receiving response.
		// currently, it seems to deadlock if there was a problem with GET (only thing tested)
		response := make([]byte, 512) // TODO: needs replacing for CopyN etc.  refactor.
		if _, err := myConn.Read(response); err != nil {
			if err == io.EOF {
				fmt.Println("<< connection closed >>")
				os.Exit(0)
			} else {
				panic(err)
			}
		}

		if strings.Contains(string(response), "goodbye") {
			fmt.Println("<< connection closed >>")
			os.Exit(0)
		}

		fmt.Println("------------------------------")
	}
}

func connect(ipAddr string, wg *sync.WaitGroup) net.Conn {
	defer wg.Done()
	conn, err := net.Dial("tcp", ipAddr)
	if err != nil {
		panic(err)
	}
	fmt.Println("connected to server...")
	return conn
}

func getUserInput() string {
	var userInput string
	for len(userInput) == 0 {
		fmt.Printf(" > ")
		fmt.Scanln(&userInput)
	}
	return strings.ToUpper(userInput)
}

func (conn myConn) parseCommand(command string, wg *sync.WaitGroup) {
	var fileName string

	switch command {
	case "DEL", "GET", "PUT":
		if !checkCommand(command) {
			fmt.Println("Bad Command!  try <CMD> <filename.ext>")
			wg.Done()
			return
		} else {
			fileName = strings.Split(command, " ")[0]
		}
	}

	switch command {
	case "GET":
		if checkFileExists(fileName) {
			conn.sendAndReceive(fileName, wg)
		} else {
			fmt.Println("File Not Found!")
			wg.Done()
			return
		}
	case "PUT":
		if checkFileExists(fileName) {
			conn.sendFile(fileName, wg)
		} else {
			fmt.Println("File Not Found!")
			wg.Done()
			return
		}
	default:
		conn.sendCommand(command, wg)
	}
}

func (conn myConn) sendCommand(command string, wg *sync.WaitGroup) error {
	defer wg.Done()
	commandBytes := []byte(command)

	_, err := io.Copy(conn, bytes.NewReader(commandBytes))
	if err != nil {
		return err
	}

	return nil
}

func (conn myConn) sendFile(fileName string, wg *sync.WaitGroup) error {
	defer wg.Done()
	size := getFileSize(fileName)
	fileBytes := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, fileBytes) // reads len(fileBytes) bytes from <io.Reader> into file.
	if err != nil {
		return err
	}

	// tell the server side the size of the file before streaming over
	binary.Write(conn, binary.LittleEndian, int64(size))

	n, err := io.CopyN(conn, bytes.NewReader(fileBytes), int64(size)) // copy TO the connection.  convert file to fit the io.Reader interface
	if err != nil {
		return err
	}

	fmt.Printf("written %d bytes over\n", n)
	return nil
}

func (conn myConn) sendAndReceive(fileName string, wg *sync.WaitGroup) error {
	fmt.Println("send file", fileName)
	return nil
}

func checkCommand(command string) bool {
	regexPattern := `/^\w*\s\S*\.[a-zA-Z]{3,4}$/gm` // two strings separated by a space, with a file extension
	regexer := regexp.MustCompile(regexPattern)
	return regexer.MatchString(command)
}

func checkFileExists(fileName string) bool {
	_, err := os.Stat(fileName)
	return os.IsExist(err)
}

func getFileSize(fileName string) int64 {
	f, err := os.Stat(fileName)
	if err != nil {
		fmt.Println("how did we get here in getting the size of a file?", err)
		return 0
	}
	return f.Size()
}