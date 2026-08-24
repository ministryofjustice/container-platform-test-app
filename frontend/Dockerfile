FROM golang:1.26@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS build

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY *.go ./
COPY index.html.tmpl ./

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /frontend

FROM scratch

WORKDIR /
COPY --from=build /frontend /frontend
COPY --from=build /app/index.html.tmpl /index.html.tmpl

USER 1001:1001

ENTRYPOINT ["/frontend"]
