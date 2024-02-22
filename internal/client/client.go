package main

import (
	"bufio"
	"bytes"
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

	wg.Add(1)
	myConn := myConn{Conn: connect(server, &wg)}
	wg.Wait()
	defer myConn.Close()

	for {
		cmd, args := getUserInput()

		wg.Add(1)
		go myConn.runCommand(cmd, args, &wg)
		wg.Wait()

		// TODO: come up with better way of receiving response.
		response := make([]byte, 512)
		n, err := myConn.Read(response)
		if err != nil {
			if err == io.EOF {
				fmt.Println("<< connection closed >>")
				os.Exit(0)
			} else {
				panic(err)
			}
		}

		wg.Add(1)
		go parseResponse(response, n, &wg)
		wg.Wait()
		fmt.Println("------------------------------")
	}
}

func connect(ipAddr string, wg *sync.WaitGroup) net.Conn {
	defer wg.Done()
	conn, err := net.Dial("tcp", ipAddr)
	if err != nil {
		fmt.Println("connection refused")
		os.Exit(0)
	}
	fmt.Println("connected to server...")
	return conn
}

func getUserInput() (string, string) {
	var userInput, command, args string

	for len(userInput) == 0 {
		fmt.Printf(" > ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		userInput = scanner.Text()

		command, args = splitCommand(userInput)
		switch strings.ToUpper(command) {
		case "DEL", "GET", "PUT":
			if !checkCommandFormat(command, args) {
				fmt.Println("Bad Command!  try <CMD> <filename.ext>")
				userInput = ""
			}
		}
	}

	return command, args
}

// TODO: this could be refactored to use decorators or whatever?  instead of coding each type of cmd here
func (conn myConn) runCommand(command string, args string, wg *sync.WaitGroup) {
	switch strings.ToUpper(command) {
	case "GET":
		if checkFileExists(args) {
			conn.sendAndReceive(args, wg)
		} else {
			fmt.Println("File Not Found!")
			wg.Done()
			return
		}
	case "PUT":
		if checkFileExists(args) {
			wg.Add(1)
			conn.sendCommand(command+" "+args, wg)
			if err := conn.sendFile(args, wg); err != nil {
				fmt.Println("error sending file", err)
			}
		} else {
			fmt.Println("File Not Found!")
			wg.Done()
			return
		}
	default:
		conn.sendCommand(command, wg)
	}
}

func parseResponse(userInput []byte, n int, wg *sync.WaitGroup) {
	defer wg.Done()
	if strings.Contains(string(userInput), "goodbye") {
		fmt.Println("<< connection closed >>")
		os.Exit(0)
	}

	response := string(userInput[:n])

	fmt.Printf(" < %s\n", response)
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

	file, err := os.Open(fileName)
	if err != nil {
		return err
	}

	binary.Write(conn, binary.LittleEndian, int64(size))

	n, err := io.CopyN(conn, file, int64(size)) // TODO: change to send the file in chunks rather than 1 go?  refactor.
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

func checkCommandFormat(command string, args string) bool {
	regexPattern := `^\w*\s\S*\.[a-zA-Z]{3,4}$` // two strings separated by a space, with a file extension
	regexer := regexp.MustCompile(regexPattern)
	return regexer.MatchString(command + " " + args)
}

func splitCommand(command string) (string, string) {
	firstSpace := strings.Index(command, " ")
	args := ""
	if firstSpace > -1 {
		if firstSpace > 0 {
			args = command[firstSpace+1:]
		}
		command = command[:firstSpace]
	}
	return command, args
}

func checkFileExists(fileName string) bool {
	_, err := os.Stat(fileName)
	return !os.IsNotExist(err)
}

func getFileSize(fileName string) int64 {
	f, err := os.Stat(fileName)
	if err != nil {
		fmt.Println("how did we get here in getting the size of a file?", err)
		return 0
	}
	return f.Size()
}
