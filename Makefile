USERNAME=bketelsen

build-forward:
	docker build -t ${USERNAME}/forwarder .
forward:
	docker run -d -p 55678:55678 -e APPINSIGHTS_INSTRUMENTATIONKEY=${APPINSIGHTS_INSTRUMENTATIONKEY} ${USERNAME}/forwarder

