package spit

import "github.com/lambrospetrou/goencoding/lpenc"

const SPIT_ID_CHARS string = "Ca1MoKtUR5A2BfeGm8LWwlFgHOx3hNk9ciTpuqZ7nrQjXyzJbvI64V0EYPsDSd"

var SpitIdEncoding = lpenc.NewEncoding(SPIT_ID_CHARS)

func ValidateId(id string) bool {
	_, err := SpitIdEncoding.Decode(id)
	return err == nil
}
