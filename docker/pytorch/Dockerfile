FROM skyhookml/base

RUN pip3 install torch==1.7.1+cu110 torchvision==0.8.2+cu110 torchaudio==0.7.2 -f https://download.pytorch.org/whl/torch_stable.html

WORKDIR /usr/src/app/skyhook
RUN mkdir data

COPY ./ ./
RUN go build cmd/container.go

EXPOSE 8080
CMD ./container
