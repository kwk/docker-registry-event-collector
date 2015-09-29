package main

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

// RepositoryStats is the layout for how an entry is stored in MongoDB.
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
