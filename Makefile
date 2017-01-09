.PHONY: all clean package

all : gw2.exe

clean :
	rm -f gw2.exe gw2util.zip

package : gw2util.zip 

gw2util.zip : gw2.exe
	rm -f gw2util.zip && zip gw2util.zip gw2.exe

gw2.exe : main.go
	GOOS=windows go build -o gw2.exe

