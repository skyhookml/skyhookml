#!/bin/bash
export COORDINATOR_URL="http://localhost:8080"
export WORKER_URL="http://localhost:8081"
export INSTANCE_ID=""
. data/config.sh
./main --url $COORDINATOR_URL --initdb --worker $WORKER_URL --instance-id $INSTANCE_ID
