# Docker Registry Event Collector, or DREC

## Overview

I hope this image conveys the message somewhat clearly.

![](https://raw.githubusercontent.com/kwk/docker-registry-event-collector/master/images/docker-registry-event-collector-overview.png)

# Compile and run

These instructions will guide you through compiling this project to create a
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

I'm working on some test but currently there a none. Feel free to clone this
repo and create a pull request for tests. I really appreciate any contribution.

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
    -dbPort=27017: mongo db host
    -dbUser="": mongo db username
    -listenOnIp="0.0.0.0": On which IP to listen for notifications from a docker registry
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

  * The DREC currently only handles `push` and `pull` events. `delete` still
    needs to be implemented.
  * When run from a docker container, the executable doesn't accept CLI flags.
