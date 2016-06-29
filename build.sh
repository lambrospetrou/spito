#go get

DIR_BUILD="build"

rm -rf $DIR_BUILD
mkdir -p $DIR_BUILD
go build -o "$DIR_BUILD/bin/spito"

# Copy all required files to the bin folder
cp Procfile "$DIR_BUILD"

# Create the bundle of the API
pushd "$DIR_BUILD"
zip -r "spitoapi-bundle.zip" .
popd