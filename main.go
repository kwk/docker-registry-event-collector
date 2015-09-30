package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/docker/distribution/notifications"
	"github.com/kwk/docker-registry-event-collector/settings"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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

	ctx := &appContext{session: nil}

	conf, err := ctx.conf.CreateFromCommandLineFlags()
	if err != nil {
		panic(err)
	}
	conf.Print()
	ctx.conf = conf

	// Connect to DB
	mongoConnStr := conf.GetMongoDBConnectionString()

	log.Printf("About to connect to MongoDB on \"%s\".", mongoConnStr)
	session, err := mgo.Dial(mongoConnStr)
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
	c := ctx.session.DB(conf.DbName).C("registry-events")

	// The repository structure shall have a uniqe key on the repository's
	// name field
	index := mgo.Index{
		Key:        []string{"repositoryname"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}

	err = c.EnsureIndex(index)
	if err != nil {
		log.Print("It looks like your mongo database in an incosinstent status. ",
			"Make sure you have no duplicate entries for repository names.")
		panic(err)
	}

	//http.HandleFunc(conf.EndpointRoute, handler)

	var httpConnectionString = conf.GetEndpointConnectionString()
	log.Printf("About to listen on \"%s%s\".", httpConnectionString, conf.EndpointRoute)

	mux := http.NewServeMux()
	appHandler := &appHandler{ctx: ctx}
	//mux.Handle(conf.EndpointRoute, appHandler)
	mux.Handle("/events", appHandler)

	log.Print(appHandler)

	err = http.ListenAndServeTLS(httpConnectionString, conf.EndpointCertPath, conf.EndpointCertKeyPath, mux)
	//err = http.ListenAndServeTLS(httpConnectionString, conf.EndpointCertPath, conf.EndpointCertKeyPath, mux)
	if err != nil {
		log.Fatal(err)
	}
}

type appHandler struct {
	ctx *appContext
}

// Our ServeHTTP method is mostly the same, and also has the ability to
// access our *appContext's fields (session, config, etc.) as well.
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

	// Try to decode body as Docker notification event
	decoder := json.NewDecoder(req.Body)
	var event notifications.Event
	err := decoder.Decode(&event)
	if err != nil {
		http.Error(w, fmt.Sprintf("Request couldn't be decoded. Error: %s\n", err), http.StatusBadRequest)
		return
	}

	// Now we can assume that "event" is a valid Docker notification event.
	// Let's try to insert into Mongo first. If that fails due to a unique key
	// constraint. Try to update an entry in Mongo.
	// NOTE: We cannot use upsert because we need to *increment* some fields in
	// case of an update and *set* some fields in case of an insert.

	maxs := make(bson.M)
	mins := make(bson.M)
	sets := make(bson.M)
	setOnInsert := make(bson.M)

	// When a repository is first created it has no stars yet.
	setOnInsert["numstars"] = 0

	// The time in the future is used to ensure $min and $max Mongo update
	// operators work. It's a bit of a hack but it was the simplest solution to
	// the problem.
	timeInFuture := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)
	timeInPast := time.Unix(0, 0)

	pullIncrement := 0
	if event.Action == notifications.EventActionPull {
		pullIncrement = 1
		maxs["lastpulled"] = event.Timestamp
		mins["firstpulled"] = event.Timestamp
		setOnInsert["firstpushed"] = timeInFuture
		setOnInsert["lastpushed"] = timeInPast
	}

	pushIncrement := 0
	if event.Action == notifications.EventActionPush {
		pushIncrement = 1
		maxs["lastpushed"] = event.Timestamp
		mins["firstpushed"] = event.Timestamp
		setOnInsert["firstpulled"] = timeInFuture
		setOnInsert["lastpulled"] = timeInPast
	}

	// TODO: handle "delete" action

	// We want something like this
	// db.registry-events.update({"repositoryname": "hallo"}, {$set: {"repositoryname": "hallo"}, $inc: {"numpulls": 1}}, {"upsert": true})

	sets["repositoryname"] = event.Target.Repository

	c := ah.ctx.session.DB(ah.ctx.conf.DbName).C("registry-events")

	var changeInfo *mgo.ChangeInfo

	selector := bson.M{"repositoryname": event.Target.Repository, "$isolated": true}
	update := bson.M{
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
	changeInfo, err = c.Upsert(selector, update)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to store stats. Error: %s\n", err), http.StatusBadGateway)
		return
	}

	log.Print(changeInfo)
}
