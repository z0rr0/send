TARGET=send
TS=$(shell date -u +"%F_%T")
TAG=$(shell git tag | sort --version-sort | tail -1)
COMMIT=$(shell git log --oneline | head -1)
VERSION=$(firstword $(COMMIT))
FLAG=-X main.Version=$(TAG) -X main.Revision=git:$(VERSION) -X main.BuildDate=$(TS)
PIDFILE=/tmp/.$(TARGET).pid
PWD=$(shell pwd)
CONFIG=config.json

all: test

build:
	go build -o $(PWD)/$(TARGET) -ldflags "$(FLAG)"

rebuild: clean lint build

check_fmt:
	@test -z "`gofmt -l .`" || { echo "ERROR: failed gofmt, for more details run - make fmt"; false; }
	@-echo "gofmt successful"

fmt:
	gofmt -d .

lint: check_fmt
	go vet $(PWD)/...
	golint -set_exit_status $(PWD)/...

test: lint
	go test -race -v -cover $(PWD)/...

clean:
	rm $(TARGET)
	find $(PWD)/ -type f -name "*.out" -delete

# for local running only
start: build
	@test ! -f $(PIDFILE) || { echo "ERROR: pid file already exists $(PIDFILE)"; false; }
	@-echo ">>> starting $(TARGET)"
	@$(PWD)/$(TARGET) -config $(CONFIG) & echo $$! > $(PIDFILE)
	@-cat $(PIDFILE)
	@-grep -A 1 "bind-host" $(CONFIG)

stop:
	@test -f $(PIDFILE) || { echo "ERROR: pid file not found $(PIDFILE)"; false; }
	@echo "kill $(TARGET) pid=`cat $(PIDFILE)`"
	@kill `cat $(PIDFILE)`
	@-rm -f $(PIDFILE)
	@-echo ">>> stopped $(TARGET)"

restart: stop start