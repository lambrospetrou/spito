#!/bin/bash

# check for updates in the packages and dependencies and download them
#go get -u
#go get
# build the new Spito
#go build -o spito-t

SPITO_DIR=/home/lambros/public/spito/public

# move the executable to the public directory while restarting supervisor
mv spito $SPITO_DIR/spito
rm -rf $SPITO_DIR/templates
cp -rf templates $SPITO_DIR/
rm -rf $SPITO_DIR/static
cp -rf static $SPITO_DIR/

sudo supervisorctl restart spitoapi

