package main

import (
	"errors"
	"fmt"
	"github.com/lambrospetrou/spito/s3"
	"github.com/lambrospetrou/spito/spit"
	"image"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MAX_FORM_SIZE int64 = 1 << 23
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
func CoreAddSpit(r *http.Request) (*spit.Spit, error, *StructCoreAdd) {
	result := &StructCoreAdd{}

	// do the validation of the parameters
	result.Errors = spit.ValidateSpitRequest(r)
	// if we have errors display the add page again
	if len(result.Errors) > 0 {
		result.InputContent = r.PostFormValue("content")
		result.InputExp = r.PostFormValue("exp")
		result.SpitType = r.PostFormValue("spit_type")
		return nil, nil, result
	}

	// create the new Spit since everything is fine
	s, err := spit.New()
	if err != nil {
		return nil, err, nil
	}
	// ignore error since it passed validation
	s.Exp, _ = strconv.Atoi(r.PostFormValue("exp"))
	s.Content = strings.TrimSpace(r.PostFormValue("content"))
	s.SpitType = strings.TrimSpace(r.PostFormValue("spit_type"))

	// Save the spit
	if err = s.Save(); err != nil {
		err = errors.New("Could not create your spit, go back and try again")
		return nil, err, nil
	}
	return s, nil, nil
}

// does the core execution of a new media spit addition.
// @param r: the request of the addition
// returns either the Spit successfully added and saved
// or an error if something went wrong during the creation or save of the spit
// or a StructCoreAdd when the validation of the request arguments failed
func CoreAddMultiSpit(r *http.Request) (*spit.Spit, error, *StructCoreAdd) {
	result := &StructCoreAdd{}

	// try to parse the form with a maximum size
	err := r.ParseMultipartForm(MAX_FORM_SIZE)
	if err != nil {
		result.Errors = make(map[string]string)
		result.Errors["Generic"] = "Too much data submitted. Try with a smaller image. (up to 8MB)"
		return nil, nil, result
	}

	var values = make(map[string]string)
	values["exp"] = r.FormValue("exp")
	values["content"] = r.FormValue("content")
	values["spit_type"] = r.FormValue("spit_type")

	// do the validation of the parameters
	result.Errors = spit.ValidateSpitValues(values)
	// if we have errors display the add page again
	if len(result.Errors) > 0 {
		result.InputContent = values["content"]
		result.InputExp = values["exp"]
		result.SpitType = values["spit_type"]
		return nil, nil, result
	}

	// extract the image if this is an image spit
	if values["spit_type"] == spit.SPIT_TYPE_IMAGE {
		// decode the image posted and check if there is a problem
		img, format, err := ParseAndDecodeImage(r)
		if err != nil {
			result.Errors = make(map[string]string)
			result.Errors["Image"] = err.Error()
			return nil, nil, result
		}
		fmt.Println("format: ", format, " : ", img.Bounds())

		// try to uplaod the image in amazon S3 and get the link
		exp, _ := strconv.Atoi(values["exp"])
		filePath, err := AWSUploadImage(r, img, format)
		if err != nil {
			result.Errors = make(map[string]string)
			result.Errors["Generic"] = err.Error()
			return nil, nil, result
		}
		fmt.Println("filePath: ", filePath)

		urlS, err := AWSSignedURL(filePath, exp)
		if err != nil {
			result.Errors = make(map[string]string)
			result.Errors["Generic"] = err.Error()
			return nil, nil, result
		}
		fmt.Println("urlSigned: ", urlS)

		values["aws_image_url"] = urlS
	}

	////////////////////////////////////////////////////////

	// create the new Spit since everything is fine
	s, err := spit.New()
	if err != nil {
		return nil, err, nil
	}

	// ignore error since it passed validation
	s.Exp, _ = strconv.Atoi(values["exp"])
	s.Content = strings.TrimSpace(values["content"])
	s.SpitType = strings.TrimSpace(values["spit_type"])

	// Save the spit
	if err = s.Save(); err != nil {
		err = errors.New("Could not save your spit, go back and try again")
		return nil, err, nil
	}
	return s, nil, nil
}

func ParseAndDecodeImage(r *http.Request) (image.Image, string, error) {
	//multiFile, multiHeader, err := r.FormFile("image")
	multiFile, _, err := r.FormFile("image")
	if err != nil {
		return nil, "", errors.New("Cannot extract the image from the submitted form")
	}
	//fmt.Println(multiHeader.Filename, multiFile.Close())

	// decode the image posted and check if there is a problem
	img, format, err := DecodeImage(multiFile)
	if err != nil {
		return nil, "unknown", errors.New("You submitted an Invalid image")
	}
	return img, format, nil
}

// uploads the image from the request.
// tries to extract the image from the parameter 'image'
// and it is assumed to be a multipart form
// returns the filePath where the image is stored on AWS S3 and if an error occured
func AWSUploadImage(r *http.Request, img image.Image, format string) (string, error) {
	multiFile, multiHeader, err := r.FormFile("image")
	if err != nil {
		return "", errors.New("Cannot extract the image from the submitted form")
	}
	fmt.Println(multiHeader.Filename, multiHeader.Header)

	b, err := ioutil.ReadAll(multiFile)
	if err != nil {
		return "", errors.New("Cannot read the image from the submitted form")
	}

	s3struct, err := s3.Instance()
	if err != nil {
		return "", errors.New("Cannot get Amazon S3 instance")
	}
	filePath := "_signed/" + multiHeader.Filename
	s, err := s3struct.UploadImage(filePath, b, "image/"+format)
	if err != nil {
		return "", errors.New("Cannot store image")
	}
	fmt.Println(s, err)
	return filePath, nil
}

// tries to generate a signed URL for the filePath specified that expires
// after the specfied exp time has passed from now.
func AWSSignedURL(filePath string, exp int) (string, error) {
	s3struct, err := s3.Instance()
	if err != nil {
		return "", errors.New("Cannot get Amazon S3 instance")
	}
	expTime := time.Now().Add(time.Duration(exp) * time.Second)
	urlS := s3struct.SignedURL(filePath, expTime)
	if r := recover(); r != nil {
		return "", errors.New("Cannot create a URL for the image")
	}

	return urlS, nil
}
