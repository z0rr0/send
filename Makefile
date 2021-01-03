TARGET=send
TS=$(shell date -u +"%F_%T")
TAG=$(shell git tag | sort --version-sort | tail -1)
COMMIT=$(shell git log --oneline | head -1)
VERSION=$(firstword $(COMMIT))
# use system environment variable TMPDIR
TEST_DIR=$(shell if test -d "$(TMPDIR)"; then echo $(TMPDIR); else echo "/tmp/"; fi)
ESCAPED_TEST_DIR=$(shell echo $(TEST_DIR) | sed 's/\//\\\//g')
FLAG=-X main.Version=$(TAG) -X main.Revision=git:$(VERSION) -X main.BuildDate=$(TS)
PIDFILE=$(TEST_DIR).$(TARGET).pid
PWD=$(shell pwd)
# use custom environment env SENDCFG for config
RUN_CFG=$(shell if test -f "$(SENDCFG)"; then echo $(SENDCFG); else echo "doc/config.toml"; fi)
LOG_FILE=$(PWD)/send.log
# test configuration
TEST_CONFIG=test_$(TARGET).toml
TEST_DB=test_$(TARGET).sqlite
TEST_STORAGE=test_$(TARGET)

all: test

build:
	go build -o $(PWD)/$(TARGET) -ldflags "$(FLAG)"

rebuild: clean lint build

prepare:
	@rm -rf $(TEST_DIR)test_$(TARGET)*
	@cp doc/config.toml $(TEST_DIR)$(TEST_CONFIG)
	@sed -i.b 's/"db.sqlite"/"$(ESCAPED_TEST_DIR)$(TEST_DB)"/' $(TEST_DIR)$(TEST_CONFIG)
	@sed -i.b 's/"storage"/"$(ESCAPED_TEST_DIR)$(TEST_STORAGE)"/' $(TEST_DIR)$(TEST_CONFIG)
	@sed -i.b 's/"html"/"$(ESCAPED_TEST_DIR)$(TEST_STORAGE)\/html"/' $(TEST_DIR)$(TEST_CONFIG)
	@sed -i.b 's/"html\/static"/"$(ESCAPED_TEST_DIR)$(TEST_STORAGE)\/html\/static"/' $(TEST_DIR)$(TEST_CONFIG)
	@mkdir $(TEST_DIR)$(TEST_STORAGE)
	@cp -r html $(TEST_DIR)$(TEST_STORAGE)
	@cat doc/schema.sql | sqlite3 $(TEST_DIR)$(TEST_DB)

check_fmt:
	@test -z "`gofmt -l .`" || { echo "ERROR: failed gofmt, for more details run - make fmt"; false; }
	@-echo "gofmt successful"

fmt:
	gofmt -d .

lint: check_fmt
	go vet $(PWD)/...
	golint -set_exit_status $(PWD)/...
	golangci-lint run $(PWD)/...

test: lint prepare
	go test -race -v -cover $(PWD)/...
	# go test -race -v -cover -coverprofile=coverage.out -trace trace.out <PACKAGE>
	# go tool cover -html=coverage.out

bench: lint prepare
	go test -race -cover -benchmem -bench=. $(PWD)/...

test_nocache: lint prepare
	go test -count=1 -race -v -cover $(PWD)/...

# github actions test
actions: check_fmt prepare
	go vet $(PWD)/...
	go test -race -cover $(PWD)/...

clean:
	rm -f $(TARGET)
	find $(PWD)/ -type f -name "*.out" -delete
	rm -rf $(TEST_DIR)test_$(TARGET)*

# for local running only
start: build
	@test ! -f $(PIDFILE) || { echo "ERROR: pid file already exists $(PIDFILE)"; false; }
	@-echo ">>> starting $(TARGET)"
	@$(PWD)/$(TARGET) -config $(RUN_CFG) -log $(LOG_FILE) & echo $$! > $(PIDFILE)
	@-cat $(PIDFILE)
#	@-grep -A 2 "server" $(RUN_CFG)

stop:
	@test -f $(PIDFILE) || { echo "ERROR: pid file not found $(PIDFILE)"; false; }
	@echo "kill $(TARGET) pid=`cat $(PIDFILE)`"
	@kill `cat $(PIDFILE)`
	@-rm -f $(PIDFILE)
	@-echo ">>> stopped $(TARGET)"

restart: stop start