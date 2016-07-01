all: aceapi cgi-server updater

aceapi: aceapi.go
	go build aceapi.go

updater: aceapi
	cp aceapi updater

cgi-server: cgi-server.go
	go build cgi-server.go

deploy-updater:
	env GOOS=linux GOARCH=amd64 go build -o updater.linux -ldflags '-w' aceapi.go
	cp updater.linux /Volumes/plymouth.acenet.us/www/api/cgi-bin/updater

upload-debug:
	md5 aceapi
	curl --data-binary @aceapi -H "Token: `cat token.txt`" -k https://localhost:9001/updater/
