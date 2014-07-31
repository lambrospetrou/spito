package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lambrospetrou/goencoding/lpenc"
	"github.com/lambrospetrou/spitty/lpdb"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const SPIT_MAX_CONTENT int = 10000

const SPIT_ID_CHARS string = "Ca1MoKtUR5A2BfeGm8LWwlFgHOx3hNk9ciTpuqZ7nrQjXyzJbvI64V0EYPsDSd"

var SpitIdEncoding = lpenc.NewEncoding(SPIT_ID_CHARS)

type Spit struct {
	IdRaw       uint64    `json:"id_raw"`
	Id          string    `json:"id"`
	Exp         int       `json:"exp"`
	Content     string    `json:"content"`
	DateCreated time.Time `json:"date_created"`
	IsURL       bool      `json:"is_url"`
	Clicks      uint64    `json:"-"`
	IsExisting  bool      `json:"-"`
}

func (spit *Spit) FormattedCreatedTime() string {
	return spit.DateCreated.Format("January 02, 2006 | Monday")
}

func (spit *Spit) Save() error {
	db, err := lpdb.CDBInstance()
	if err != nil {
		return errors.New("Could not get instance of Couchbase")
	}

	// make sure a valid ID exists
	if !spit.IsExisting && spit.IdRaw == math.MaxUint64 {
		spit_raw_id, err := nextSpitId(db)
		if err != nil {
			return errors.New("Could not create a unique id for the new spit")
		}
		spit.IdRaw = spit_raw_id
		spit.Id = SpitIdEncoding.Encode(spit.IdRaw)
	}
	spit.DateCreated = time.Now()
	if spit.Exp < 0 {
		spit.Exp = 0
	}
	content := strings.TrimSpace(spit.Content)
	spit.IsURL = isUrl(content)
	if spit.IsURL {
		spit.Content = content
	}
	jsonBytes, err := json.Marshal(spit)
	if err != nil {
		return errors.New("Could not convert Spit to JSON format!")
	}
	err = db.Set("spit::clicks::"+spit.Id, spit.Exp, 0)
	if err != nil {
		return errors.New("Could not create the new Spit!")
	}
	return db.SetRaw("spit::"+spit.Id, spit.Exp, jsonBytes)
}

func (spit *Spit) Del() error {
	db, err := lpdb.CDBInstance()
	if err != nil {
		return errors.New("Could not get instance of Couchbase")
	}
	return db.Delete("spit::" + spit.Id)
}

func (spit *Spit) AbsoluteURL() string {
	return AbsoluteSpittyURL(spit.Id)
}

func (spit *Spit) ClickInc() error {
	db, err := lpdb.CDBInstance()
	if err != nil {
		return errors.New("Could not connect to Couchbase")
	}
	_, err = db.FAI("spit::clicks::" + spit.Id)
	if err != nil {
		return errors.New("Could not increase clicks for Spit.")
	}
	return nil
}

/////////////////////////////////////////////////////
////////////////// GENERAL FUNCTIONS
/////////////////////////////////////////////////////

func LoadSpit(id string) (*Spit, error) {
	spit := &Spit{}
	db, err := lpdb.CDBInstance()
	if err != nil {
		return nil, errors.New("Could not connect to Couchbase")
	}
	err = db.Get("spit::"+id, &spit)
	if err != nil {
		return nil, errors.New("No Spit exists with this Id")
	}
	err = db.Get("spit::clicks::"+id, &spit.Clicks)
	if err != nil {
		return nil, errors.New("No clicks exist for this Spit")
	}
	spit.IsExisting = true
	return spit, nil
}

func NewSpit() (*Spit, error) {
	// use UTC time everywhere
	spit := &Spit{Exp: 24 * 60, DateCreated: time.Now().UTC()}
	// set the ID to maximum uint64 in order to be changed by Save()
	spit.IdRaw = math.MaxUint64
	spit.Id = ""
	// this is a new spit
	spit.IsExisting = false
	return spit, nil
}

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

func ValidateSpitParameters(r *http.Request) map[string]string {
	errorsMap := make(map[string]string)
	// validate the fields
	var expInt int
	exp := r.PostFormValue("exp")
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

	content := strings.TrimSpace(r.PostFormValue("content"))
	if len(content) == 0 {
		errorsMap["Content"] = "Empty spit is not allowed"
	}
	if len(content) > SPIT_MAX_CONTENT {
		errorsMap["Content"] = fmt.Sprintf("Spit content should be less than %v characters",
			SPIT_MAX_CONTENT)
	}
	return errorsMap
}
