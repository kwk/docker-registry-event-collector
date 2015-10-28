package events

import (
	"reflect"
	"strings"
	"time"

	"github.com/kwk/docker-registry-event-collector/Godeps/_workspace/src/gopkg.in/mgo.v2/bson"

	"encoding/json"
	"testing"

	"github.com/kwk/docker-registry-event-collector/Godeps/_workspace/src/github.com/docker/distribution/notifications"
)

// TestPushEventProcessing tests how a valid docker registry push event when
// processed will create an upsert (aka. update or insert) map for MongoDB to
// consume.
// TODO: (kwk) Create negative test: Decode an obviously wrong JSON strin
// into an notifications.Event and report error on success.
func TestManifestPushEventProcessing(t *testing.T) {
	eventStr := strings.TrimSpace(`{
	}`)

	// Decode JSON string into a notifications.Event structure
	var event notifications.Event
	decoder := json.NewDecoder(strings.NewReader(eventStr))
	err := decoder.Decode(&event)
	if err != nil {
		t.Fatalf("Failed to decode event: %s\n", err)
	}

	// Let's get the actual result
	actualResult, err := ProcessEventPullOrPush(&event)
	if err != nil {
		t.Fatalf("Failed to get the actual result: %s\n", err)
	}

	// Let's build the expected result
	timeFormat := "2006-01-02 15:04:05 +0000 UTC"
	firstPushedTime, err := time.Parse(timeFormat, "2006-01-02 15:04:05 +0000 UTC")
	if err != nil {
		t.Fatalf("Failed parse time: %s\n", err)
	}
	firstPulledTime, err := time.Parse(timeFormat, "2099-01-01 00:00:00 +0000 UTC")
	if err != nil {
		t.Fatalf("Failed parse time: %s\n", err)
	}
	lastPushedTime, err := time.Parse(timeFormat, "2006-01-02 15:04:05 +0000 UTC")
	if err != nil {
		t.Fatalf("Failed parse time: %s\n", err)
	}
	lastPulledTime, err := time.Parse(timeFormat, "1970-01-01 00:00:00 +0000 UTC")
	if err != nil {
		t.Fatalf("Failed parse time: %s\n", err)
	}

	expectedResult := bson.M{
		"$min":      bson.M{"firstpushed": firstPushedTime},
		"$addToSet": bson.M{"actors": "test-actor"},
		"$inc": bson.M{
			"numpushs": 1,
			"numpulls": 0,
		},
		"$set": bson.M{"repositoryname": "library/test"},
		"$setOnInsert": bson.M{
			"numstars":    0,
			"firstpulled": firstPulledTime,
			"lastpulled":  lastPulledTime,
		},
		"$max": bson.M{"lastpushed": lastPushedTime},
	}

	// Compare if actual and expected results are equal and if not, print what
	// we expected as JSON for the reader to spot the errors.
	if !reflect.DeepEqual(actualResult, expectedResult) {
		t.Fail() // Mark test as failed but continue execution

		t.Logf("Result map doesn't match expected map.\n")

		// Print expected result as JSON
		expectedResultJSON, err := json.Marshal(expectedResult)
		if err != nil {
			t.Fatalf("Failed to marshal expected result to JSON: %s\n", err)
		}
		t.Logf("Expected result: \n\n%s\n\n", expectedResultJSON)

		// Print actual result as JSON
		actualResultJSON, err := json.Marshal(actualResult)
		if err != nil {
			t.Fatalf("Failed to marshal actual result to JSON: %s\n", err)
		}
		t.Logf("Actual result: \n\n%s\n\n", actualResultJSON)
		t.FailNow()
	}
}

func TestLayerPushEvent(t *testing.T) {
	eventStr := strings.TrimSpace(`{
		"id": "asdf-asdf-asdf-asdf-1",
		"timestamp": "2006-01-02T15:04:05Z",
		"action": "push",
		"target": {
			"mediaType": "application/vnd.docker.container.image.rootfs.diff+x-gtar",
			"size": 2,
			"digest": "tarsum.v2+sha256:0123456789abcdef1",
			"length": 2,
			"repository": "library/test",
			"url": "http://example.com/v2/library/test/manifests/latest"
		},
		"request": {
			"id": "asdfasdf",
			"addr": "client.local",
			"host": "registrycluster.local",
			"method": "PUT",
			"useragent": "test/0.1"
		},
		"actor": {
			"name": "test-actor"
		},
		"source": {
			"addr": "hostname.local:port"
		}
	}`)

	// Decode JSON string into a notifications.Event structure
	var event notifications.Event
	decoder := json.NewDecoder(strings.NewReader(eventStr))
	err := decoder.Decode(&event)
	if err != nil {
		t.Fatalf("Failed to decode event: %s\n", err)
	}

	// Let's get the actual result
	actualResult, err := ProcessEventPullOrPush(&event)
	if err == nil { // NOTE: We expect this to fail and raise an error when it doesn't
		t.Fatalf("A layer push event must not result in any DB update: %s\n", err)
	}

	// Check that the update command is empty
	if len(actualResult) != 0 {
		t.Fatalf("Expected update string must be empty: %s\n", actualResult)
	}
}
