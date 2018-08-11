mkdir -p $GOPATH/src/golang.org/x
# 安装引用包
go get github.com/golang/sys
mv $GOPATH/src/github.com/golang/sys $GOPATH/src/golang.org/x/sys

go get github.com/golang/crypto
mv $GOPATH/src/github.com/golang/crypto $GOPATH/src/golang.org/x/crypto