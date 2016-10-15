go build -o %GOPATH%/bin/tower.exe
go build -ldflags "-X main.build=0" -o %GOPATH%/bin/tower-product.exe
pause