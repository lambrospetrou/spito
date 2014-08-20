#!/bin/bash

# check for updates in the packages and dependencies and download them
go get -u
# build the new Spitty
go build -o spito-t

./stop.sh

mv spito-t spito

./start.sh

