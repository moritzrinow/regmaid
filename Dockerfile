FROM golang:1.24.2 AS build

WORKDIR /src

COPY go.mod go.sum .

RUN go mod download

COPY *.go ./

COPY internal/regmaid/ ./internal/regmaid

RUN CGO_ENABLED=0 go build -o /regmaid/regmaid .

FROM alpine:3.17.0

WORKDIR /regmaid

COPY README.md LICENSE .

COPY --from=build /regmaid/regmaid .

ENTRYPOINT ["/regmaid/regmaid", "-c", "/etc/regmaid/regmaid.yaml"]
