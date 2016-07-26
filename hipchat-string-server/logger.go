// logger.go
//
// A very simple golang native logger wrapper
// Supports multiple log levels :
//
// Debug ---- Non threatening information. Useful for tracing and debugging the program.
// Info ----- Possible threatening information. Useful for identifying potential problems.
// Warning -- A known problem is occurring.
// Error ---- An unhandled case has occurred. This is probably a bug.
// Critical - A major error has occurred. The program is unusable and should be considered unsafe.
//
// Author: Kris Childress <kris@nivenly.com>

package hipchat_string_server

import (
	"io"
	"log"
	"os"
)

var hssl hss_logger
var hssl_init = false

type hss_logger struct {
	Debug    *log.Logger
	Info     *log.Logger
	Warning  *log.Logger
	Error    *log.Logger
	Critical *log.Logger
}

func NewLogger(dh, ih, wh, eh, ch io.Writer) hss_logger {
	l := hss_logger{}
	l.Debug = log.New(dh, "Debug: ", log.Ldate | log.Ltime | log.Lshortfile)
	l.Info = log.New(dh, "Info: ", log.Ldate | log.Ltime | log.Lshortfile)
	l.Warning = log.New(dh, "Warning: ", log.Ldate | log.Ltime | log.Lshortfile)
	l.Error = log.New(dh, "Error: ", log.Ldate | log.Ltime | log.Lshortfile | log.Lmicroseconds)
	l.Critical = log.New(dh, "Critical: ", log.Ldate | log.Ltime | log.Lshortfile | log.Lmicroseconds)
	return l
}

func SetLogger(hlog hss_logger) {
	hssl = hlog
	hssl_init = true
}

func GetLogger() hss_logger {
	if hssl_init == false {
		hssl = NewLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr, os.Stderr)
		hssl_init = true
	}
	return hssl
}