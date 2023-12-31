.PHONY: all build 

BIN_DIR := ./bin
version := $(shell git rev-parse --short=12 HEAD)
timestamp := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
COMPOSE_CMD := docker-compose
DOCKER_COMPOSE_FOLDER=deployments/docker
DELPHI_SERVER_PGK=github.com/verity-team/dws/internal/delphi/server

all: build

clean:
	rm -f $(BIN_DIR)/buck
	rm -f $(BIN_DIR)/delphi
	rm -f $(BIN_DIR)/pulitzer
	rm -f $(BIN_DIR)/tt

build: lint
	rm -f $(BIN_DIR)/buck
	go build -o $(BIN_DIR)/buck -v -ldflags \
		"-X main.rev=$(version) -X main.bts=$(timestamp)" cmd/buck/main.go
	rm -f $(BIN_DIR)/delphi
	go build -o $(BIN_DIR)/delphi -v -ldflags \
		"-X main.rev=$(version) -X main.bts=$(timestamp) -X $(DELPHI_SERVER_PGK).rev=$(version) -X $(DELPHI_SERVER_PGK).bts=$(timestamp)" cmd/delphi/main.go
	rm -f $(BIN_DIR)/pulitzer
	go build -o $(BIN_DIR)/pulitzer -v -ldflags \
		"-X main.rev=$(version) -X main.bts=$(timestamp)" cmd/pulitzer/main.go
	rm -f $(BIN_DIR)/tt
	go build -o $(BIN_DIR)/tt -v -ldflags \
		"-X main.rev=$(version) -X main.bts=$(timestamp)" cmd/tt/main.go

run_db:
	$(COMPOSE_CMD) -f $(DOCKER_COMPOSE_FOLDER)/db.yaml  up -d dws-db

prompt:
	export PGPASSWORD=postgres && psql -U postgres -h localhost -p 27501 -d dwsdb

destroy_db:
	docker kill docker_dws-db_1 && docker rm docker_dws-db_1 && docker volume rm docker_dws_db_volume

codegen:
	oapi-codegen -config configs/models.cfg.yaml api/delphi.yaml
	oapi-codegen -config configs/server.cfg.yaml api/delphi.yaml

lint:
	golangci-lint run

run_delphi:
	rm -f /tmp/delphi*.log
	bin/delphi 2>&1 | tee /tmp/delphi-`date +'%Y-%m-%d_%H-%M-%S'`.log

run_pulitzer:
	rm -f /tmp/pulitzer*.log
	bin/pulitzer 2>&1 | tee /tmp/pulitzer-`date +'%Y-%m-%d_%H-%M-%S'`.log
	
run_buck_latest:
	rm -f /tmp/buck-latest*.log
	bin/buck --monitor-latest 2>&1 | tee /tmp/buck-latest-`date +'%Y-%m-%d_%H-%M-%S'`.log

run_buck_final:
	rm -f /tmp/buck-final*.log
	bin/buck --monitor-final 2>&1 | tee /tmp/buck-final-`date +'%Y-%m-%d_%H-%M-%S'`.log

run_buck_old_unconfirmed:
	rm -f /tmp/buck-old_unconfirmed*.log
	bin/buck --monitor-old-unconfirmed 2>&1 | tee /tmp/buck-old_unconfirmed-`date +'%Y-%m-%d_%H-%M-%S'`.log

test: lint
	go test ./...
