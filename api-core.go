package main

import (
	"errors"
	"fmt"
	"github.com/lambrospetrou/spito/s3"
	"github.com/lambrospetrou/spito/spit"
	"image"
	"io/ioutil"
	"log"
	"net/http"
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
type ErrCoreAdd struct {
	InputExp     string
	InputContent string
	Errors       map[string]string
	SpitType     string
}

func (e *ErrCoreAdd) Error() string {
	return fmt.Sprintf("ErrCoreAdd: %v", e.Errors)
}

type ErrCoreAddDB struct {
	NewSpit spit.ISpit
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
func CoreAddMultiSpit(r *http.Request) (spit.ISpit, error) {
	result := &ErrCoreAdd{}

	// try to parse the form with a maximum size
	err := r.ParseMultipartForm(MAX_FORM_SIZE)
	if err != nil {
		result.Errors = make(map[string]string)
		result.Errors["Generic"] = "Too much data submitted (up to 8MB) or invalid form data!"
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

	// Save the spit
	if err = nSpit.Save(); err != nil {
		errDB := &ErrCoreAddDB{NewSpit: nSpit, Message: "Could not save spit in database!"}
		log.Printf("%s, %v", err.Error(), errDB)
		return nil, errDB
	}
	return nSpit, nil
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
