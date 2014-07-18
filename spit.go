package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/lambrospetrou/spitty/lpdb"
	"strconv"
	"time"
)

type Spit struct {
	IdRaw       uint64    `json:"id_raw"`
	Id          string    `json:"id"`
	Exp         int       `json:"exp"`
	Content     string    `json:"content"`
	DateCreated time.Time `json:"date_created"`
}

func (spit *Spit) FormattedCreatedTime() string {
	return spit.DateCreated.Format("January 02, 2006 | Monday")
}

func (spit *Spit) Save() error {
	spit.DateCreated = time.Now()
	if spit.Exp < 0 {
		spit.Exp = 0
	}
	db, err := lpdb.CDBInstance()
	if err != nil {
		return errors.New("Could not get instance of Couchbase")
	}
	jsonBytes, err := json.Marshal(spit)
	if err != nil {
		return errors.New("Could not convert Spit to JSON format!")
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
	return spit, nil
}

func NewSpit() (*Spit, error) {
	spit := &Spit{Exp: 24 * 60, DateCreated: time.Now()}
	db, err := lpdb.CDBInstance()
	if err != nil {
		return nil, errors.New("Could not connect to Couchbase")
	}
	spit_raw_id, err := db.FAI("spit::count")
	if err != nil {
		return nil, errors.New("Could not create unique id for Spit.")
	}
	spit.IdRaw = spit_raw_id
	spit.Id = base64.URLEncoding.EncodeToString([]byte(strconv.FormatUint(spit.IdRaw, 10)))
	//spit.DateCreated = time.Now()
	//spit.Exp = 0
	return spit, nil
}
