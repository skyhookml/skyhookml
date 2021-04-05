FROM skyhookml/base

WORKDIR /usr/src/app/skyhook
RUN mkdir data

COPY ./ ./
RUN go build cmd/main.go

EXPOSE 8080
CMD ["./docker/coordinator/entrypoint.sh"]
