#!/bin/bash

# check for updates in the packages and dependencies and download them
#go get -u
#go get
# build the new Spito
#go build -o spito-t

SPITO_DIR=/home/lambros/public/spito/public

# move the executable to the public directory while restarting supervisor
mv spito $SPITO_DIR/spito
rm -rf $SPITO_DIR/spitoweb
cp -rf spitoweb $SPITO_DIR/

sudo supervisorctl restart spitoapi

