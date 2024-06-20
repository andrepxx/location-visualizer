GCFLAGS_DEBUG := 'all=-N -l'
LDFLAGS_RELEASE := 'all=-w -s'
GOPATH := `pwd`/../../../..

all: locviz locviz-debug

.PHONY: clean fmt keys test

clean:
	rm -rf dist/
	rm -f locviz locviz-debug

locviz:
	go build -o locviz -ldflags $(LDFLAGS_RELEASE)

locviz-debug:
	go build -o locviz-debug -gcflags $(GCFLAGS_DEBUG)

fmt:
	gofmt -w .
	find \( -iname '*.css' -o -iname '*.js' -o -iname '*.json' -o -iname '*.md' -o -iname '*.xhtml' \) -execdir sed -i s/[[:space:]]*$$// {} \;

keys:
	mkdir keys
	openssl genrsa -out keys/private.pem 4096
	openssl req -new -x509 -days 365 -sha512 -key keys/private.pem -out keys/public.pem -subj "/C=DE/ST=Berlin/L=Berlin/O=None/OU=None/CN=localhost"

