package spit

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/lambrospetrou/spito/ids"
	"github.com/lambrospetrou/spito/utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

///////////////////////////////////////////////////////////////////////

const (
	// TODO read the following parameters from the environment of the application
	_TABLE_NAME_SPITS_DATA     = "SpitsData"
	_TABLE_NAME_SPITS_META     = "SpitsMeta"
	_SPIT_ID_CNT_TOTAL     int = 4

	_SPIT_ID_CNT_PREFIX   string = "spit::cnt::"
	_SPIT_ID_CHARS_PREFIX string = "spit::chars::"
	_SPIT_KEY_PREFIX      string = "spit::id::"
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

type _SpitIdCharModel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

////////////////////////////////////////////////////////////////////////

// init() will try to fetch the sequence generators to be used when encoding the ids.
// If no sequence exists in the database new ones will be created.
func (p *awsDynamoDBStorager) init() {
	log.Println("dynamo_adapter::init()")

	// Create the key generators
	// 62 valid characters A-Za-z0-9
	base62Chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	chars := make([]string, 0)
	for i := 0; i < _SPIT_ID_CNT_TOTAL; i++ {
		chars = append(chars, utils.ShuffleString(base62Chars))
	}
	log.Println("Will try to use: ", chars)

	finalChars := make([]string, 0)

	// Store the generators in Dynamo if someone else did not!
	for i := 0; i < _SPIT_ID_CNT_TOTAL; i++ {
		key := _SPIT_ID_CHARS_PREFIX + strconv.Itoa(i+1)
		charsNew := chars[i]
		item, _ := dynamodbattribute.Marshal(&_SpitIdCharModel{key, charsNew})

		log.Println("Trying item:", item)
		params := &dynamodb.PutItemInput{
			Item:                item.M,
			TableName:           aws.String(_TABLE_NAME_SPITS_META),
			ConditionExpression: aws.String("attribute_not_exists(#idName)"),
			ExpressionAttributeNames: map[string]*string{
				"#idName": aws.String("key"),
			},
			ReturnValues: aws.String("ALL_OLD"),
		}
		resp, err := p.svc.PutItem(params)
		if err == nil {
			// Added the new character sequence
			finalChars = append(finalChars, charsNew)
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			log.Println("dynamo_adapter::init::", err.Error(), resp)

			// Try to get the Item since we failed to put it
			charExisting := &_SpitIdCharModel{}
			err = p.GetRaw(_TABLE_NAME_SPITS_META, "key", key, charExisting)
			if err != nil {
				log.Fatalln("Failed to get the existing ID Char generators from DB.")
				return
			}
			finalChars = append(finalChars, charExisting.Value)
		}
	}
	log.Println("Final sequence generators used: ", finalChars)
	// Initialize the ID generator
	ids.InitWith(finalChars...)
}

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
			log.Println("dynamo_adapter::Get::", err.Error(), resp)
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
	cntTotal := _SPIT_ID_CNT_TOTAL
	cntInc := r.Intn(cntTotal) + 1
	nextId := ""
	for i := 1; i <= cntTotal; i++ {
		diff := 0
		if cntInc == i {
			diff = 1
		}
		// increase the counter selected randomly only
		cntCurrent, err := p.FAI(_TABLE_NAME_SPITS_META, "key", _SPIT_ID_CNT_PREFIX+strconv.Itoa(i), "value", diff)
		if err != nil {
			return "-_-INVALID-_-", err
		}
		//nextId += SpitIdEncoding.Encode(uint64(cntCurrent))
		nextId += ids.Encode(uint64(cntCurrent), i-1)
	}
	return nextId, nil
}
