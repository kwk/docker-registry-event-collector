package events

import (
	"fmt"
	"time"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/notifications"
	"gopkg.in/mgo.v2/bson"
)

// ProcessEventPullOrPush returns a key->value map (bson.M) that can be
// used to upsert (aka. insert or update) a statistics document about a
// repository in a Mongo database.
func ProcessEventPullOrPush(event *notifications.Event) (bson.M, error) {

	// Check that it's a pull or push event
	if event.Action != notifications.EventActionPush && event.Action != notifications.EventActionPull {
		return nil, fmt.Errorf("Wrong event.Action: %s", event.Action)
	}

	// Check that the mediatype equals: application/vnd.docker.distribution.manifest.v1+json
	// TODO: (kwk) The documentation says it can also be application/json
	// (see https://github.com/docker/distribution/blob/master/manifest/schema1/manifest.go#L15)
	if event.Target.MediaType != schema1.ManifestMediaType {
		return nil, fmt.Errorf("Wrong event.Target.MediaType: \"%s\". Expected: \"%s\"", event.Target.MediaType, schema1.ManifestMediaType)
	}

	maxs := make(bson.M)
	mins := make(bson.M)
	sets := make(bson.M)
	setOnInsert := make(bson.M)

	setOnInsert["numstars"] = 0

	timeInFuture := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)
	timeInPast := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)

	pullIncrement := 0
	pushIncrement := 0

	if event.Action == notifications.EventActionPull {
		pullIncrement = 1
		maxs["lastpulled"] = event.Timestamp
		mins["firstpulled"] = event.Timestamp
		setOnInsert["firstpushed"] = timeInFuture
		setOnInsert["lastpushed"] = timeInPast
	}

	if event.Action == notifications.EventActionPush {
		pushIncrement = 1
		maxs["lastpushed"] = event.Timestamp
		mins["firstpushed"] = event.Timestamp
		setOnInsert["firstpulled"] = timeInFuture
		setOnInsert["lastpulled"] = timeInPast
	}

	sets["repositoryname"] = event.Target.Repository

	updateBson := bson.M{
		"$set":         sets,
		"$setOnInsert": setOnInsert,
		"$max":         maxs,
		"$min":         mins,
		"$addToSet":    bson.M{"actors": event.Actor.Name},
		"$inc": bson.M{
			"numpushs": pushIncrement,
			"numpulls": pullIncrement,
		},
	}

	return updateBson, nil
}
