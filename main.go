package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/docker/distribution/notifications"
	"github.com/kwk/docker-magpie/settings"
	"gopkg.in/mgo.v2"
)

var (
	conf    settings.Settings
	session *mgo.Session
	c       string
)

// The main function sets up the connection to the storage backend for
// aggregated events (e.g. MongoDB) and fires up an HTTPs server which acts as
// an endpoint for docker notifications. In other words, the HTTPs server
// accepts connections on
func main() {

	conf, err := conf.CreateFromCommandLineFlags()
	if err != nil {
		panic(err)
	}
	conf.Print()

	// Connect to DB
	mongoConnStr := conf.GetMongoDBConnectionString()

	log.Printf("About to connect to MongoDB on %s", mongoConnStr)
	session, err := mgo.Dial(mongoConnStr)
	if err != nil {
		panic(err)
	}
	defer session.Close()
	//session.SetMode(mgo.Monotonic. true)
	// err = session.DB("test").DropDatabase()
	// if (err != nil) {
	//   panic(err)
	// }

	// Collection people
	c := session.DB("test").C("people")

	// Index
	index := mgo.Index{
		Key:        []string{"RepositoryName"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}

	err = c.EnsureIndex(index)
	if err != nil {
		panic(err)
	}

	http.HandleFunc(conf.EndpointRoute, handler)

	var httpConnectionString = conf.GetEndpointConnectionString()
	log.Printf("About to listen ond %s", httpConnectionString)
	err = http.ListenAndServeTLS(httpConnectionString, conf.EndpointCertPath, conf.EndpointCertKeyPath, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, req *http.Request) {
	// A request needs to be made via POST and must have a body
	if req.Method != "POST" || req.Body == nil {
		w.WriteHeader(400)
		return
	}

	// Test for correct mimetype and reject all others
	if req.Header.Get("Content-Type") != notifications.EventsMediaType {
		w.WriteHeader(400)
	}

	// Try to decode body as Docker event
	decoder := json.NewDecoder(req.Body)
	var event notifications.Event
	err := decoder.Decode(&event)
	if err != nil {
		w.WriteHeader(400)
		log.Printf("Error: %s", err)
		return
	}

	log.Println(event.Action)
	log.Println(event.Actor.Name)
	log.Println(event.Target.Repository)
	log.Println(event.Timestamp)

	c := session.DB("test").C("people")
	err = c.Insert(&RepositoryStats{
		RepositoryName: event.Target.Repository,
		LastPushed:     event.Timestamp,
		FirstPushed:    event.Timestamp,
		NumPulls:       0,
		NumPushs:       1,
		NumStars:       0,
	})

	if err != nil {
		// TODO: Maybe it's better to return HTTP 500 Error code rather than panic
		panic(err)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("\nThis is an example server.\n"))

}

// RepositoryStats holds some statistics about a repository.
type RepositoryStats struct {
	RepositoryName string
	LastPushed     time.Time
	LastPulled     time.Time
	FirstPushed    time.Time
	FirstPulled    time.Time
	NumPulls       uint
	NumPushs       uint
	//Actors []string
	NumStars uint
}
