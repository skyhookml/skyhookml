FROM skyhookml/base

WORKDIR /usr/src/app/skyhook
RUN mkdir data

COPY ./ ./
RUN go build cmd/container.go

EXPOSE 8080
CMD ./container
