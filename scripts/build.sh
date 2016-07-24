#!/bin/bash

# Well create binaries and executables
#
# hssd          HipChat String Server Daemon
# hss_devel     Handy tool for developing
# hss_test      Will run the test program
# hss_time_test Will wrap the test program and time the execution
#

mkdir -p bin

# Daemon
go build main.go
mv main bin/hssd

# Devel tool
go build devel.go
mv devel bin/hss_devel

# Test program
go build test.go
mv test bin/hss_test
touch bin/hss_time_test
echo "#!/bin/bash" >> bin/hss_time_test
echo "time $(pwd)/bin/hss_test" >> bin/hss_time_test
chmod +x bin/hss_time_test

