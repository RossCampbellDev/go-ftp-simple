package main

import (
	"time"

	"github.com/RossCampbellDev/go-ftp-simple/internal/server"
)

func main() {
	// little bodge to do some testing with live server
	go func() {
		time.Sleep(2 * time.Second)
		server.SendFile(1000, ":10021")
	}()

	ftpServer := server.NewFtpServer()
	ftpServer.Listen(10021)
}
