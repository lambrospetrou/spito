#!/bin/bash

BUILD_DIR="build-release"

# build the executable
go build

# create the release folder and put the required files inside
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

cp -r spito static templates "$BUILD_DIR/"

# zip the bundle into a single file
tar -cvzf spitoapi.tgz "$BUILD_DIR"
mv spitoapi.tgz "$BUILD_DIR/"

