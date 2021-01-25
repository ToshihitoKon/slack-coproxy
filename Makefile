help:
	@cat Makefile | grep "^\w"

run:
	go run src/*

build:
	go build -o bin/slack-coproxy src/*
