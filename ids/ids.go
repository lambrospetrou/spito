package ids

import (
	"log"

	"github.com/lambrospetrou/goencoding/lpenc"
)

const SPIT_ID_CHARS string = "Ca1MoKtUR5A2BfeGm8LWwlFgHOx3hNk9ciTpuqZ7nrQjXyzJbvI64V0EYPsDSd"

var _SpitIdEncoding = lpenc.NewEncoding(SPIT_ID_CHARS)

var _SpitIdEncodings []*lpenc.Encoding

func InitWith(chars ...string) {
	_SpitIdEncodings = make([]*lpenc.Encoding, len(chars))
	for i := 0; i < len(chars); i++ {
		_SpitIdEncodings[i] = lpenc.NewEncoding(chars[i])
	}
}

func Encode(n uint64, encodingIdx int) string {
	if encodingIdx >= len(_SpitIdEncodings) || encodingIdx < 0 {
		log.Println("Encoding idx given is invalid: ", encodingIdx)
		return ""
	}
	return _SpitIdEncodings[encodingIdx].Encode(n)
}

// ValidateId validates that the given id has the right format.
// It only checks that the characters used belong to our key space domain.
func ValidateId(id string) bool {
	_, err := _SpitIdEncoding.Decode(id)
	return err == nil
}
