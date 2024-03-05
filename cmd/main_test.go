package main

import (
	"testing"

	"github.com/RossCampbellDev/go-ftp-simple/internal/server"
)

func TestMain(t *testing.T) {
	ftpServer := server.NewFtpServer()
	ftpServer.Listen(10021)
}
