FROM golang:1.26@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /backend

FROM scratch

COPY --from=build /backend /backend

USER 1001:1001

ENTRYPOINT ["/backend"]
