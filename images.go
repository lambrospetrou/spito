package main

import (
	"fmt"
	"image"
	"io"
	// the following are with underscores just to register their
	// type and be able to decode them with image.Decode
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

func DecodeImage(r io.Reader) (image.Image, string, error) {
	img, format, err := image.Decode(r)
	if err != nil {
		fmt.Errorf("Error while decoding image: %s\n", err.Error())
		return nil, "unknown", err
	}
	return img, format, nil
}
