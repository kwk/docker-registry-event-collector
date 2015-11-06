package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/kwk/docker-registry-event-collector/Godeps/_workspace/src/github.com/docker/distribution/notifications"
	"github.com/kwk/docker-registry-event-collector/Godeps/_workspace/src/gopkg.in/mgo.v2"
	"github.com/kwk/docker-registry-event-collector/Godeps/_workspace/src/gopkg.in/mgo.v2/bson"
	"github.com/kwk/docker-registry-event-collector/events"
	"github.com/kwk/docker-registry-event-collector/settings"
)

type appContext struct {
	conf    settings.Settings
	session *mgo.Session
	c       string
}

// The main function sets up the connection to the storage backend for
// aggregated events (e.g. MongoDB) and fires up an HTTPs server which acts as
// an endpoint for docker notifications.
func main() {

	// Create our application context
	ctx := &appContext{session: nil}

	// Load config from command line and print it
	conf, err := ctx.conf.CreateFromCommandLineFlags()
	if err != nil {
		panic(err)
	}
	conf.Print()
	ctx.conf = conf

	// Connect to DB
	mongoConnStr := ctx.conf.GetMongoDBConnectionString()
	log.Printf("About to connect to MongoDB on \"%s\".", mongoConnStr)
	session, err := mgo.DialWithInfo(&DialInfo{Addrs: [ctx.conf.DbHost]}) // TODO: Complete here
	if err != nil {
		log.Print(err)
		panic(err)
	}
	ctx.session = session
	defer session.Close()

	// Wait for errors on inserts and updates and for flushing changes to disk
	session.SetSafe(&mgo.Safe{FSync: true})

	//session.SetMode(mgo.Monotonic. true)
	// err = session.DB("test").DropDatabase()
	// if (err != nil) {
	//   panic(err)
	// }

	// TODO: make collection configurable
	collection := ctx.session.DB(ctx.conf.DbName).C(ctx.conf.DbStatsCollectionName)

	// The repository structure shall have a uniqe key on the repository's
	// name field
	index := mgo.Index{
		Key:        []string{"repositoryname"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}

	err = collection.EnsureIndex(index)
	if err != nil {
		log.Print("It looks like your mongo database in an incosinstent status. ",
			"Make sure you have no duplicate entries for repository names.")
		panic(err)
	}

	// Setup HTTP endpoint
	var httpConnectionString = ctx.conf.GetEndpointConnectionString()
	log.Printf("About to listen on \"%s%s\".", httpConnectionString, ctx.conf.EndpointRoute)

	mux := http.NewServeMux()
	appHandler := &appHandler{ctx: ctx}
	mux.Handle(ctx.conf.EndpointRoute, appHandler)
	err = http.ListenAndServeTLS(httpConnectionString, ctx.conf.EndpointCertPath, ctx.conf.EndpointCertKeyPath, mux)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Exiting.")
}

type appHandler struct {
	ctx *appContext
}

// ServeHTTP has the ability to access our *appContext's fields (session,
// config, etc.) as well.
func (ah appHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	// The docker registry sends events to HTTP endpoints and queues them up if
	// the endpoint refuses to accept those events. We are only interested in
	// manifest updates, therefore we ignore all others by answering with an HTTP
	// 200 OK. This should prevent the docker registry from getting too full.

	// A request needs to be made via POST
	if req.Method != "POST" {
		http.Error(w, fmt.Sprintf("Ignoring request. Required method is \"POST\" but got \"%s\".\n", req.Method), http.StatusOK)
		return
	}

	// A request must have a body.
	if req.Body == nil {
		http.Error(w, "Ignoring request. Required non-empty request body.\n", http.StatusOK)
		return
	}

	// Test for correct mimetype and reject all others
	// Even the documentation on docker notfications says that we shouldn't be to
	// picky about the mimetype. But we are and let the caller know this.
	contentType := req.Header.Get("Content-Type")
	if contentType != notifications.EventsMediaType {
		http.Error(w, fmt.Sprintf("Ignoring request. Required mimetype is \"%s\" but got \"%s\"\n", notifications.EventsMediaType, contentType), http.StatusOK)
		return
	}

	// Try to decode HTTP body as Docker notification envelope
	decoder := json.NewDecoder(req.Body)
	var envelope notifications.Envelope
	err := decoder.Decode(&envelope)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode envelope: %s\n", err), http.StatusBadRequest)
		return
	}

	var collection *mgo.Collection
	collection = ah.ctx.session.DB(ah.ctx.conf.DbName).C(ah.ctx.conf.DbStatsCollectionName)

	for index, event := range envelope.Events {
		log.Printf("Processing event %d of %d\n", index+1, len(envelope.Events))

		// Handle all three cases: push, pull, and delete
		if event.Action == notifications.EventActionPull || event.Action == notifications.EventActionPush {

			updateBson, err := events.ProcessEventPullOrPush(&event)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to process push or pull event. Error: %s\n", err), http.StatusBadRequest)
				return
			}
			changeInfo, err := collection.Upsert(bson.M{"repositoryname": event.Target.Repository, "$isolated": true}, updateBson)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to update DB. Error: %s\n", err), http.StatusBadGateway)
				return
			}
			log.Printf("Number of updated documents: %d", changeInfo.Updated)

		} else if event.Action == notifications.EventActionDelete {

			err := collection.Remove(bson.M{"repositoryname": event.Target.Repository})
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to delete document from DB. Error: %s\n", err), http.StatusBadGateway)
				return
			}

		} else {

			http.Error(w, fmt.Sprintf("Invalid event action: %s\n", event.Action), http.StatusBadRequest)
			return

		}

	}

	http.Error(w, fmt.Sprintf("Done\n"), http.StatusOK)
}
