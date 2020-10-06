module github.com/silbinarywolf/cmd/rm2kfixwatcher

go 1.15

require (
	github.com/karrick/godirwalk v1.16.1
	github.com/silbinarywolf/rm2kpng v1.0.0
	golang.org/x/sys v0.0.0-20200930185726-fdedc70b468f // indirect
	gopkg.in/fsnotify.v1 v1.4.7
)

replace github.com/silbinarywolf/rm2kpng => ../..
