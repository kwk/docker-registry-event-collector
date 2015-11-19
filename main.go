package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/docker/distribution/notifications"
	"github.com/kwk/docker-registry-event-collector/events"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"flag"
	"github.com/golang/glog"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"
)

// AppContext holds all relevant options needed in various places of the app
type AppContext struct {
	Config  *Config
	Session *mgo.Session
	c       string
}

// NewAppContext creates an empty application context object
func NewAppContext() (*AppContext, error) {
	return &AppContext{Session: nil}, nil
}

// The main function sets up the connection to the storage backend for
// aggregated events (e.g. MongoDB) and fires up an HTTPs server which acts as
// an endpoint for docker notifications.
func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	glog.CopyStandardLogTo("INFO")

	// Create our application context
	ctx, _ := NewAppContext()

	// Load config file given by first argument
	configFilePath := flag.Arg(0)
	if configFilePath == "" {
		glog.Exit("Config file not specified")
	}
	c, err := LoadConfig(configFilePath)
	if err != nil {
		glog.Exit(err)
	}
	ctx.Config = c

	// Connect to MongoDB
	// Read in the password (if any)
	if ctx.Config.DialInfo.PasswordFile != "" {
		passBuf, err := ioutil.ReadFile(ctx.Config.DialInfo.PasswordFile)
		if err != nil {
			glog.Exitf(`Failed to read password file "%s": %s`, ctx.Config.DialInfo.PasswordFile, err)
		}
		ctx.Config.DialInfo.DialInfo.Password = strings.TrimSpace(string(passBuf))
	}

	glog.V(2).Infof("Creating MongoDB session (operation timeout %s)", ctx.Config.DialInfo.DialInfo.Timeout)
	session, err := mgo.DialWithInfo(&ctx.Config.DialInfo.DialInfo)
	if err != nil {
		glog.Exit(err)
	}
	defer session.Close()
	ctx.Session = session

	// Wait for errors on inserts and updates and for flushing changes to disk
	session.SetSafe(&mgo.Safe{FSync: true})

	collection := ctx.Session.DB(ctx.Config.DialInfo.DialInfo.Database).C(ctx.Config.Collection)

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
		glog.Exitf("It looks like your mongo database is incosinstent. ",
			"Make sure you have no duplicate entries for repository names.")
	}

	// Setup HTTP endpoint
	var httpConnectionString = ctx.Config.GetEndpointConnectionString()
	glog.Infof("About to listen on \"%s%s\".", httpConnectionString, ctx.Config.Server.Route)

	mux := http.NewServeMux()
	appHandler := &appHandler{ctx: ctx}
	mux.Handle(ctx.Config.Server.Route, appHandler)
	err = http.ListenAndServeTLS(httpConnectionString, ctx.Config.Server.Ssl.Cert, ctx.Config.Server.Ssl.CertKey, mux)
	if err != nil {
		glog.Exit(err)
	}

	glog.Info("Exiting.")
}

type appHandler struct {
	ctx *AppContext
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
	collection = ah.ctx.Session.DB(ah.ctx.Config.DialInfo.DialInfo.Database).C(ah.ctx.Config.Collection)

	for index, event := range envelope.Events {
		glog.V(2).Infof("Processing event %d of %d\n", index+1, len(envelope.Events))

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
