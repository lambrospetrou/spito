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

var ActiveSpitTypes map[string]bool = map[string]bool{
	SPIT_TYPE_URL:  true,
	SPIT_TYPE_TEXT: true,
}

type Spit struct {
	IdRaw_       uint64    `json:"id_raw"`
	Id_          string    `json:"id"`
	Exp_         int       `json:"exp"`
	Content_     string    `json:"content"`
	DateCreated_ time.Time `json:"date_created"`
	SpitType_    string    `json:"spit_type"`

	// maybe remove this and just use the method to check the spit type
	IsURL_ bool `json:"is_url"`

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

	RemainingExpiration() int
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
	//spit.Exp_ = time.Unix(int64(time.Now().Second())+int64(e), 0).Second()
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

// calculates the remaining expiration - DEDUCTING 1 just to count in the network delays
func (spit *Spit) RemainingExpiration() int {
	return int(spit.DateCreated().Add(time.Duration(spit.Exp())*time.Second).Unix()-
		time.Now().UTC().Unix()) - 1
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

	//fmt.Println("saving: ", spit.Exp_, spit.Content_, spit.SpitType_)
	var epochExp int = 0
	if spit.Exp() > 0 {
		epochExp = int(spit.DateCreated().Unix() + int64(spit.Exp()))
		spit.Exp_ = epochExp
	}
	//fmt.Println(epochExp)

	jsonBytes, err := json.Marshal(spit)
	if err != nil {
		return errors.New("Could not convert Spit to JSON format!")
	}
	err = db.Set("spit::clicks::"+spit.Id_, epochExp, 0)
	if err != nil {
		return errors.New("Could not create the new Spit!")
	}
	return db.SetRaw("spit::"+spit.Id_, epochExp, jsonBytes)
}

func (spit *Spit) Del() error {
	db, err := lpdb.Instance()
	if err != nil {
		return errors.New("Could not get instance of Couchbase")
	}
	if err = db.Delete("spit::clicks::" + spit.Id_); err != nil {
		return errors.New("Could not delete information of Spit!")
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
	timeNow := time.Now().UTC()
	spit := &Spit{Exp_: int(timeNow.Unix()) + (24 * 60), DateCreated_: timeNow}
	// set the ID to maximum uint64 in order to be changed by Save()
	spit.IdRaw_ = math.MaxUint64
	spit.Id_ = ""
	// this is a new spit
	spit.IsExisting_ = false
	return spit, nil
}

type SpitError struct {
	ErrorsMap map[string]string
}

func (e *SpitError) Error() string {
	return fmt.Sprintf("SpitError: %v", e.ErrorsMap)
}

// tries to extract data from the request and map them to a newly created Spit.
// it reads the spit_type in order to determine what spit type will return.
// if there is an error with the parameters then a map of the errors with
// the key being the parameter is returned inside the SpitError.
// Return
//      the spit if everything parsed successfully
//      a SpitError if something went wrong that contains a map[string]string
//			containing any errors occured validating the parameters
func NewFromRequest(r *http.Request) (ISpit, error) {
	exp := r.FormValue("exp")
	spitType := r.FormValue("spit_type")
	content := r.FormValue("content")

	spitError := &SpitError{make(map[string]string)}

	// extract spit type from request
	spitType = strings.TrimSpace(spitType)
	if len(spitType) == 0 {
		spitError.ErrorsMap["SpitType"] = "Empty spit type is not allowed"
	} else {
		if !ActiveSpitTypes[spitType] {
			spitError.ErrorsMap["SpitType"] = "Invalid spit type specified"
		}
	}

	// validate the expiration
	var expInt int
	var err error
	if len(exp) == 0 {
		spitError.ErrorsMap["Exp"] = "Cannot find expiration time"
	} else {
		expInt, err = strconv.Atoi(exp)
		if err != nil {
			spitError.ErrorsMap["Exp"] = "Invalid expiration time posted"
		}
		if expInt < 0 {
			spitError.ErrorsMap["Exp"] = "Negative expiration time not allowed"
		}
	}

	// make sure we are fine so far - MIDDLE CHECK
	if len(spitError.ErrorsMap) > 0 {
		return nil, spitError
	}
	// create the new Spit since everything is fine
	nSpit, err := New()
	if err != nil {
		spitError.ErrorsMap["Generic"] = "Could not create a new spit"
		return nil, spitError
	}
	nSpit.SetSpitType(spitType)
	nSpit.SetExp(expInt)

	// extract the image if this is an image spit
	if spitType == SPIT_TYPE_IMAGE {
		// decode the image posted and check if there is a problem
		img, format, err := parseAndDecodeImage(r)
		if err != nil {
			spitError.ErrorsMap["Image"] = err.Error()
			return nil, spitError
		} else {
			fmt.Println("image format: ", format, " : ", img.Bounds())
		}

		// TODO ---------

	} else if spitType == SPIT_TYPE_TEXT || spitType == SPIT_TYPE_URL {
		// TEXT AND URL SPITS

		content = strings.TrimSpace(content)
		if len(content) == 0 {
			spitError.ErrorsMap["Content"] = "Empty spit is not allowed"
			return nil, spitError
		}
		if len(content) > SPIT_MAX_CONTENT {
			spitError.ErrorsMap["Content"] = fmt.Sprintf("Spit content should be less than %v characters",
				SPIT_MAX_CONTENT)
			return nil, spitError
		}
		nSpit.SetContent(content)

	} // end of spit type checking

	return nSpit, nil
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
