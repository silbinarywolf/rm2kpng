module github.com/silbinarywolf/cmd/rm2kfixwatcher

go 1.19

require (
	github.com/karrick/godirwalk v1.17.0
	github.com/silbinarywolf/rm2kpng v1.0.0
	gopkg.in/fsnotify.v1 v1.4.7
)

require (
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	golang.org/x/sys v0.3.0 // indirect
)

replace github.com/silbinarywolf/rm2kpng => ../..
