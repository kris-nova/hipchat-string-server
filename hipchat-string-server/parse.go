// parse.go
//
// This is the file that contains all logic for actually parsing a single string and returning a response struct
//
// Author: Kris Childress <kris@nivenly.com>

package hipchat_string_server

import (
	"fmt"
	"strings"
	"time"
	"errors"
	"crypto/md5"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

type Response struct {
	Emoticons []string
	Mentions  []string
	Links     []Link
}

type Link struct {
	Url   string
	Title string
}

type mentionBack struct {
	val   string
	found bool
}

type emoticonBack struct {
	val   string
	found bool
}

type linkBack struct {
	val   Link
	found bool
}

// ParseString() is the function that will actually take an input string and return a data structure of parsed values.
// This function takes advantage of golang's concurrency heavily.
//
// This function attempts to demonstrate that sometimes repetitive, concurrent work can be faster than clever, procedural work.

// In this case we blindly parse each word for a type (mention, emoticon, link) despite
// the foresight that if a word is evaluated to 1 type, it is impossible for it to be another type.
//
// By design this function will continue to evaluate types for words even if a word has already been matched. This function
// still attempts to be faster than a procedural evaluation of the same word list, and the cost of more (but concurrent)
// CPU cycles.
//
// This function hard codes a timeout to 3 seconds. To change this just update the timeout := 3 line
func ParseString(input string) (Response, error) {
	r := Response{}
	e := errors.New("")
	timeout := 3 //Seconds
	hlog := GetLogger()
	words := strings.Split(input, " ")
	wl := len(words)
	hlog.Debug.Printf("Parsing %d words", wl)
	total := 0

	// Channels
	mch := make(chan mentionBack)
	ech := make(chan emoticonBack)
	lch := make(chan linkBack)

	for _, word := range (words) {
		if word == " " {
			continue
		}

		//Mentions
		go parseMention(word, mch)
		total++

		//Emoticons
		go parseEmoticon(word, ech)
		total++

		//Links
		go parseLink(word, lch)
		total++
	}

	tch := getTimeout(timeout)
	hlog.Debug.Printf("Setting timeout: %d seconds", timeout)
	read := 0
	for read < total && read != -1 {
		select {
		case mb := <-mch:
			read++
			if mb.found {
				r.Mentions = append(r.Mentions, mb.val)
			}
			break
		case eb := <-ech:
			read++
			if eb.found {
				r.Emoticons = append(r.Emoticons, eb.val)
			}
			break
		case lb := <-lch:
			read++
			if lb.found {
				r.Links = append(r.Links, lb.val)
			}
			break
		case t := <-tch:
			if t {
				read = -1
				e = getErrorAndLog(fmt.Sprintf("Major Timeout. Waiting more than %d seconds for response in ParseString()", timeout))
				r = Response{} // We HAVE to ignore all parsed data as this request is completely invalid now
			}
			break
		}
	}

	hlog.Debug.Printf("Cleaning up channels..")
	close(mch)
	close(ech)
	close(lch)

	if read == -1 {
		return r, e
	}
	return r, nil
}

// Unique goroutine that will look for a mention in a string
func parseMention(str string, ch chan mentionBack) {
	brkChars := ",;-.!?/@<>[]{}_=+#$%^&*()'\"\\"
	found := false
	l := len(str)
	li := l - 1
	val := ""
	for i, rune := range (str) {
		char := string(rune)
		if char == "@" && i != li {
			found = true
			continue
		}
		if found {
			if !strings.Contains(brkChars, char) {
				val = val + char
			} else {
				break
			}
		}
	}
	// Check for no-ops
	if len(val) < 1 {
		found = false
	}
	// Send our results back
	ch <- mentionBack{val: val, found: found}
}

// Unique goroutine that will look for an emoticon in a string
func parseEmoticon(str string, ch chan emoticonBack) {
	brkChars := ",;-.!?/@<>[]{}_=+#$%^&*('\"\\"
	found := false
	val := ""
	for i, rune := range (str) {
		if i == 14 {
			val = ""
			found = false
			break
		}
		char := string(rune)
		if char == "(" {
			found = true
			continue
		}
		if found {
			if char == ")" {
				break
			} else if !strings.Contains(brkChars, char) {
				val = val + char
			} else {
				found = false
			}
		}
	}
	ch <- emoticonBack{val: val, found: found}
}

// Unique goroutine that will look for a URL in a string, and find it's title if possible
func parseLink(str string, ch chan linkBack) {
	url := ""
	title := ""
	found := false
	if strings.Contains(str, "http://") {
		//HTTP
		split := strings.SplitAfter(str, "http://")
		url = "http://" + split[1]
		found = true

	} else if strings.Contains(str, "https://") {
		//HTTPs
		split := strings.SplitAfter(str, "https://")
		url = "https://" + split[1]
		found = true
	}
	if found {
		title = getTitleFromUrl(url)
	}
	ch <- linkBack{val: Link{Title: title, Url: url}, found: found}
}

// We are suggesting that the entire string between <title> and </title> tags is the title of the page
func getTitleFromUrl(url string) string {
	//Send a very basic GET request
	resp, err := http.Get(url)
	hlog := GetLogger()
	if err != nil {
		hlog.Warning.Printf("Error while sending GET request to URL %s Message %s", url, err.Error())
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		hlog.Warning.Printf("Error while sending GET request to URL %s Message %s", url, err.Error())
		return ""
	}
	bodyStr := string(body)
	bspl := strings.Split(bodyStr, "<title>")
	if len(bspl) < 2 {
		hlog.Warning.Printf("Unable to find <title> tag in page content for url %s", url)
		return ""
	}
	bbspl := strings.Split(bspl[1], "</title>")
	if len(bbspl) < 2 {
		hlog.Warning.Printf("Unable to find </title> tag in page content for url %s", url)
		return ""
	}
	return bbspl[0] //The title of the page
}

func getTimeout(seconds int) chan bool {
	ch := make(chan bool)
	go func() {
		time.Sleep(time.Duration(seconds) * time.Second)
		ch <- true
	}()
	return ch
}

func getErrorAndLog(str string) error {
	e := errors.New(str)
	hlog := GetLogger()
	hlog.Error.Printf("%s\n", str)
	return e
}

func (l *Link) ToHash() string {
	data := []byte(fmt.Sprintf("%#v", l))
	return fmt.Sprintf("%x", md5.Sum(data))
}

func (r *Response) ToJson() string {
	hlog := GetLogger()
	bytes, err := json.Marshal(r)
	if err != nil {
		hlog.Warning.Printf("Unable to marshal JSON %s\n", err.Error())
		return ""
	}
	str := string(bytes)
	return str
}