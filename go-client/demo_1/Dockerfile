FROM golang:1.24 As build-stage
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct
COPY . .
RUN go mod download &&  go build main.go

FROM ubuntu:22.04
WORKDIR /
COPY --from=build-stage /app/main /main
ENTRYPOINT ["/main"]