package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// the struct that is passed in the Add handlers
// when there are wrong arguments in the request.
// InputExp: is the value of Expiration in request
// InputContent: is the value of the content in request
// Errors: is a map containing errors with keys 'exp' & 'content'.
// 			Keys only exist if there is an error with them.
type StructCoreAdd struct {
	InputExp     string
	InputContent string
	Errors       map[string]string
}

// does the core execution of a new spit addition.
// @param r: the request of the addition
// returns either the Spit successfully added and saved
// or an error if something went wrong during the creation or save of the spit
// or a StructCoreAdd when the validation of the request arguments failed
func CoreAddSpit(r *http.Request) (*Spit, error, *StructCoreAdd) {
	result := &StructCoreAdd{}

	// do the validation of the parameters
	result.Errors = ValidateSpitParameters(r)
	// if we have errors display the add page again
	if len(result.Errors) > 0 {
		result.InputContent = r.PostFormValue("content")
		result.InputExp = r.PostFormValue("exp")
		return nil, nil, result
	}

	// create the new Spit since everything is fine
	spit, err := NewSpit()
	if err != nil {
		return nil, err, nil
	}
	// ignore error since it passed validation
	spit.Exp, _ = strconv.Atoi(r.PostFormValue("exp"))
	spit.Content = strings.TrimSpace(r.PostFormValue("content"))

	// Save the spit and return the view page
	if err = spit.Save(); err != nil {
		err = errors.New("Could not create your spit, go back and try again")
		return nil, err, nil
	}
	return spit, nil, nil
}
