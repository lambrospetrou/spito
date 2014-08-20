package main

import (
	"math/rand"
	"net/http"
	"time"
	"unicode/utf8"
)

////////////////// HELPERS /////////////////////////
func isUrl(u string) bool {
	//resp, err := http.Get(u)
	_, err := http.Head(u)
	return err == nil
}

func AbsoluteSpittyURL(id string) string {
	return "http://spi.to/" + id
}

func ShuffleString(s string) string {
	rand.Seed(time.Now().UnixNano())
	rs := make([]rune, utf8.RuneCountInString(s))
	newRSlen := len(rs)
	for _, c := range s {
		pos := rand.Intn(newRSlen)
		for ; rs[pos] != 0; pos = (pos + 1) % newRSlen {
		} // end valid position
		rs[pos] = c
	} // all characters processed
	return string(rs)
}
