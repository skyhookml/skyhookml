FROM ubuntu:18.04

RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections

RUN apt-get update && \
	apt-get dist-upgrade -y && \
	apt-get install -y build-essential git python3-dev wget curl software-properties-common libmetis-dev ffmpeg unzip && \
	add-apt-repository -y ppa:longsleep/golang-backports && \
	apt-get update && \
	apt-get install -y golang-1.13-go
RUN curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py && \
	python3 get-pip.py && \
	pip3 install scikit-image scikit-video numpy requests Pillow opencv-python PyYAML matplotlib

# golang and youtube-dl dependencies
RUN ln -s /usr/lib/go-1.13/bin/go /usr/bin/go && \
	go get github.com/google/uuid && \
	go get github.com/googollee/go-socket.io && \
	go get github.com/mattn/go-sqlite3 && \
	go get github.com/mitroadmaps/gomapinfer/common && \
	go get github.com/sasha-s/go-deadlock && \
	go get golang.org/x/image/font && \
	go get golang.org/x/image/font/basicfont && \
	go get golang.org/x/image/math/fixed && \
	curl -L https://yt-dl.org/downloads/latest/youtube-dl -o /usr/local/bin/youtube-dl
