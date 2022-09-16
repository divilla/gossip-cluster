MODULE = $(shell go list -m)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || echo "1.0.0")
PACKAGES := $(shell go list ./... | grep -v /vendor/)
LDFLAGS := -ldflags "-X main.Version=${VERSION}"

TIMESTAMP=$(shell date +'%Y-%m-%d_%H:%M:%S')
DB_CONTAINER = ds-mysql-olx-pl
DB_NAME = osc

#CONFIG_FILE ?= ./config/local.yml
#APP_DSN ?= $(shell sed -n 's/^dsn:[[:space:]]*"\(.*\)"/\1/p' $(CONFIG_FILE))
#MIGRATE := docker run -v $(shell pwd)/migrations:/migrations --network host migrate/migrate:v4.10.0 -path=/migrations/ -database "$(APP_DSN)"

PID_FILE := './.pid'
FSWATCH_FILE := './fswatch.cfg'

.PHONY: default
default: help

# generate help info from comments: thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help: ## help information about make commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: ## run unit tests
	@echo "mode: count" > coverage-all.out
	@$(foreach pkg,$(PACKAGES), \
		go test -p=1 -cover -covermode=count -coverprofile=coverage.out ${pkg}; \
		tail -n +2 coverage.out >> coverage-all.out;)

.PHONY: test-cover
test-cover: test ## run unit tests and show test coverage information
	go tool cover -html=coverage-all.out

.PHONY: run
run: ## run the API server
	go run ${LDFLAGS} cmd/console/main.go

.PHONY: run-restart
run-restart: ## restart the API server
	@pkill -P `cat $(PID_FILE)` || true
	@printf '%*s\n' "80" '' | tr ' ' -
	@echo "Source file changed. Restarting server..."
	@go run ${LDFLAGS} cmd/server/main.go & echo $$! > $(PID_FILE)
	@printf '%*s\n' "80" '' | tr ' ' -

run-live: ## run the API server with live reload support (requires fswatch)
	@go run ${LDFLAGS} cmd/server/main.go & echo $$! > $(PID_FILE)
	@fswatch -x -o --event Created --event Updated --event Renamed -r internal pkg cmd config | xargs -n1 -I {} make run-restart

.PHONY: build
build:  ## build the API server binary
	@cp config/config.yaml gc.yaml
	CGO_ENABLED=0 go build ${LDFLAGS} -a -o gc $(MODULE)/cmd/console/.

.PHONY: build
build-aws:  ## build the API server binary
	@cp config/config.yml ~/projects/ansible/kinesis/apps/mysql-balanced-loader-config.yml
	CGO_ENABLED=0 go build ${LDFLAGS} -a -o ~/projects/ansible/kinesis/apps/mysql-balanced-loader-console $(MODULE)/cmd/console/.

.PHONY: build-docker
build-docker: ## build the API server as a docker image
	docker build -f cmd/server/Dockerfile -t server .

.PHONY: clean
clean: ## remove temporary files
	rm -rf server coverage.out coverage-all.out

.PHONY: version
version: ## display the version of the API server
	@echo $(VERSION)

.PHONY: db-run
db-run: ## start the database server
	mkdir -p mysql/.data
	chmod a+rwx mysql/.data
	docker run \
		--name ${DB_CONTAINER} \
		-p 3306:3306 \
		-v "$(shell pwd)/mysql/conf/my_custom.cnf":/opt/bitnami/mysql/conf/my_custom.cnf:ro \
		-v "$(shell pwd)/mysql/docker-entrypoint-initdb.d":/docker-entrypoint-initdb.d \
		-v "$(shell pwd)/mysql/.data":/bitnami/mysql/data \
		-e ALLOW_EMPTY_PASSWORD=yes \
		-e MYSQL_USER=user \
		-e MYSQL_PASSWORD=pass \
		-e MYSQL_DATABASE=${DB_NAME} \
		-d bitnami/mysql:5.7

.PHONY: db-remove
db-remove: ## stop the database server
	docker stop ${DB_CONTAINER}
	docker container rm ${DB_CONTAINER}

.PHONY: db-start
db-start: ## start the database server
	docker start ${DB_CONTAINER}

.PHONY: db-stop
db-stop: ## stop the database server
	docker stop ${DB_CONTAINER}

.PHONY: db-ssh
db-ssh: ## stop the database server
	docker exec -it ${DB_CONTAINER} /bin/bash

.PHONY: db-backup
db-backup: ## backup database to 'mysql/backup.sql' & copy with rename to 'mysql/dump' directory
	mkdir -p data/dump
	docker exec -i ${DB_CONTAINER} \
		/opt/bitnami/mysql/bin/mysqldump -u root ${DB_NAME} > mysql/backup.sql
	cp mysql/backup.sql data/dump/${DB_NAME}_${TIMESTAMP}.sql

.PHONY: db-restore
db-restore: ## restore database
	cat mysql/backup.sql | \
	docker exec -i ${DB_CONTAINER} \
		/opt/bitnami/mysql/bin/mysql -u root ${DB_NAME}

LOGFILE=$(shell date +'%Y-%m-%d-%H-%M-%S')
.PHONY: date
date:
	$(LOGFILE)

.PHONY: testdata
testdata: ## populate the database with test data
	make migrate-reset
	@echo "Populating test data..."
	@docker exec -it postgres psql "$(APP_DSN)" -f /testdata/testdata.sql

.PHONY: lint
lint: ## run golint on all Go package
	@golint $(PACKAGES)

.PHONY: fmt
fmt: ## run "go fmt" on all Go packages
	@go fmt $(PACKAGES)
