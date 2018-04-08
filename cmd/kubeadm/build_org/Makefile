Version =v3.0.0

all: build

build: clean
	GOPATH=$(GOPATH):$(shell cd ../../;pwd);go get;CGO_ENABLED=0 go build

godep-build: clean
	GOPATH=$(GOPATH):$(shell cd ../../;pwd);CGO_ENABLED=0 godep go build

compress:
	docker run --rm -v  $(shell pwd):/src/  -w /src index.tenxcloud.com/tenx_containers/goupx:latest goupx kubeadm

save:
	GOPATH=$(GOPATH):$(shell cd ../../;pwd);go get;godep save

docker-build: clean
	docker run --rm -v $(shell pwd):/go/src/$(shell basename $(shell pwd))  \
	-w /go/src/$(shell basename $(shell pwd)) index.tenxcloud.com/tenx_containers/godep:alpine  	\
	/bin/sh -c "CGO_ENABLED=0 godep go build -gcflags \"-N -l\""

tde: compress
	docker build -t 192.168.1.55/tenx_containers/tde:${Version}  .

build-tde: godep-build compress
	docker build -t index.tenxcloud.com/tenx_containers/tde:${Version}  .

build-container:
	docker build -t index.tenxcloud.com/tenx_containers/tde:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
