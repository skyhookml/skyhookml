#!/bin/bash
./main --url http://localhost:8080 --initdb --worker http://localhost:8081 &
PIDS[0]=$!
./worker localhost 8081 process &
PIDS[1]=$!
trap "kill ${PIDS[*]}" SIGINT
wait
