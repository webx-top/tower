go build -o $GOPATH/bin/tower
go build -ldflags "-X main.build=0" -o $GOPATH/bin/tower-product
