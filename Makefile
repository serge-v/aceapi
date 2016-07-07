VERSION=$(shell git describe --long --tags)
DATE=$(shell date +%Y-%m-%dT%H:%M:%S%z)
LDFLAGS="-X main.date=$(DATE) -X main.version=$(VERSION)"

all: aceapi cgi-server updater

debug: aceapi.go aceapidbg.go
	go build -o aceapi.debug -ldflags $(LDFLAGS) aceapi.go aceapidbg.go
	cgdb -d /opt/local/bin/ggdb --args ./aceapi.debug -t

test: *.go
	go test -test.v aceapi.go aceapi_test.go

aceapi: aceapi.go aceapidbg.go Makefile
	go build -ldflags $(LDFLAGS) aceapi.go aceapidbg.go

updater: aceapi
	cp aceapi updater

cgi-server: cgi-server.go
	go build cgi-server.go

aceapi.linux: aceapi
	env GOOS=linux GOARCH=amd64 go build -o aceapi.linux -ldflags $(LDFLAGS) aceapi.go

deploy-updater: aceapi.linux
	cp aceapi.linux /Volumes/plymouth.acenet.us/www/api/cgi-bin/updater

deploy-prod: aceapi.linux
	./post-v1.sh

deploy-debug: aceapi
	curl --data-binary @aceapi -H "Token: `cat token.txt`" -k https://localhost:9001/updater/
	md5 aceapi

clean:
	rm -f aceapi.linux aceapi updater cgi-server
