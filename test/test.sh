#!/bin/bash

curl -vvv -X POST -d "@event.push.simple.json" -H "Content-Type: application/vnd.docker.distribution.events.v1+json" -k https://127.0.0.1:10443/events
