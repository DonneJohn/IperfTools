#go version
go version go1.12.7 windows/386


#WALK INSTALL
#https://github.com/lxn/walk
go get github.com/lxn/walk

rsrc INSTALL
#https://github.com/akavel/rsrc
go get github.com/akavel/rsrc
rsrc -manifest main.manifest -o rsrc.syso

#Build app
#In the directory containing main.go run
go build

#To get rid of the cmd window, instead run
go build -ldflags="-H windowsgui"

#parse ini file
https://github.com/go-gcfg/gcfg
go get gopkg.in/gcfg.v1


#gbk utf transfor
go get github.com/axgle/mahonia