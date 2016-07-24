# Makefile - hipchat-string-server

default:    clean build

build:
	@sh -c "'$(CURDIR)/scripts/build.sh'"

clean:
	@sh -c "'$(CURDIR)/scripts/clean.sh'"