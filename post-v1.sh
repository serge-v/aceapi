curl \
	-H "Content-Type: application/octet-stream" \
	--cacert acenet.crt \
	--data-binary @aceapi.linux \
	-H "Token: `cat token.txt`" \
	--url https://api.voilokov.com/updater/

md5 aceapi.linux
