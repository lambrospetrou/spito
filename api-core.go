package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/lambrospetrou/spito/spit"
)

const (
	MAX_FORM_SIZE int64 = 1 << 17 // 128KB

	CONTENT_TYPE_MULTIPART  string = "multipart/form-data"
	CONTENT_TYPE_URLENCODED string = "application/x-www-form-urlencoded"
)

var CORSAllowedOrigins map[string]bool = map[string]bool{
	"http://localhost:63342": true,
	"http://localhost:40090": true,
	"http://localhost:8080":  true,
	"http://localhost":       true,
	"http://spi.to":          true,
}

// the struct that is passed in the Add handlers
// when there are wrong arguments in the request.
// Errors: is a map containing errors with keys 'exp' & 'content'.
// 			Keys only exist if there is an error with them.
type ErrCoreAdd struct {
	Errors map[string]string
}

func (e *ErrCoreAdd) Error() string {
	return fmt.Sprintf("ErrCoreAdd: %v", e.Errors)
}

type ErrCoreAddDB struct {
	NewSpit *spit.Spit
	Message string
}

func (e *ErrCoreAddDB) Error() string {
	return fmt.Sprintf("ErrCoreAddDB: %v\nSpit: %v", e.Message, e.NewSpit)
}

// does the core execution of a new media spit addition.
// @param r: the request of the addition
// returns either the Spit successfully added and saved
// or an error ErrCoreAddDB if something went wrong during the creation or save of the spit
// or an error ErrCoreAdd when the validation of the request arguments failed
func CoreAddMultiSpit(r *http.Request) (*spit.Spit, error) {
	result := &ErrCoreAdd{}

	requestType := r.Header.Get("content-type")
	//fmt.Println(requestType)

	// try to parse the form with a maximum size
	if strings.HasPrefix(requestType, CONTENT_TYPE_MULTIPART) {
		err := r.ParseMultipartForm(MAX_FORM_SIZE)
		if err != nil {
			result.Errors = make(map[string]string)
			result.Errors["Generic"] = fmt.Sprintf("Too much data submitted (up to %d bytes) or invalid form data!", MAX_FORM_SIZE)
			return nil, result
		}
	} else if strings.HasPrefix(requestType, CONTENT_TYPE_URLENCODED) {
		err := r.ParseForm()
		if err != nil {
			result.Errors = make(map[string]string)
			result.Errors["Generic"] = "Invalid form data!"
			return nil, result
		}
	} else {
		result.Errors = make(map[string]string)
		result.Errors["Generic"] = "Invalid Content-Type specified!"
		return nil, result
	}

	// parse the request and try to create a spit
	nSpit, err := spit.NewFromRequest(r)
	if err != nil {
		if _, ok := err.(*spit.SpitError); ok {
			spitErr := err.(*spit.SpitError)
			result.Errors = spitErr.ErrorsMap
			return nil, result
		} else {
			log.Fatal(err.Error())
			return nil, err
		}
	}
	//log.Printf("%v\n", nSpit)

	// Save the spit
	if err = nSpit.Save(); err != nil {
		errDB := &ErrCoreAddDB{NewSpit: nSpit, Message: "Could not save spit in database!"}
		log.Printf("%s, %v", err.Error(), errDB)
		return nil, errDB
	}
	return nSpit, nil
}
