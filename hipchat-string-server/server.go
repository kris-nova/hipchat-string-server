// server.go
//
// The worlds simplest concurrent webserver.
// This basically just wraps the golang http library and handles JSON marshaling.
//
// Author: Kris Childress <kris@nivenly.com>

package hipchat_string_server

import (
	"io"
	"net/http"
	"fmt"
)

func parse(w http.ResponseWriter, r *http.Request) {
	out := ""
	input := r.URL.Query().Get("input")
	if input == "" {
		out = fmt.Sprintf("{\"failure\":\"%s\"}", "Missing input parameter")
	} else {
		resp, err := ParseString(input)
		out = resp.ToJson()
		if err != nil {
			out = fmt.Sprintf("{\"failure\":\"%s\"}", err.Error())
		}
	}
	io.WriteString(w, out)
}

func Listen(port int) {
	hlog := GetLogger()
	hlog.Debug.Printf("Starting HipChat String Server.. Listening on localhost:%d", port)
	http.HandleFunc("/parse", parse)
	hlog.Debug.Printf("Example GET request: http://localhost:%d/parse?input=@kris", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil) //Concurrent requests handled! Thanks go
}