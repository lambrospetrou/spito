package spit

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lambrospetrou/goencoding/lpenc"
	"github.com/lambrospetrou/spito/lpdb"
	"github.com/lambrospetrou/spito/utils"
	"image"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const SPIT_MAX_CONTENT int = 10000

const SPIT_ID_CHARS string = "Ca1MoKtUR5A2BfeGm8LWwlFgHOx3hNk9ciTpuqZ7nrQjXyzJbvI64V0EYPsDSd"

var SpitIdEncoding = lpenc.NewEncoding(SPIT_ID_CHARS)

const (
	SPIT_TYPE_URL   string = "url"
	SPIT_TYPE_TEXT  string = "text"
	SPIT_TYPE_IMAGE string = "img"
)

type Spit struct {
	IdRaw_       uint64    `json:"id_raw"`
	Id_          string    `json:"id"`
	Exp_         int       `json:"exp"`
	Content_     string    `json:"content"`
	DateCreated_ time.Time `json:"date_created"`
	IsURL_       bool      `json:"is_url"`
	SpitType_    string    `json:"spit_type"`

	Clicks_     uint64 `json:"-"`
	IsExisting_ bool   `json:"-"`
}

type ImageSpit struct {
	Spit

	ImageFormat_ string      `json:"image_fmt"`
	ImageBytes_  []byte      `json:"-"`
	Image_       image.Image `json:"-"`
}

type ISpit interface {
	IdRaw() uint64
	Id() string
	Exp() int
	Content() string
	DateCreated() time.Time
	IsURL() bool
	SpitType() string

	SetIdRaw(uint64) uint64
	SetId(string) string
	SetExp(int) int
	SetContent(string) string
	SetSpitType(string) string

	FormattedCreatedTime() string
	AbsoluteURL() string
	Clicks() uint64
	IsExisting() bool

	Save() error
	Del() error

	ClickInc() error
}

func (spit *Spit) SetIdRaw(idraw uint64) uint64 {
	spit.IdRaw_ = idraw
	return spit.IdRaw_
}

func (spit *Spit) SetId(id string) string {
	spit.Id_ = id
	return spit.Id_
}

func (spit *Spit) SetExp(e int) int {
	spit.Exp_ = e
	return spit.Exp_
}

func (spit *Spit) SetContent(c string) string {
	spit.Content_ = c
	return spit.Content_
}

func (spit *Spit) SetSpitType(t string) string {
	spit.SpitType_ = t
	return spit.SpitType_
}

func (spit *Spit) IdRaw() uint64 {
	return spit.IdRaw_
}

func (spit *Spit) Id() string {
	return spit.Id_
}

func (spit *Spit) Exp() int {
	return spit.Exp_
}

func (spit *Spit) Content() string {
	return spit.Content_
}

func (spit *Spit) DateCreated() time.Time {
	return spit.DateCreated_
}

func (spit *Spit) IsURL() bool {
	return spit.IsURL_
}

func (spit *Spit) SpitType() string {
	return spit.SpitType_
}

func (spit *Spit) Clicks() uint64 {
	return spit.Clicks_
}

func (spit *Spit) IsExisting() bool {
	return spit.IsExisting_
}

func (spit *Spit) FormattedCreatedTime() string {
	return spit.DateCreated_.Format("January 02, 2006 | Monday")
}

func (spit *Spit) Save() error {
	db, err := lpdb.Instance()
	if err != nil {
		return errors.New("Could not get instance of Couchbase")
	}

	// make sure a valid ID exists
	if !spit.IsExisting_ && spit.IdRaw_ == math.MaxUint64 {
		spit_raw_id, err := nextSpitId(db)
		if err != nil {
			return errors.New("Could not create a unique id for the new spit")
		}
		spit.IdRaw_ = spit_raw_id
		spit.Id_ = SpitIdEncoding.Encode(spit.IdRaw_)
	}
	spit.DateCreated_ = time.Now()
	if spit.Exp_ < 0 {
		spit.Exp_ = 0
	}
	content := strings.TrimSpace(spit.Content_)
	spit.IsURL_ = utils.IsUrl(content)
	if spit.IsURL_ {
		spit.Content_ = content
	}
	jsonBytes, err := json.Marshal(spit)
	if err != nil {
		return errors.New("Could not convert Spit to JSON format!")
	}
	err = db.Set("spit::clicks::"+spit.Id_, spit.Exp_, 0)
	if err != nil {
		return errors.New("Could not create the new Spit!")
	}
	return db.SetRaw("spit::"+spit.Id_, spit.Exp_, jsonBytes)
}

func (spit *Spit) Del() error {
	db, err := lpdb.Instance()
	if err != nil {
		return errors.New("Could not get instance of Couchbase")
	}
	return db.Delete("spit::" + spit.Id_)
}

func (spit *Spit) AbsoluteURL() string {
	return utils.AbsoluteSpitoURL(spit.Id_)
}

func (spit *Spit) ClickInc() error {
	db, err := lpdb.Instance()
	if err != nil {
		return errors.New("Could not connect to Couchbase")
	}
	_, err = db.FAI("spit::clicks::" + spit.Id_)
	if err != nil {
		return errors.New("Could not increase clicks for Spit.")
	}
	return nil
}

/////////////////////////////////////////////////////
////////////////// GENERAL FUNCTIONS
/////////////////////////////////////////////////////

func Load(id string) (ISpit, error) {

	// TODO - determine type quickly without marshalling into any specific type
	// http://blog.golang.org/json-and-go#TOC_5.

	spit := &Spit{}
	db, err := lpdb.Instance()
	if err != nil {
		return nil, errors.New("Could not connect to Couchbase")
	}
	err = db.Get("spit::"+id, &spit)
	if err != nil {
		return nil, errors.New("No Spit exists with this Id")
	}
	err = db.Get("spit::clicks::"+id, &spit.Clicks_)
	if err != nil {
		return nil, errors.New("No clicks exist for this Spit")
	}
	spit.IsExisting_ = true
	return spit, nil
}

func New() (ISpit, error) {
	// use UTC time everywhere
	spit := &Spit{Exp_: 24 * 60, DateCreated_: time.Now().UTC()}
	// set the ID to maximum uint64 in order to be changed by Save()
	spit.IdRaw_ = math.MaxUint64
	spit.Id_ = ""
	// this is a new spit
	spit.IsExisting_ = false
	return spit, nil
}

// tries to extract data from the request and map them to a newly created Spit.
// it reads the spit_type in order to determine what spit type will return.
// if there is an error with the parameters then a map of the errors with
// the key being the parameter is returned.
// Return
//      the spit if everything parsed successfully
//      an error if something went wrong
//      a map[string]string containing any errors occured validating the parameters
/*
func NewFromRequest(r *http.Request) (*ISpit, error, map[string]string) {
	var reqErrors map[string]string

	// do the validation of the parameters
	reqErrors = ValidateSpitRequest(r)
	// if we have errors display the add page again
	if len(result.Errors) > 0 {
		return nil, errors.New("Request has invalid spit parameters"), reqErrors
	}

	// create the new Spit since everything is fine
	nSpit, err := New()
	if err != nil {
		return nil, err, nil
	}

	// ignore error since it passed validation
	nSpit.Exp, _ = strconv.Atoi(r.FormValue["exp"])
	nSpit.Content = strings.TrimSpace(r.FormValue["content"])
	nSpit.SpitType = strings.TrimSpace(r.FormValue["spit_type"])

	return s
}
*/

func nextSpitId(db *lpdb.CDB) (uint64, error) {
	raw_id, err := db.FAI("spit::count")
	if err != nil {
		return math.MaxUint64, errors.New("Could not create unique id for Spit.")
	}
	return raw_id, nil
}

func ValidateSpitID(id string) bool {
	_, err := SpitIdEncoding.Decode(id)
	return err == nil
}

func ValidateSpitRequest(r *http.Request) map[string]string {
	exp := r.FormValue("exp")
	spitType := r.FormValue("spit_type")
	content := r.FormValue("content")

	errorsMap := make(map[string]string)
	// validate the fields
	var expInt int
	if len(exp) == 0 {
		errorsMap["Exp"] = "Cannot find expiration time"
	} else {
		_, err := strconv.Atoi(exp)
		if err != nil {
			errorsMap["Exp"] = "Invalid expiration time posted"
		}
		if expInt < 0 {
			errorsMap["Exp"] = "Negative expiration time not allowed"
		}
	}

	spitType = strings.TrimSpace(spitType)
	if len(spitType) == 0 {
		errorsMap["SpitType"] = "Empty spit type is not allowed"
	} else {
		if spitType != SPIT_TYPE_IMAGE &&
			spitType != SPIT_TYPE_TEXT &&
			spitType != SPIT_TYPE_URL {
			errorsMap["SpitType"] = "Wrong spit type specified"
		}
	}

	// extract the image if this is an image spit
	if spitType == SPIT_TYPE_IMAGE {
		// decode the image posted and check if there is a problem
		img, format, err := parseAndDecodeImage(r)
		if err != nil {
			errorsMap["Image"] = err.Error()
		} else {
			fmt.Println("IMAGE format: ", format, " : ", img.Bounds())
		}

	} else if spitType == SPIT_TYPE_TEXT || spitType == SPIT_TYPE_URL {
		// TEXT AND URL SPITS
		content = strings.TrimSpace(content)
		if len(content) == 0 {
			errorsMap["Content"] = "Empty spit is not allowed"
		}
		if len(content) > SPIT_MAX_CONTENT {
			errorsMap["Content"] = fmt.Sprintf("Spit content should be less than %v characters",
				SPIT_MAX_CONTENT)
		}
	}

	return errorsMap
}

func parseAndDecodeImage(r *http.Request) (image.Image, string, error) {
	fmt.Println("decoding")
	multiFile, multiHeader, err := r.FormFile("image")
	//multiFile, _, err := r.FormFile("image")
	if err != nil {
		return nil, "", errors.New("Cannot extract the image from the submitted form")
	}
	fmt.Println(multiHeader.Filename, multiFile.Close())

	// decode the image posted and check if there is a problem
	img, format, err := image.Decode(multiFile)
	if err != nil {
		return nil, "unknown", errors.New("You submitted an Invalid image")
	}
	return img, format, nil
}

func ValidateSpitValues(values map[string]string) map[string]string {
	return ValidateSpitParameters(values["exp"],
		values["spit_type"],
		values["content"])
}

func ValidateSpitParameters(exp, spitType, content string) map[string]string {
	errorsMap := make(map[string]string)
	// validate the fields
	var expInt int
	if len(exp) == 0 {
		errorsMap["Exp"] = "Cannot find expiration time"
	} else {
		_, err := strconv.Atoi(exp)
		if err != nil {
			errorsMap["Exp"] = "Invalid expiration time posted"
		}
		if expInt < 0 {
			errorsMap["Exp"] = "Negative expiration time not allowed"
		}
	}

	spitType = strings.TrimSpace(spitType)
	if len(spitType) == 0 {
		errorsMap["SpitType"] = "Empty spit type is not allowed"
	} else {
		if spitType != SPIT_TYPE_IMAGE &&
			spitType != SPIT_TYPE_TEXT &&
			spitType != SPIT_TYPE_URL {
			errorsMap["SpitType"] = "Wrong spit type specified"
		}
	}

	// extract the image if this is an image spit
	if spitType == SPIT_TYPE_IMAGE {
		// decode the image posted and check if there is a problem
		/*
			img, format, err := ParseAndDecodeImage(r)
			if err != nil {
				errorsMap["Image"] = err.Error()
			}
			fmt.Println("IMAGE format: ", format, " : ", img.Bounds())
		*/
	} else if spitType == SPIT_TYPE_TEXT || spitType == SPIT_TYPE_URL {
		// TEXT AND URL SPITS
		content = strings.TrimSpace(content)
		if len(content) == 0 {
			errorsMap["Content"] = "Empty spit is not allowed"
		}
		if len(content) > SPIT_MAX_CONTENT {
			errorsMap["Content"] = fmt.Sprintf("Spit content should be less than %v characters",
				SPIT_MAX_CONTENT)
		}
	}

	return errorsMap
}
