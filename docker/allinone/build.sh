#!/bin/bash

docker build -t skyhookml/base -f docker/base/Dockerfile . &&
docker build -t skyhookml/allinone -f docker/allinone/Dockerfile .
