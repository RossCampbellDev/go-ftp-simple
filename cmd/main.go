package main

import (
	"github.com/RossCampbellDev/go-ftp-simple/internal/server"
)

func main() {
	ftpServer := server.NewFtpServer()
	ftpServer.Listen(10021)
}
