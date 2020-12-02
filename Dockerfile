FROM golang:alpine as builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor  -a -installsuffix cgo -ldflags '-extldflags "-static"' -o azure_casb .
FROM scratch
FROM microsoft/azure-cli
#COPY --from=builder /build/azure_casb /app/
COPY --from=builder /build/azure_casb $GOPATH/bin
WORKDIR /app
#CMD ["/bin/bash"]