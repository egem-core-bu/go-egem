.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    git.egem.io/team/go-egem/cmd/swarm
	go build -o bin/geth     git.egem.io/team/go-egem/cmd/geth
	go build -o bin/bootnode git.egem.io/team/go-egem/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v git.egem.io/team/go-egem/swarm/...
