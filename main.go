// main.go
//
// This is the command line wrapper for the server.
// Default port : 1313
//
// Usage : hipchat_string_server --port <number>
//
// Author: Kris Childress <kris@nivenly.com>

package main

import (
	"github.com/kris-nova/hipchat-string-server/hipchat-string-server"
	"os"
	"strings"
	"strconv"
)

var port = 1313

func main() {
	handleArgs()
	hipchat_string_server.Listen(port)
}

func handleArgs() {
	if len(os.Args) == 1 {
		return
	}
	for i, arg := range (os.Args) {
		if strings.Contains(arg, "--port") && len(os.Args) > i {
			pstr := os.Args[i + 1]
			port, _ = strconv.Atoi(pstr)
		}
	}
}