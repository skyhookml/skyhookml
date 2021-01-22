FROM skyhookml/base

RUN pip3 install torch==1.7.0+cu110 torchvision==0.8.1+cu110 torchaudio===0.7.0 -f https://download.pytorch.org/whl/torch_stable.html

WORKDIR /usr/src/app/skyhook
RUN mkdir items models

COPY ./ ./
RUN go build container.go

EXPOSE 8080
CMD go run container.go
