FROM golang:1.4

# Fetch dependencies
RUN go get github.com/tools/godep

# Add project directory to Docker image.
ADD . /go/src/github.com/Recras/exactonline

ENV HTTP_ADDR 8888
ENV HTTP_DRAIN_INTERVAL 1s
ENV COOKIE_SECRET 4iKivAZAZORgZ3ya

WORKDIR /go/src/github.com/Recras/exactonline

RUN godep go build

EXPOSE 8888
CMD ./exactonline
