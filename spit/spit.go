package spit

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lambrospetrou/spito/ids"
	"github.com/lambrospetrou/spito/utils"
)

const (
	SPIT_MAX_CONTENT int = 10000
)

func _BuildSpitKey(id string) string {
	return _SPIT_KEY_PREFIX + id
}

func _BuildSpitIdFromKey(key string) string {
	return strings.Replace(key, _SPIT_KEY_PREFIX, "", -1)
}

var storager Storager = nil

func init() {
	storager = NewDefaultStorager()
}

const (
	SPIT_TYPE_URL  string = "url"
	SPIT_TYPE_TEXT string = "text"
)

var ActiveSpitTypes map[string]bool = map[string]bool{
	SPIT_TYPE_URL:  true,
	SPIT_TYPE_TEXT: true,
}

type Spit struct {
	Id             string `json:"id"`
	Exp            int    `json:"exp"`
	Content        string `json:"content"`
	DateCreated    string `json:"date_created"`
	DateExpiration string `json:"date_expiration"`
	SpitType       string `json:"spit_type"`
	MetricClicks   uint64 `json:"metric_clicks"`
}

func (spit *Spit) DateCreatedTime() time.Time {
	t, _ := time.Parse(time.RFC3339, spit.DateCreated)
	return t
}

func (spit *Spit) FormattedCreatedTime() string {
	return spit.DateCreatedTime().Format("January 02, 2006 | Monday")
}

// calculates the remaining expiration - DEDUCTING 1 just to count in the network delays
func (spit *Spit) RemainingExpiration() int {
	return int(spit.DateCreatedTime().Add(time.Duration(spit.Exp)*time.Second).Unix()-
		time.Now().UTC().Unix()) - 1
}

func (spit *Spit) IdHashOnly() string {
	return _BuildSpitIdFromKey(spit.Id)
}

func (spit *Spit) Save() error {
	id, err := storager.NextId()
	if err != nil {
		log.Println("Error while building next id: ", err, spit)
		return err
	}
	spit.Id = _BuildSpitKey(id)
	if err = storager.Put(spit); err != nil {
		log.Println(err)
	}
	spit.Id = _BuildSpitIdFromKey(spit.Id)
	return nil
}

/////////////////////////////////////////////////////
////////////////// GENERAL FUNCTIONS
/////////////////////////////////////////////////////
/*
func nextSpitId(db *lpdb.CDB) (uint64, error) {
	raw_id, err := db.FAI("spit::count")
	if err != nil {
		return math.MaxUint64, errors.New("Could not create unique id for Spit.")
	}
	return raw_id, nil
}
*/
func ValidateSpitId(id string) bool {
	return ids.ValidateId(id)
}

func AbsoluteUrl(spit *Spit) string {
	return utils.AbsoluteSpitoURL(spit.IdHashOnly())
}

func IsUrl(spit *Spit) bool {
	return spit.SpitType == SPIT_TYPE_URL
}

type SpitError struct {
	ErrorsMap map[string]string
}

func (e *SpitError) Error() string {
	return fmt.Sprintf("SpitError: %v", e.ErrorsMap)
}

func Load(id string) (*Spit, error) {
	return storager.GetWithAnalytics(_BuildSpitKey(id))
}

func _NewSpit(content string, exp int, spit_type string) (*Spit, error) {
	// use UTC time everywhere
	timeNow := time.Now().UTC()
	// Parse the expiration - Assume that now it is a number of seconds
	expirationDate := timeNow.Add(time.Duration(exp) * time.Second)

	spit := &Spit{
		SpitType:       spit_type,
		Exp:            exp,
		DateExpiration: expirationDate.Format(time.RFC3339),
		DateCreated:    timeNow.Format(time.RFC3339),
		Content:        content,
		MetricClicks:   0,
		Id:             "",
	}
	return spit, nil
}

func NewUrlSpit(url string, exp int) (*Spit, error) {
	return _NewSpit(url, exp, SPIT_TYPE_URL)
}

func NewTextSpit(text string, exp int) (*Spit, error) {
	return _NewSpit(text, exp, SPIT_TYPE_TEXT)
}

// NewFromRequest tries to extract data from the request and map them to a newly created Spit.
// it reads the spit_type in order to determine what spit type will return.
// if there is an error with the parameters then a map of the errors with
// the key being the parameter is returned inside the SpitError.
// Return
//      the spit if everything parsed successfully
//      a SpitError if something went wrong that contains a map[string]string
//			containing any errors occured validating the parameters
func NewFromRequest(r *http.Request) (*Spit, error) {
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
	// make sure the URL is correct if it is a URL type
	if spitType == SPIT_TYPE_URL {
		isurl := utils.IsUrl(content)
		if !isurl {
			spitError.ErrorsMap["Content"] = "URL specified is not valid..."
			return nil, spitError
		}
		return NewUrlSpit(content, expInt)
	}

	return NewTextSpit(content, expInt)
}
