package main

import (
	"net/http"
)

////////////////// HELPERS /////////////////////////
func isUrl(u string) bool {
	//resp, err := http.Get(u)
	_, err := http.Head(u)
	return err == nil
}

func AbsoluteSpittyURL(id string) string {
	return "http://lp.gs/" + id
}
