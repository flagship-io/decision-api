package assignments_managers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

type assignmentsTimeStamp struct {
	assignments map[string]string
	timestamp   int64
}

// Define a mock struct to be used in your unit tests of myFunc.
type mockDynamoDBClient struct {
	dynamodbiface.DynamoDBAPI
	assignments map[string]*assignmentsTimeStamp
}

var lock = sync.Mutex{}

func (m *mockDynamoDBClient) GetItemWithContext(ctx context.Context, input *dynamodb.GetItemInput, options ...request.Option) (*dynamodb.GetItemOutput, error) {
	var campaigns map[string]string = nil
	for k, v := range m.assignments {
		if k == *input.Key["id"].S {
			campaigns = v.assignments
			campaigns["d"] = fmt.Sprint(v.timestamp)
		}
	}

	var item map[string]*dynamodb.AttributeValue = nil
	if campaigns != nil {
		item, _ = dynamodbattribute.MarshalMap(campaigns)
	}

	return &dynamodb.GetItemOutput{
		Item: item,
	}, nil
}

func (m *mockDynamoDBClient) UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	visitorID := *input.Key["id"].S
	values := strings.Split(strings.Replace(*input.UpdateExpression, "SET ", "", 1), ",")

	newAssignments := make(map[string]string)
	var time int64 = 0
	for _, v := range values {
		keyValue := strings.Split(v, " = ")
		if input.ExpressionAttributeValues[keyValue[1]].S != nil {
			newAssignments[strings.Trim(keyValue[0], " ")] = *input.ExpressionAttributeValues[keyValue[1]].S
		}
		if input.ExpressionAttributeValues[keyValue[1]].N != nil {
			time, _ = strconv.ParseInt(*input.ExpressionAttributeValues[keyValue[1]].N, 10, 64)
		}
	}
	lock.Lock()
	if m.assignments == nil {
		m.assignments = make(map[string]*assignmentsTimeStamp)
	}
	m.assignments[visitorID] = &assignmentsTimeStamp{
		assignments: newAssignments,
		timestamp:   time,
	}
	lock.Unlock()
	return &dynamodb.UpdateItemOutput{}, nil
}
