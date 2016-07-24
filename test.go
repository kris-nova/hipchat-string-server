// test.go
//
// Will run a number of test cases against the provided hipchat-string-server library, as well as running
// the same test cases against a listening HTTP socket. Both in unique goroutines.
// Will assert expected responses from each method of testing.
//
// Running the program will exit with the following exit codes: (See STDOUT for more information.)
//	0    All tests passed.
// 	1+   One or more tests failed. The exit code will correspond to the number of failures.
//	-1   This is a major failure in the program.
//
//
// Author: Kris Childress <kris@nivenly.com>

package main

import (
	"github.com/kris-nova/hipchat-string-server/hipchat-string-server"
	"fmt"
	"os"
	"net/http"
	"strings"
	"strconv"
	"io/ioutil"
	"encoding/json"
	"net/url"
)

var port = 1313

type testResults struct {
	serverDetected bool
	totalTests     int
	tested         chan int
	failures       []*testCase
	successes      []*testCase
}

type testCase struct {
	name             string
	success          bool
	input            string
	expectedResponse *hipchat_string_server.Response
	actualResponse   *hipchat_string_server.Response
}


// This main() function just runs all of the test cases, and then will output the results
func main() {
	handleArgs()
	expr := hipchat_string_server.Response{} //Init empty test case
	tr := testResults{} //Init new testResults struct
	tr.tested = make(chan int)
	// Verify the server is running
	_, err := http.Get(fmt.Sprintf("http://localhost:%d/parse", port))
	if err != nil {
		fmt.Printf("Unable to reach server. Will not run server tests!\n")
		fmt.Printf("Server Error: %s\n", err.Error())
	} else {
		tr.serverDetected = true
	}
	fmt.Printf("\n\n [LOG OUTPUT]\n\n")


	//--[TEST]------------------------------------------------------------------------------------------------------
	expr = hipchat_string_server.Response{
		Mentions: []string{"chris"},
	}
	tr.test("Provided #1 (Mentions)", "@chris you around?", expr)



	//--[TEST]------------------------------------------------------------------------------------------------------
	expr = hipchat_string_server.Response{
		Emoticons: []string{"megusta", "coffee"},
	}
	tr.test("Provided #2 (Emoticons)", "Good morning! (megusta) (coffee)", expr)



	//--[TEST]------------------------------------------------------------------------------------------------------
	expr = hipchat_string_server.Response{
		Links: []hipchat_string_server.Link{hipchat_string_server.Link{
			Url: "http://www.nbcolympics.com",
			Title: "2016 Rio Olympic Games | NBC Olympics"}},
	}
	tr.test("Provided #3 (URLs)", "Olympics are starting soon; http://www.nbcolympics.com", expr)



	//--[TEST]------------------------------------------------------------------------------------------------------
	expr = hipchat_string_server.Response{
		Mentions: []string{"bob", "john"},
		Emoticons: []string{"success"},
		Links: []hipchat_string_server.Link{hipchat_string_server.Link{
			Url: "https://twitter.com/jdorfman/status/430511497475670016",
			Title: "Justin Dorfman on Twitter: &quot;nice @littlebigdetail from @HipChat (shows hex colors when pasted in chat). http://t.co/7cI6Gjy5pq&quot;"}},
	}
	tr.test("Provided #4 (Full Test)", "@bob @john (success) such a cool feature; https://twitter.com/jdorfman/status/430511497475670016", expr)



	//--[TEST]------------------------------------------------------------------------------------------------------
	expr = hipchat_string_server.Response{
		Mentions: []string{"kris", "kjersti", "charlie", "niche", "hank"},
	}
	tr.test("Mention Edge Cases", "@kris asdf@kjersti, -@charlie! @@ @@@@ @@ @@niche asf8!@#!!@$asdf9124 sdf@hank! dfafd          ", expr)



	//--[TEST]------------------------------------------------------------------------------------------------------
	expr = hipchat_string_server.Response{
		Emoticons: []string{"kris", "kjersti"},
	}
	tr.test("Emoticons Edge Cases", "a asf f!(kris), (kjersti)    (123456789101112131415)", expr)



	//--[TEST]------------------------------------------------------------------------------------------------------
	expr = hipchat_string_server.Response{
		Links: []hipchat_string_server.Link{hipchat_string_server.Link{
			Url: "http://google.com",
			Title: "Google"}},
	}
	tr.test("URLs Edge Cases", "a!@#!(*http://google.com", expr)

	// Lets see what happened
	tr.out()
}

func (tr *testResults) test(name, input string, expr hipchat_string_server.Response) {
	hlog := hipchat_string_server.GetLogger()
	hlog.Debug.Printf("Running test: %s\n", name)
	go tr.testLibrary("[LIBRARY] " + name, input, expr)
	tr.totalTests++
	if tr.serverDetected {
		go tr.testServer("[SERVER] " + name, input, expr)
		tr.totalTests++
	}
}

func (tr *testResults) testServer(name, input string, expr hipchat_string_server.Response) {
	// Init the testCase
	hlog := hipchat_string_server.GetLogger()
	testCase := testCase{
		success: false,
		input: input,
		name: name,
		expectedResponse: &expr,
		actualResponse: &hipchat_string_server.Response{},
	}
	url := fmt.Sprintf("http://localhost:%d/parse?input=", port) + url.QueryEscape(input)
	hlog.Debug.Printf("Sending GET : %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		tr.failures = append(tr.failures, &testCase)
		hlog.Warning.Printf("GET failure: %s\n", err.Error())
		return
	}
	defer resp.Body.Close()
	body, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		tr.failures = append(tr.failures, &testCase)
		hlog.Warning.Printf("GET body parse failure: %s\n", err2.Error())
		return
	}
	actr := hipchat_string_server.Response{}
	err3 := json.Unmarshal(body, &actr)
	if err3 != nil {
		tr.failures = append(tr.failures, &testCase)
		hlog.Warning.Printf("JSON parse failure: %s\n", err3.Error())
		return
	}
	testCase.actualResponse = &actr

	if compare(testCase.expectedResponse, testCase.actualResponse) {
		//Success!
		testCase.success = true
		tr.successes = append(tr.successes, &testCase)
	} else {
		//Failure!
		tr.failures = append(tr.failures, &testCase)
	}
	tr.tested <- 1
}

// A simple test function to compare an expected response struct to an actual response struct
func (tr *testResults) testLibrary(name, input string, expr hipchat_string_server.Response) {
	// Init the testCase
	testCase := testCase{
		success: false,
		input: input,
		name: name,
		expectedResponse: &expr,
	}
	actr, err := hipchat_string_server.ParseString(input) // Get the actual response
	if err != nil {
		// This is beyond a major problem
		fmt.Printf("Unable to call ParseString()! Major Error!\n")
		fmt.Println(err.Error())
		os.Exit(-1)
	}
	testCase.actualResponse = &actr // Add the response to the testCase

	if compare(testCase.expectedResponse, testCase.actualResponse) {
		//Success!
		testCase.success = true
		tr.successes = append(tr.successes, &testCase)
	} else {
		//Failure!
		tr.failures = append(tr.failures, &testCase)
	}
	tr.tested <- 1
}

// Will parse the test results and output the results.
// Will also handle exiting the test program with a meaningful exit code.
func (tr *testResults) out() {
	// Hang until all tests are done
	i := 0
	for i < tr.totalTests {
		i = i + <-tr.tested
	}

	fmt.Printf("\n\n [TEST RESULTS]\n\n")
	for _, f := range (tr.failures) {
		fmt.Printf("Failure: %s\n   Input:    %s\n   Expected: %v\n   Actual:   %v\n", f.name, f.input,
			*f.expectedResponse, *f.actualResponse)
	}
	if len(tr.failures) > 0 {
		fmt.Printf("\n\n")
	}
	for _, s := range (tr.successes) {
		fmt.Printf("Success: %s\n", s.name)
	}

	if len(tr.failures) == 0 {
		fmt.Printf("\n\nSuccessful test. No failures. \n%d tests passed.\n\n", len(tr.successes))
		os.Exit(0) //Lots of double negatives here. But, if we DON'T have a failure, then exit nice
	}
	os.Exit(len(tr.failures)) //We had a test fail, so exit with the number of failures
}

// This in an intersting compare function. This will assert that the two structs contain the same data
// without caring about their order.
func compare(a, b *hipchat_string_server.Response) bool {
	//Mentions
	f := false
	for _, m := range (a.Mentions) {
		f = false
		for _, mm := range (b.Mentions) {
			if m == mm {
				f = true
				break
			}
		}
		if !f {
			return f
		}
	}

	//Emoticons
	for _, e := range (a.Emoticons) {
		f = false
		for _, ee := range (b.Emoticons) {
			if e == ee {
				f = true
				break
			}
		}
		if !f {
			return f
		}
	}

	//Links
	for _, l := range (a.Links) {
		f = false
		lh := l.ToHash()
		for _, ll := range (b.Links) {
			llh := ll.ToHash()
			if lh == llh {
				f = true
				break
			}
		}
		if !f {
			return f
		}
	}
	return f
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
