#!/bin/bash
set -m
./main --url http://127.0.0.1:8080 --initdb &
sleep 5
./worker localhost 8081 http://localhost:8080 process
