SHELL = /bin/bash
TARGETS = oaimi

# http://docs.travis-ci.com/user/languages/go/#Default-Test-Script
test: deps
	go test -v ./...

deps:
	go get ./...

imports:
	go get golang.org/x/tools/cmd/goimports
	goimports -w .

vet:
	go vet ./...

cover:
	go test -cover ./...

all: $(TARGETS)

oaimi: imports deps
	go build -o oaimi cmd/oaimi/main.go

clean:
	rm -f $(TARGETS)
	rm -f oaimi_*deb
	rm -f oaimi-*rpm
	rm -rf ./packaging/deb/oaimi/usr

deb: $(TARGETS)
	mkdir -p packaging/deb/oaimi/usr/sbin
	cp $(TARGETS) packaging/deb/oaimi/usr/sbin
	cd packaging/deb && fakeroot dpkg-deb --build oaimi .
	mv packaging/deb/oaimi_*.deb .

rpm: $(TARGETS)
	mkdir -p $(HOME)/rpmbuild/{BUILD,SOURCES,SPECS,RPMS}
	cp ./packaging/rpm/oaimi.spec $(HOME)/rpmbuild/SPECS
	cp $(TARGETS) $(HOME)/rpmbuild/BUILD
	./packaging/rpm/buildrpm.sh oaimi
	cp $(HOME)/rpmbuild/RPMS/x86_64/oaimi*.rpm .

cloc:
	cloc --max-file-size 1 --exclude-dir assets --exclude-dir assetutil --exclude-dir tmp --exclude-dir fixtures .

sites.tsv:
	curl "http://www.openarchives.org/pmh/registry/ListFriends" | \
		xmlstarlet sel -t -m "/BaseURLs/baseURL/text()" -c . -n - | grep -v '^$$' > sites.tsv

harvest: sites.tsv
	while IFS='' read -r line || [[ -n "$line" ]]; do time oaimi -verbose "$line" > /dev/null; done < sites.tsv
