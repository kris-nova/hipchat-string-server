// parse.go
//
// This is the file that contains all logic for actually parsing a single string and returning a response struct
//
// The algorithm here makes a few assumptions, they are :
// 	1. That all input strings contain 1 or more words.
// 	2. That all mentions, emoticons, or links are contained within a single word.
//
// The basic logic of the algorithm is :
// 	1. Break down the input into words. Where a word is a string of text delimited by one or more spaces.
//	2. Create channels for each goroutine to report back on
// 	3. Iterate over the words, checking each word for a mention, and emoticon, or a link in a unique goroutine.
//	4. If a mention, emoticon, or link is found a TOTAL parse needs to be handled, and then the goroutine
//		can report back over it's channel.
//      5. As soon as all goroutines have reported back the parse is over and can finally return
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
//
// Returns (Response{}, err)
//
func ParseString(input string) (Response, error) {
	r := Response{} //New response for every call
	e := errors.New("") //New possible error every call
	timeout := 3 //Define timeout in seconds - 3 Should be plenty
	hlog := GetLogger()
	words := strings.Split(input, " ") //Split on a single space.. empty words will be ignored
	wl := len(words)
	hlog.Debug.Printf("Parsing %d words", wl)
	total := 0 //The total number of parsed words
	// Channels for each type
	mch := make(chan mentionBack)
	ech := make(chan emoticonBack)
	lch := make(chan linkBack)

	// Goroutine for each word with it's own channel
	for _, word := range (words) {
		// Skip non-words
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
	// Read responses back from goroutines until we have read them all or we timeout
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
				read = -1 // Break out of the loop, we timed out
				e = getErrorAndLog(fmt.Sprintf("Major Timeout. Waiting more than %d seconds for response in ParseString()", timeout))
				r = Response{} // We HAVE to ignore all parsed data as this request is completely invalid now
			}
			break
		}
	}

	//Cleanup channels
	hlog.Debug.Printf("Cleaning up channels..")
	close(mch)
	close(ech)
	close(lch)

	// Return values
	if read == -1 {
		return r, e
	}
	return r, nil
}

// Unique goroutine that will look for a mention in a string
//
// Notes:
// It's a mention if
// 1. Contains @
// 2. Ends with a alphanumeric character
// 2a. Ends with non-alphanumeric a character other than space. This needs to be ignored from the mention in general
func parseMention(str string, ch chan mentionBack) {
	brkChars := ",;-.!?/@<>[]{}_=+#$%^&*()'\"\\"
	found := false
	l := len(str) //Length of the string starting at 1
	li := l - 1 //Length of the string starting at 0 to match our i index
	val := ""
	for i, rune := range (str) {
		char := string(rune)
		if char == "@" && i != li {
			// We have an @ and this is not the last char
			found = true
			continue
		}
		if found {
			//Check for a breaking character
			if !strings.Contains(brkChars, char) {
				val = val + char //Append the current val
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
//
// Notes:
// You only need to consider 'custom' emoticons which are alphanumeric strings, no longer than 15 characters, contained in parenthesis.
// You can assume that anything matching this format is an emoticon.
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
				val = val + char //Append the current val
			} else {
				found = false // This never ended
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
	//Find our opening <title>
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

// Handy for comparing link structs together
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