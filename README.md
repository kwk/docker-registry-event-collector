[![Build Status](https://travis-ci.org/kwk/docker-registry-event-collector.svg?branch=master)](https://travis-ci.org/kwk/docker-registry-event-collector)

# Docker Registry Event Collector, or DREC

## Synopsis

### Overview

Here's an overview of the architecture:

![](https://raw.githubusercontent.com/kwk/docker-registry-event-collector/master/images/docker-registry-event-collector-overview.png)

### History

Docker is a fantastic tool, there's no doubt about it. One of the main reasons
for it's success is that is provides a central place from which everybody can
grab pre-built images for famous tools or Linux distribution. This place is
called the [Docker Hub](https://hub.docker.com).

Besides the official Hub the Docker team gives everybody the tools to host their
own hub, aka *registry*. They used to have a registry written in python that has
been around for quite some time. It is often referred to as registry v1 because
it was the first incarnation of the registry (actually, the latest version ever
released was `0.9.1`).

With the advent of the [docker/distribution](http://github.com/docker/distribution)
project, the registry v1 must be considered deprecated. The docker/distribution
project also contains a registry component that is written in Go, just like the
rest of Docker. I read somewhere that the intention is to share more code
between all the different projects and to have the Docker team members be able
to work in a common programming language: Go. This registry component is often
referred to as registry v2.

The HTTP-API to query a registry has changed a lot from v1 to
[v2](https://github.com/docker/distribution/blob/master/docs/spec/api.md).
Although v2 responses contain some of the information that existed in v1
responses, this is only in the for compatibility with older Docker client
versions.

As an alternative to the compatibility information, the docker registry v2 has
introduced the concept of [registry event notifications to HTTP endpoints](https://github.com/docker/distribution/blob/master/docs/notifications.md).

Let me shamelessly copy some relevant information for you:

> The Registry supports sending webhook notifications in response to events
> happening within the registry. Notifications are sent in response to manifest
> pushes and pulls and layer pushes and pulls. These actions are serialized into
> events. The events are queued into a registry-internal broadcast system which
> queues and dispatches events to Endpoints.
> [...]
> Notifications are sent to endpoints via HTTP requests.

### Goal of this project and architecture

The docker registry event collector listens for events coming from a docker
registry v2. Upon receiving an event, it inspects the event and inserts a
statistics entry for the specific repository into a Mongo database. If there
already is an entry for a repository, it will be updated (e.g. number of pushs
or pulls will be incremented). All insert and update operations are performed
atomically.

The Mongo database then becomes a storage for extra information on a repository
that isn't available from the pure registry v2 API. The information stored give
answers to these questions:

 * How often has a repository been pushed or pulled?
 * When was it first pulled or pushed?
 * What actors contributed to it?

### Future

In addition to this information, there's room for more. One can for instance
think of keeping track of how many stars a repository has by simply adding
a `numstars` field to each repository document in the Mongo database. A frontend
then can simply increment this entry using a Mongo update call:

    $ db.registry-events.update({"repositoryname": "yourrepo"}, $inc: {"numstars": 1}})

If a frontend is written with [Meteor](https://www.meteor.com/), then the
changes to the database would be immediately reflected in all clients that have
a subscription on the `numstars` field.

# Build the tool on your own

These instructions walk you through compiling this project to create a
single *standalone* binary that you can copy and run almost wherever you want.

## Check go version

Make sure you have at least `go` version `1.4.2`. The version that ships with
current LTS Ubuntu (14.04) is too old to compile the code.

To check your current `go` version:

    $ go version
    go version go1.4.2 linux/amd64

## Get the source code

I highly encourage you to follow the steps I present here to make the
compilation experience as smooth as possible. If you're new to Go you might
find it silly but Go has some good reasons for operating this way, trust me.

    $ mkdir -p ~/gopath/src/github.com/kwk
    $ export GOPATH=~/gopath
    $ cd ~/gopath/src/github.com/kwk
    $ git clone https://github.com/kwk/docker-registry-event-collector.git
    $ cd docker-registry-event-collector

## Download all dependencies

This will download all dependencies specified in the source code's `import`
sections. It might take some time to finish. Again, if you're new to Go you
might wonder if there's no `Makefile` or something. In fact, there is none!

    $ go get

## Tests

The translation from a JSON notification Event to an upsert (aka. update or
insert) MongoDB command is tested for manifest events. I also test that
layer push events are not considered to be updates and don't result in any
MongoDB commands.
I really appreciate any contribution.

## Build

To build the executable run this command. It should take no more than a few
seconds.

    $ go build

## Run

To run the executable and see the options with which you can configure it do:

    $ ./docker-registry-event-collector -h
    Usage of ./docker-registry-event-collector:
      -certKeyPath="certs/domain.key": Path to SSL certificate key
      -certPath="certs/domain.crt": Path to SSL certfificate file
      -dbHost="127.0.0.1": mongo db host
      -dbName="docker-registry-db": mongo database name
      -dbPassword="": mongo db password
      -dbStatsCollectionName="repository-stats": mongo database collection name
      -dbUser="": mongo db username
      -dpPort=27017: mongo db host
      -listenOnIP="0.0.0.0": On which IP to listen for notifications from a docker registry
      -listenOnPort=10443: On which port to listen for notifications from a docker registry
      -route="/events": HTTP route at which docker-registry events are accepted (must start with "/")

### Note about certificates

Please note, that the DREC only works with HTTPS and so you must specify a
certificate and a key file. There are default files you can use for testing but
you should definitively create your own files.

# Docker image

I've setup a job to build an publish docker images for this project at the
[Docker Hub](https://hub.docker.com/r/konradkleine/docker-registry-event-collector/).

# Known issues

  * ~~The DREC currently only handles `push` and `pull` events. `delete` still
    needs to be implemented.~~ (Update: delete is translated into a MongoDB remove).
  * When run from a docker container, the executable doesn't accept CLI flags.
