.PHONY: all clean

all : gw2.exe

clean :
	rm gw2.exe

gw2.exe : main.go
	GOOS=windows go build -o gw2.exe
