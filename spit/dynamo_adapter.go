package spit

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func _BuildSpitFromDynamo(dbAttrValue map[string]*dynamodb.AttributeValue, ns *Spit) (*Spit, error) {
	if ns == nil {
		ns = &Spit{}
	}
	if err := dynamodbattribute.UnmarshalMap(dbAttrValue, ns); err != nil {
		log.Println("Error while unmarshalling DynamoDB item: ", err)
		return nil, err
	}
	return ns, nil
}

func _BuildDynamoAtributeValueFromSpit(s *Spit) *dynamodb.AttributeValue {
	av, err := dynamodbattribute.Marshal(s)
	if err != nil {
		log.Println("Error while marshalling Spit to DynamoDB item: ", err)
		return nil
	}
	return av
}

const (
	_TABLE_NAME_SPITS_DATA = "SpitsData"
	_TABLE_NAME_SPITS_META = "SpitsMeta"
)

type awsDynamoDBStorager struct {
	session *session.Session
	svc     *dynamodb.DynamoDB
}

type DynamoDbItemNotFoundError struct {
	msg string
}

func (e DynamoDbItemNotFoundError) Error() string {
	return fmt.Sprintf("DynamoDbItemNotFoundError: %v", e.msg)
}

func (p *awsDynamoDBStorager) Put(s *Spit) error {
	av := _BuildDynamoAtributeValueFromSpit(s)
	if av == nil {
		return errors.New("dynamo_adapter::Put::Could not marshal Spit")
	}

	params := &dynamodb.PutItemInput{
		Item:      av.M,
		TableName: aws.String(_TABLE_NAME_SPITS_DATA),
	}
	resp, err := p.svc.PutItem(params)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and Message from an error.
		log.Println("dynamo_adapter::Put::", err.Error(), resp)
		return err
	}
	return nil
}

func (p *awsDynamoDBStorager) Get(key string) (*Spit, error) {
	s := &Spit{}
	err := p.GetRaw(_TABLE_NAME_SPITS_DATA, "id", key, s)
	if err != nil {
		// Only log if it is an error, not just Item not Found
		if _, ok := err.(DynamoDbItemNotFoundError); !ok {
			log.Println("dynamo_adapter::Get::", err)
		}
		return nil, err
	}

	// Check the expiration date and delete it if necessary
	timeNow := time.Now().UTC()
	timeThen, _ := time.Parse(time.RFC3339, s.DateExpiration)
	if s.Exp > 0 && timeThen.Before(timeNow) {
		// delete the item and return nil
		params := &dynamodb.DeleteItemInput{
			Key: map[string]*dynamodb.AttributeValue{ // Required
				"id": {
					S: aws.String(key),
				},
			},
			TableName: aws.String(_TABLE_NAME_SPITS_DATA),
		}
		resp, err := p.svc.DeleteItem(params)
		if err != nil {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			log.Println("dynamo_adapter::Put::", err.Error(), resp)
			return nil, err
		}
		return nil, errors.New("Spit expired!")
	}
	return s, nil
}

func (p *awsDynamoDBStorager) GetWithAnalytics(key string) (*Spit, error) {
	_, err := p.Get(key)
	if err != nil {
		return nil, err
	}
	// Update the clicks
	params := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{ // Required
			"id": {
				S: aws.String(key),
			},
		},
		AttributeUpdates: map[string]*dynamodb.AttributeValueUpdate{
			"metric_clicks": {
				Action: aws.String("ADD"),
				Value: &dynamodb.AttributeValue{
					N: aws.String("1"),
				},
			},
		},
		ReturnValues: aws.String("ALL_NEW"),
		TableName:    aws.String(_TABLE_NAME_SPITS_DATA),
	}
	resp, err := p.svc.UpdateItem(params)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and Message from an error.
		log.Println("dynamo_adapter::GetWithAnalytics::", err.Error(), resp)
		return nil, err
	}

	return _BuildSpitFromDynamo(resp.Attributes, nil)
}

func (p *awsDynamoDBStorager) GetRaw(tableName string, keyName string, keyValue string, o interface{}) error {
	params := &dynamodb.QueryInput{
		KeyConditionExpression: aws.String("#idName = :idVal"),
		ExpressionAttributeNames: map[string]*string{
			"#idName": &keyName,
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":idVal": {
				S: aws.String(keyValue),
			},
		},
		TableName: aws.String(tableName),
	}
	resp, err := p.svc.Query(params)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and Message from an error.
		log.Println("dynamo_adapter::GetRaw::", err.Error())
		return err
	}
	if len(resp.Items) == 0 {
		return DynamoDbItemNotFoundError{fmt.Sprintf("%v:%v:%v", tableName, keyName, keyValue)}
	}
	if err := dynamodbattribute.UnmarshalMap(resp.Items[0], o); err != nil {
		log.Println("dynamo_adapter::GetRaw::", "Error while unmarshalling DynamoDB item: ", err)
		return err
	}
	return nil
}

func (p *awsDynamoDBStorager) FAI(tableName string, keyName string, keyValue string, valueName string, diff int) (int, error) {
	params := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{ // Required
			keyName: {
				S: aws.String(keyValue),
			},
		},
		AttributeUpdates: map[string]*dynamodb.AttributeValueUpdate{
			valueName: {
				Action: aws.String("ADD"),
				Value: &dynamodb.AttributeValue{
					N: aws.String(strconv.Itoa(diff)),
				},
			},
		},
		ReturnValues: aws.String("ALL_NEW"),
		TableName:    aws.String(tableName),
	}
	resp, err := p.svc.UpdateItem(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and Message from an error.
		log.Println("dynamo_adapter::FAI::", err.Error())
		return 0, err
	}

	defaultValue := 0
	if err := dynamodbattribute.Unmarshal(resp.Attributes[valueName], &defaultValue); err != nil {
		log.Println("dynamo_adapter::FAI::", "Error while unmarshalling DynamoDB item: ", err)
		return 0, nil
	}
	return defaultValue, nil
}

// NextId() generates the next unique ID to be used as id
func (p *awsDynamoDBStorager) NextId() (string, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	cntTotal := 3
	cntInc := r.Intn(cntTotal) + 1
	nextId := ""
	for i := 1; i <= cntTotal; i++ {
		diff := 0
		if cntInc == i {
			diff = 1
		}
		// increase the counter selected randomly only
		cntCurrent, err := p.FAI(_TABLE_NAME_SPITS_META, "key", _SPIT_CNT_PREFIX+strconv.Itoa(i), "value", diff)
		if err != nil {
			return "-_-INVALID-_-", err
		}
		nextId += SpitIdEncoding.Encode(uint64(cntCurrent))

	}
	return nextId, nil
}
