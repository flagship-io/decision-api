package assignments_managers

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	common "github.com/flagship-io/flagship-common"
)

type DynamoManagerOptions struct {
	Client              dynamodbiface.DynamoDBAPI
	TableName           string
	PrimaryKeySeparator string
	PrimaryKeyField     string
	GetItemTimeout      time.Duration
	LogLevel            string
	LogFormat           logger.LogFormat
}

type DynamoManager struct {
	options DynamoManagerOptions
	logger  *logger.Logger
}

func InitDynamoManager(options DynamoManagerOptions) *DynamoManager {
	logger := logger.New(options.LogLevel, options.LogFormat, "dynamo")
	return &DynamoManager{
		options: options,
		logger:  logger,
	}
}

func (d *DynamoManager) getPrimaryKey(envID string, visitorID string) string {
	return envID + d.options.PrimaryKeySeparator + visitorID
}

func (d *DynamoManager) getItemWithTimeout(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.options.GetItemTimeout)
	defer cancel()

	return d.options.Client.GetItemWithContext(ctx, input)
}

func (d *DynamoManager) getCampaignsAssignment(id string) (map[string]string, error) {
	d.logger.Infof("getCampaignsAssignment for id %s: not found, querying dynamodb\n", id)
	input := &dynamodb.GetItemInput{
		TableName: aws.String(d.options.TableName),
		Key: map[string]*dynamodb.AttributeValue{
			d.options.PrimaryKeyField: {
				S: aws.String(id),
			},
		},
	}

	// Retrieve the item from DynamoDB. If no matching item is found
	// return nil.
	result, err := d.getItemWithTimeout(input)
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, nil
	}

	res := make(map[string]string)
	err = dynamodbattribute.UnmarshalMap(result.Item, &res)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func buildVGAssigns(assigns map[string]string) (res map[string]*common.VisitorCache, assignTime int64) {
	res = make(map[string]*common.VisitorCache)
	for vgID, vID := range assigns {
		// Check if field is date
		if vgID == "d" {
			timestamp, err := strconv.ParseInt(vID, 10, 64)
			if err != nil {
				log.Printf("Error when casting timestamp : %v", err)
			} else {
				assignTime = timestamp
			}
			continue
		}

		// If not date, then it's a variation group.
		// Split variation ID and activation flag and save to VGCacheItem
		infos := strings.Split(vID, ":")
		res[vgID] = &common.VisitorCache{
			VariationID: infos[0],
			Activated:   len(infos) == 2,
		}
	}
	return res, assignTime
}

// LoadAssignments gets all the visitor cache assignments for a specific env ID and visitor ID
func (d *DynamoManager) LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error) {
	id := d.getPrimaryKey(envID, visitorID)
	cAssigns, err := d.getCampaignsAssignment(id)

	if err != nil {
		return nil, err
	}

	if cAssigns == nil {
		return nil, nil
	}

	assigns, timestamp := buildVGAssigns(cAssigns)
	visAssign := &common.VisitorAssignments{
		Timestamp:   timestamp,
		Assignments: assigns,
	}

	return visAssign, nil
}

func (d *DynamoManager) updateAssignmentItem(id string, vgIDAssignments map[string]*common.VisitorCache, date time.Time) error {
	attrValues := map[string]*dynamodb.AttributeValue{
		":date": {
			N: aws.String(strconv.FormatInt(date.AddDate(0, 6, 0).Unix(), 10)),
		},
	}

	updateSets := []string{"d = :date"}

	for vgID, assign := range vgIDAssignments {
		value := assign.VariationID
		if assign.Activated {
			value += ":1"
		}
		updateSets = append(updateSets, vgID+" = :vID"+assign.VariationID)
		attrValues[":vID"+assign.VariationID] = &dynamodb.AttributeValue{
			S: aws.String(value),
		}
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(d.options.TableName),
		Key: map[string]*dynamodb.AttributeValue{
			d.options.PrimaryKeyField: {
				S: aws.String(id),
			},
		},
		ExpressionAttributeValues: attrValues,
		UpdateExpression:          aws.String("SET " + strings.Join(updateSets, ", ")),
	}

	_, err := d.options.Client.UpdateItem(input)
	return err
}

func (d *DynamoManager) ShouldSaveAssignments(context connectors.SaveAssignmentsContext) bool {
	return true
}

// SaveAssignments saves all visitor new assignments into dynamo table
func (d *DynamoManager) SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time) error {
	id := d.getPrimaryKey(envID, visitorID)
	err := d.updateAssignmentItem(id, vgIDAssignments, date)

	if err != nil {
		d.logger.Errorf("error persisting assignments visitor %s : %s", visitorID, err)
	} else {
		d.logger.Infof("successfully persisted assignments for visitor %s", visitorID)
	}
	return err
}
