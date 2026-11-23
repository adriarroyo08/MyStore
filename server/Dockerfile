FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY web ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /mystore .

FROM gcr.io/distroless/static
COPY --from=build /mystore /mystore
ENV MYSTORE_DATA=/data
VOLUME /data
EXPOSE 8080
# Sin curl en distroless: el propio binario pregunta a /healthz.
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 CMD ["/mystore", "healthcheck"]
ENTRYPOINT ["/mystore"]
