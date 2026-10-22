module hfd-demo

go 1.27

require github.com/hyggshi/hfd v0.0.0

require (
	github.com/godbus/dbus/v5 v5.2.4 // indirect
	golang.org/x/sys v0.27.0 // indirect
)

replace github.com/hyggshi/hfd => ../../core
