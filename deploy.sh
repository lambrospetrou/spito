#!/bin/bash

# check for updates in the packages and dependencies and download them
go get -u
# build the new Spitty
go build -o spitty-t

./stop.sh

mv spitty-t spitty

./start.sh

