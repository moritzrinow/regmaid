FROM golang:1.24.2 AS build

WORKDIR /src

COPY go.mod go.sum .

RUN go mod download

COPY *.go ./

COPY internal/regmaid/ ./internal/regmaid

RUN CGO_ENABLED=0 go build -o /regmaid/regmaid .

FROM busybox:1.36

WORKDIR /regmaid

COPY README.md LICENSE .

COPY --from=build /regmaid/regmaid .

ENTRYPOINT ["/regmaid/regmaid", "-c", "/etc/regmaid/regmaid.yaml"]
