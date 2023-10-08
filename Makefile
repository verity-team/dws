.PHONY: all build 

BIN_DIR := ./bin
version := $(shell git rev-parse --short=12 HEAD)
timestamp := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
COMPOSE_CMD := docker-compose
DOCKER_COMPOSE_FOLDER=deployments/docker

all: build

clean:
	rm -f $(BIN_DIR)/delphi

build:
	rm -f $(BIN_DIR)/delphi
	go build -o $(BIN_DIR)/delphi -v -ldflags \
  		"-X main.rev=$(version) -X main.bts=$(timestamp)" cmd/delphi/main.go


run_dws_db:
	$(COMPOSE_CMD) -f $(DOCKER_COMPOSE_FOLDER)/db.yaml  up -d dws-db

dws_db_prompt:
	export PGPASSWORD=postgres && psql -U postgres -h localhost -p 27501 -d dwsdb
