package lpdb

import (
	"errors"
	"fmt"
	"github.com/couchbase/gomemcached"
	"github.com/couchbaselabs/go-couchbase"
	"sync"
)

const (
	_COUCHBASE_URL    = "http://lambrospetrou.com:8091/"
	_COUCHBASE_POOL   = "default"
	_COUCHBASE_BUCKET = "spitty"
)

type CDB struct {
	client couchbase.Client
	pool   couchbase.Pool
	bucket *couchbase.Bucket
}

// singleton since it will not be visible outside this package
var _This_lock sync.Once
var _This *CDB
var err error

func connect() {
	fmt.Println("Starting connection to Couchbase")
	_This = new(CDB)
	_This.client, err = couchbase.Connect(_COUCHBASE_URL)
	if err != nil {
		fmt.Printf("Error connecting:  %v", err)
		err = errors.New("Could not connect to couchbase!")
		return
	}
	_This.pool, err = _This.client.GetPool(_COUCHBASE_POOL)
	if err != nil {
		fmt.Printf("Error getting pool:  %v", err)
		err = errors.New("Could not pool couchbase!")
		return
	}
	_This.bucket, err = _This.pool.GetBucket(_COUCHBASE_BUCKET)
	if err != nil {
		fmt.Printf("Error getting bucket:  %v", err)
		err = errors.New("Could not get bucket " + _COUCHBASE_BUCKET)
		return
	}
	fmt.Printf("Successfully connected to %s\n", _COUCHBASE_URL)
	return
}

func Instance() (*CDB, error) {
	_This_lock.Do(func() {
		connect()
	})
	return _This, err
}

func (db *CDB) Bucket() *couchbase.Bucket {
	return db.bucket
}

func (db *CDB) Get(key string, obj interface{}) error {
	return db.bucket.Get(key, obj)
}

func (db *CDB) GetRaw(key string) ([]byte, error) {
	return db.bucket.GetRaw(key)
}

func (db *CDB) GetBulk(keys []string) (map[string]*gomemcached.MCResponse, error) {
	return db.bucket.GetBulk(keys)
}

func (db *CDB) Set(key string, expiry int, data interface{}) error {
	return db.bucket.Set(key, expiry, data)
}

func (db *CDB) SetRaw(key string, expiry int, data []byte) error {
	return db.bucket.SetRaw(key, expiry, data)
}

func (db *CDB) Delete(key string) error {
	return db.bucket.Delete(key)
}

func (db *CDB) FAI(key string) (uint64, error) {
	// key, incr by amount, initial amount if not exists, ttl
	return db.bucket.Incr(key, 1, 1, 0)
}
