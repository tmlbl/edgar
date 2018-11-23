bin/gogurt: $(wildcard *.go) $(wildcard cmd/*.go)
	mkdir -p bin
	go build -o bin/gogurt cmd/*.go

