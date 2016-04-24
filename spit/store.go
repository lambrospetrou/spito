package spit

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type Storager interface {
	Put(s *Spit) error
	Get(key string) (*Spit, error)
	GetWithAnalytics(key string) (*Spit, error)
	NextId() (string, error)
}

func NewDynamoStorager() Storager {
	session := session.New()
	svc := dynamodb.New(session, aws.NewConfig().WithRegion("eu-west-1"))
	dynamoDBStorager := &awsDynamoDBStorager{session, svc}
	dynamoDBStorager.init()
	return dynamoDBStorager
}

func NewDefaultStorager() Storager {
	return NewDynamoStorager()
}
