CFLAGS = -g -Wall
LDFLAGS = -L. -lrclone -lpthread -ldl

static: main

shared: exports.go
	go build --buildmode=c-shared -o librclone.so exports.go

main: main.o librclone.h
	$(CC) $(CFLAGS) -o $@ $< $(LDFLAGS)

main.o: main.c librclone.h
	$(CC) $(CFLAGS) -c $^ $(LDFLAGS)

build: exports.go
	go build

librclone.h: exports.go
	go build --buildmode=c-archive -o librclone.a exports.go

clean:
	rm -f tmp main *.o *.a *.h *.gch
