# hipchat-string-server
A lightweight webserver that parses a chat string according to a few rules.

# Building

Note: This assumes go is already installed on the system.

Note: This assumes that the repository is in a valid GOPATH

To get the repository

    go get github.com/kris-nova/hipchat-string-server

The repository comes with a very simple Makefile. To use the makefile navigate to the repositories top level directory and run make.

    cd hipchat-string-server
    make

Make will create a bin directory, and drop the executables and binaries in there.

    cd bin

 - hssd
   - The server daemon
 - hss_devel
    - A simple developers tool for running arbitrary strings against ParseString()
 - hss_test
    - A test program that will test the library and the daemon (if running) for functionality.
 - hss_time_test
    - A wrapper for hss_test that will time the test program


# Running

To run the server as a daemon use the following commands

    cd hipchat-string-server
    ./bin/hssd > server.log 2>&1 &

Make sure to note the PID of the daemon

Now that the server is running you can execute the test program

    cd hipchat-string-server
    ./bin/hss_time_test

You can now kill the pid that was running the daemon

    kill <pid>

# Notes

The server and test program default to port 1313
To change the port add the --port <num> flag when running the binaries.

Example to run on port 8000 :

    ./bin/hssd --port 8000 > server.log 2>&1 &



