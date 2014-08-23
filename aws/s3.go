package main

import (
	"fmt"
	"io/ioutil"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"log"
	"sync"
)

const (
	_AWS_ACCESS_KEY = "AKIAIETLKEB4IXCKYOKQ"
	_AWS_SECRET_KEY = "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"
	_S3_BUCKET      = "img.spi.to"
)

type S3Struct struct {
	Bucket *s3.Bucket
}

// singleton since it will not be visible outside this package
var _This_lock sync.Once
var _This *S3Struct
var err error

func connect() {
	fmt.Println("connect() called")
	auth := aws.Auth{
		AccessKey: _AWS_ACCESS_KEY,
		SecretKey: _AWS_SECRET_KEY,
	}
	euwest := aws.EUWest

	connection := s3.New(auth, euwest)
	mybucket := connection.Bucket(_S3_BUCKET)
	_This = &S3Struct{Bucket: mybucket}
}

func Instance() (*S3Struct, error) {
	_This_lock.Do(func() {
		connect()
	})
	return _This, err
}

func UploadImage(fileName string, img []byte, mimeType string) (string, error) {
	mS3, _ := Instance()
	err := mS3.Bucket.Put(fileName, img, mimeType, s3.BucketOwnerFull)
	if err != nil {
		return "", err
	}
	return "Uploaded", nil
}

///////////////// TESTING it ////////////////////////

func listFiles(i int, ch chan int) {
	myS3, _ := Instance()
	res, err := myS3.Bucket.List("", "", "", 1000)
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range res.Contents {
		fmt.Println(i, v.Key)
	}
	ch <- i
}

func main() {
	ch := make(chan int, 10)
	for i := 0; i < 10; i++ {
		go listFiles(i, ch)
	}
	for i := 0; i < 10; i++ {
		fmt.Println(<-ch, " finished")
	}

	img, err := ioutil.ReadFile("img.jpg")
	if err != nil {
		panic("testign image does not exist")
	}
	UploadImage("_public/testing_image.jpg", img, "image/jpeg")
}
