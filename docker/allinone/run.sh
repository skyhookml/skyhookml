#!/bin/bash
docker start -i skyhookml ||
docker run --mount "src=$(pwd)/data,target=/usr/src/app/skyhook/data,type=bind" --gpus all -p 8080:8080 --name skyhookml --shm-size 1G skyhookml/allinone
