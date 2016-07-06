if [[ $1 == "get" ]] ; then
	curl \
		-s \
		-H "Token: `cat token.txt`" \
		--cacert acenet.crt \
		--url https://api.voilokov.com/v1/cron | tee 1~.jobs
fi

if [[ $1 == "set" ]] ; then
	curl \
		-s \
		-X POST
		--data-binary @1~.jobs
		-H "Token: `cat token.txt`" \
		-H "Content-type: text/plain" \
		--cacert acenet.crt \
		--url https://api.voilokov.com/v1/cron
fi

if [[ $1 == "" ]] ; then
	curl -k -u `cat pwd.txt` https://api.voilokov.com/v1/cron
fi
