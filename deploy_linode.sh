#!/bin/bash

# check for updates in the packages and dependencies and download them
go get -u
# build the new Spito
go build -o spito-t

# move the executable to the public directory
mv spito-t spito

# stop and restart supervisor
