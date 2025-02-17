GO111MODULE=on

.PHONY: build
build: markdown-blog

.PHONY: markdown-blog
markdown-blog:
	go build $(RACE) -o bin/markdown-blog ./

.PHONY: build-race
build-race: enable-race build

.PHONY: run
run: build
	./bin/markdown-blog web -e dev

.PHONY: run-race
run-race: enable-race run

.PHONY: test
test:
	go test $(RACE) ./...

.PHONY: test-race
test-race: enable-race test

.PHONY: enable-race
enable-race:
	$(eval RACE = -race)

.PHONY: package
package: build
	bash ./package.sh

.PHONY: package-all
package-all: build
	bash ./package.sh -p 'linux darwin windows'

.PHONY: clean
clean:
	rm bin/markdown-bolg
