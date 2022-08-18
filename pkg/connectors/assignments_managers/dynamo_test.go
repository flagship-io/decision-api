package assignments_managers

import (
	"testing"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	decision "github.com/flagship-io/flagship-common"
	"github.com/stretchr/testify/assert"
)

func TestDynamoAssignmentsManager(t *testing.T) {
	envID := "env_id"
	visitorID := "visitor_id"
	mockClient := &mockDynamoDBClient{
		assignments: map[string]*assignmentsTimeStamp{},
	}

	d := &DynamoManager{
		options: DynamoManagerOptions{
			Client:              mockClient,
			TableName:           "testTable",
			PrimaryKeySeparator: ".",
			PrimaryKeyField:     "id",
			GetItemTimeout:      10 * time.Millisecond,
		},
		logger: logger.New("info", logger.FORMAT_TEXT, "dynamodbManager"),
	}

	assignments, err := d.LoadAssignments(envID, visitorID)

	assert.Nil(t, err)
	assert.Nil(t, assignments)

	err = d.SaveAssignments(envID, visitorID, map[string]*decision.VisitorCache{
		"vgID": {
			Activated:   true,
			VariationID: "vID",
		},
	}, time.Now())
	assert.Nil(t, err)

	assignments, err = d.LoadAssignments(envID, visitorID)

	assert.Nil(t, err)
	assert.NotNil(t, assignments)
	assert.Equal(t, time.Now().AddDate(0, 6, 0).Unix(), assignments.Timestamp)
	assert.Equal(t, "vID", assignments.Assignments["vgID"].VariationID)
	assert.Equal(t, true, assignments.Assignments["vgID"].Activated)

	shouldSaveAssignments := d.ShouldSaveAssignments(connectors.SaveAssignmentsContext{
		AssignmentScope: connectors.Decision,
	})
	assert.True(t, shouldSaveAssignments)
	shouldSaveAssignments = d.ShouldSaveAssignments(connectors.SaveAssignmentsContext{
		AssignmentScope: connectors.Activation,
	})
	assert.True(t, shouldSaveAssignments)
}
