VERSION := v0.0.8

.PHONY: hoyo resin stamina charge all clean login

ENV := CGO_ENABLED=1 GOOS=windows GOARCH=amd64
LDFLAGS := -ldflags "-H=windowsgui -X resin/pkg/config.VERSION=$(VERSION)"

# Default: build the unified binary
all: hoyo

hoyo:
	${ENV} go build $(LDFLAGS) -o hoyo.exe cmd/hoyo/main.go

# Legacy individual binaries (kept for backward compatibility)
resin:
	${ENV} go build $(LDFLAGS) -o resin.exe cmd/resin/main.go

stamina:
	${ENV} go build $(LDFLAGS) -o stamina.exe cmd/stamina/main.go

charge:
	${ENV} go build $(LDFLAGS) -o charge.exe cmd/charge/main.go

legacy: resin stamina charge

login:
	VERSION=$(VERSION) ./buildLogin

clean:
	rm -rf hoyo*.exe
	rm -rf resin*.exe
	rm -rf stamina*.exe
	rm -rf charge*.exe
	rm -rf login/*.exe.WebView2

