# 安装引用包
go get github.com/golang/sys
mkdir $GOPATH/src/golang.org/x/sys
mv $GOPATH/src/github.com/golang/sys/* $GOPATH/src/golang.org/x/sys
sudo mv $GOPATH/src/github.com/golang/sys/.git* $GOPATH/src/golang.org/x/sys
rm -rf $GOPATH/src/github.com/golang/sys

go get github.com/golang/crypto
mkdir $GOPATH/src/golang.org/x/crypto
mv $GOPATH/src/github.com/golang/crypto/* $GOPATH/src/golang.org/x/crypto
mv $GOPATH/src/github.com/golang/crypto/.git* $GOPATH/src/golang.org/x/crypto
rm -rf $GOPATH/src/github.com/golang/crypto