package main

import (
	"time"

	"github.com/kwk/docker-registry-event-collector/Godeps/_workspace/src/gopkg.in/mgo.v2/bson"
)

// RepositoryStats is the layout for how an entry is stored in MongoDB.
// "omitempty" means: only include the field if it's not set to the zero value
// for the type or to empty slices or maps.
type RepositoryStats struct {
	ID             bson.ObjectId `bson:"_id,omitempty"`
	RepositoryName string        `json:"repositoryname,string"`
	LastPushed     time.Time     `json:"lastpushed,omitempty"`
	LastPulled     time.Time     `json:"lastpulled,omitempty"`
	FirstPushed    time.Time     `json:"firstpushed,omitempty"`
	FirstPulled    time.Time     `json:"firstpulled,omitempty"`
	NumPulls       uint          `json:"numpulls,number"`
	NumPushs       uint          `json:"numpushs,number"`
	NumStars       uint          `json:"numstars,number"`
	Actors         []string      `json:"actors,omitempty"`
}
