all: aceapi cgi-server updater

aceapi: aceapi.go Makefile
	go build -ldflags "-X main.date=`date +%Y-%m-%dT%H:%M:%S%z` -X main.version=`git describe --long --tags`" aceapi.go

updater: aceapi
	cp aceapi updater

cgi-server: cgi-server.go
	go build cgi-server.go

aceapi.linux: aceapi.go Makefile
	env GOOS=linux GOARCH=amd64 go build -o aceapi.linux -ldflags "-w -X main.date=`date +%Y-%m-%dT%H:%M:%S%z` -X main.version=`git describe --long --tags`" aceapi.go

deploy-updater: aceapi.linux
	cp aceapi.linux /Volumes/plymouth.acenet.us/www/api/cgi-bin/updater

upload-prod: aceapi.linux
	./post-v1.sh

upload-debug: aceapi
	curl --data-binary @aceapi -H "Token: `cat token.txt`" -k https://localhost:9001/updater/
	md5 aceapi

clean:
	rm -f aceapi.linux aceapi updater cgi-server
