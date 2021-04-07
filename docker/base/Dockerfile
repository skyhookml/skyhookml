FROM nvidia/cuda:11.0.3-devel-ubuntu20.04

RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections

RUN apt-get update && \
	apt-get dist-upgrade -y && \
	apt-get install -y build-essential git python3-dev wget curl software-properties-common libmetis-dev ffmpeg unzip zip && \
	add-apt-repository -y ppa:longsleep/golang-backports && \
	apt-get update && \
	apt-get install -y golang-1.16-go
RUN curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py && \
	python3 get-pip.py && \
	pip3 install scikit-image scikit-video numpy requests Pillow opencv-python PyYAML matplotlib tqdm pandas seaborn

# golang
RUN ln -s /usr/lib/go-1.16/bin/go /usr/bin/go
