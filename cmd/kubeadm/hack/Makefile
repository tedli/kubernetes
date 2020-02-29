Version =v4.1.1

all: tde

compress:
	upx --best --ultra-brute -o kubeadm kubeadm.elf

tde: compress
	docker build -t 192.168.1.52/system_containers/tde:${Version}  .

arm64:
	docker build -t 192.168.1.52/system_containers/tde-arm64:${Version}  .

clean:
	rm -f ./$(shell basename $(shell pwd)) >/dev/null
