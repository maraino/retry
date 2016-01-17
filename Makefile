PACKAGE=github.com/maraino/retry

all:
	go build $(PACKAGE)

test:
	go test -cover $(PACKAGE)

cover:
	go test -coverprofile=c.out $(PACKAGE)
	go tool cover -html=c.out
