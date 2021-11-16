rsrc -manifest main.manifest -ico img/logo.ico -o rsrc.syso
rem go build -ldflags="-H windowsgui -linkmode internal"
go build -tags walk_use_cgo -ldflags="-H windowsgui -linkmode internal"
rem go build -ldflags="-H windowsgui"
pause