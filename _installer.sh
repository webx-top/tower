go get github.com/admpub/glide
glide install
go install
go install -ldflags "-X main.build=0"