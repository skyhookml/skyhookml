#!/bin/bash
set -m
./main --url http://localhost:8080 --initdb --worker http://localhost:8081 &
./worker localhost 8081 process
