#!/bin/bash

docker build -t skyhookml/base -f docker/base/Dockerfile . &&
docker build -t skyhookml/basic -f docker/basic/Dockerfile . &&
docker build -t skyhookml/pytorch -f docker/pytorch/Dockerfile . &&
docker build -t skyhookml/allinone -f docker/allinone/Dockerfile .
