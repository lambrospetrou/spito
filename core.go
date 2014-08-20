package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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
	SpitType     string
}

// does the core execution of a new spit addition.
// @param r: the request of the addition
// returns either the Spit successfully added and saved
// or an error if something went wrong during the creation or save of the spit
// or a StructCoreAdd when the validation of the request arguments failed
func CoreAddSpit(r *http.Request) (*Spit, error, *StructCoreAdd) {
	result := &StructCoreAdd{}

	// do the validation of the parameters
	result.Errors = ValidateSpitRequest(r)
	// if we have errors display the add page again
	if len(result.Errors) > 0 {
		result.InputContent = r.PostFormValue("content")
		result.InputExp = r.PostFormValue("exp")
		result.SpitType = r.PostFormValue("spit_type")
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

	// Save the spit
	if err = spit.Save(); err != nil {
		err = errors.New("Could not create your spit, go back and try again")
		return nil, err, nil
	}
	return spit, nil, nil
}

// does the core execution of a new media spit addition.
// @param r: the request of the addition
// returns either the Spit successfully added and saved
// or an error if something went wrong during the creation or save of the spit
// or a StructCoreAdd when the validation of the request arguments failed
func CoreAddMultiSpit(r *http.Request) (*Spit, error, *StructCoreAdd) {
	result := &StructCoreAdd{}

	// 8MB memory
	reader, err := r.MultipartReader()
	if err != nil {
		result.Errors = make(map[string]string)
		result.Errors["generic"] = "Cannot parse the submitted form"
		return nil, nil, result
	}

	var values = make(map[string]string)
	var buf = new(bytes.Buffer)
	for {
		part, err := reader.NextPart()
		if err != nil {
			// no other part exists
			break
		}

		if part.FileName() != "" {
			// this is a file

			continue
		}

		// handle regular Form data
		if part.FormName() == "" {
			continue
		}

		fmt.Println(part.FormName())
		if _, err := io.Copy(buf, part); err != nil {
			result.Errors = make(map[string]string)
			result.Errors["generic"] = "Cannot read parts of the submitted form"
			return nil, nil, result
		}
		values[part.FormName()] = buf.String()
		fmt.Println(buf.String())

		buf.Reset()
	}

	// do the validation of the parameters
	result.Errors = ValidateSpitValues(values)
	// if we have errors display the add page again
	if len(result.Errors) > 0 {
		result.InputContent = values["content"]
		result.InputExp = values["exp"]
		result.SpitType = values["spit_type"]
		return nil, nil, result
	}

	// create the new Spit since everything is fine
	spit, err := NewSpit()
	if err != nil {
		return nil, err, nil
	}

	// try to uplaod the image in amazon S3 and get the link
	// TODO

	// ignore error since it passed validation
	spit.Exp, _ = strconv.Atoi(values["exp"])
	spit.Content = strings.TrimSpace(values["content"])
	spit.SpitType = strings.TrimSpace(values["spit_type"])

	// Save the spit
	if err = spit.Save(); err != nil {
		err = errors.New("Could not create your spit, go back and try again")
		return nil, err, nil
	}
	return spit, nil, nil
}
