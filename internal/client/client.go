package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
)

type myConn struct {
	net.Conn // embed in the struct so we inherit from net.Conn
}

func main() {
	var wg sync.WaitGroup
	// connect to server
	wg.Add(1)
	myConn := myConn{Conn: connect(":10021", &wg)}
	wg.Wait()
	defer myConn.Close()

	for {
		cmd := getUserInput()

		wg.Add(1)
		go myConn.sendCommand(cmd, &wg)
		wg.Wait()

		response := make([]byte, 1) // TODO: needs replacing for CopyN etc.  refactor.
		if _, err := myConn.Read(response); err != nil {
			if err == io.EOF {
				fmt.Println("<< connection closed >>")
				os.Exit(0)
			} else {
				panic(err)
			}
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
	// fmt.Printf(" > ")
	// fmt.Scanln(&userInput)

	for len(userInput) == 0 {
		fmt.Printf(" > ")
		fmt.Scanln(&userInput)
	}

	return strings.ToUpper(userInput)
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

func (conn myConn) sendFile(size int, ipAddr string) error {
	file := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, file) // reads len(file) bytes from <io.Reader> into file.
	if err != nil {
		return err
	}

	// add this.  tell the server side the size of the file before streaming over
	binary.Write(conn, binary.LittleEndian, int64(size))

	n, err := io.CopyN(conn, bytes.NewReader(file), int64(size)) // copy TO the connection.  convert file to fit the io.Reader interface
	if err != nil {
		return err
	}

	fmt.Printf("written %d bytes over\n", n)
	return nil
}
