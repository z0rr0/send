TARGET=send
TS=$(shell date -u +"%F_%T")
TAG=$(shell git tag | sort --version-sort | tail -1)
COMMIT=$(shell git log --oneline | head -1)
VERSION=$(firstword $(COMMIT))
FLAG=-X main.Version=$(TAG) -X main.Revision=git:$(VERSION) -X main.BuildDate=$(TS)
PIDFILE=/tmp/.$(TARGET).pid
PWD=$(shell pwd)
CONFIG=config.toml
# test configuration
TEST_CONFIG=test_$(TARGET).toml
TEST_DB=test_$(TARGET).sqlite
TEST_STORAGE=test_$(TARGET)

all: test

build:
	go build -o $(PWD)/$(TARGET) -ldflags "$(FLAG)"

rebuild: clean lint build

prepare:
	@rm -rf /tmp/test_$(TARGET)*
	@cp doc/config.toml /tmp/$(TEST_CONFIG)
	@sed -i.b 's/"db.sqlite"/"\/tmp\/$(TEST_DB)"/' /tmp/$(TEST_CONFIG)
	@sed -i.b 's/"storage"/"\/tmp\/$(TEST_STORAGE)"/' /tmp/$(TEST_CONFIG)
	@mkdir /tmp/$(TEST_STORAGE)
	@cat doc/schema.sql | sqlite3 /tmp/$(TEST_DB)

check_fmt:
	@test -z "`gofmt -l .`" || { echo "ERROR: failed gofmt, for more details run - make fmt"; false; }
	@-echo "gofmt successful"

fmt:
	gofmt -d .

lint: check_fmt
	go vet $(PWD)/...
	golint -set_exit_status $(PWD)/...

test: lint prepare
	go test -race -v -cover $(PWD)/...

# github actions test
actions: check_fmt prepare
	go vet $(PWD)/...
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