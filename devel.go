// devel.go
//
// A simple program for developing against the library
// Takes the first argument as an input string and runs it against ParseString()
// Will output the returned response as JSON
//
// Author: Kris Childress <kris@nivenly.com>

package main

import (
	"os"
	"fmt"
	"github.com/kris-nova/hipchat-string-server/hipchat-string-server"
	"encoding/json"
)

func main() {
	if len(os.Args) == 1 {
		usage()
	}
	input := os.Args[1]
	resp, err := hipchat_string_server.ParseString(input)
	if err != nil {
		fmt.Printf("Major error, unable to parse. See logs.\n")
		os.Exit(1)
	}
	//fmt.Printf("%v\n", resp)
	bytes, err := json.Marshal(resp)
	if err != nil {
		fmt.Printf("Invalid JSON!\n")
	}
	jstring := string(bytes)
	fmt.Printf("%s\n", jstring)
	os.Exit(0)
}

func usage() {
	fmt.Printf("Usage: devel.go <input_string>\n")
	os.Exit(1)
}
