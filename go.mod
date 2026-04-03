module github.com/netgarden/rrpc-gen

go 1.24.1

replace github.com/netgarden/rrpc-def => ../rrpc-def

replace github.com/netgarden/rrpc-gen-go => ../rrpc-gen-go

replace github.com/netgarden/rrpc-gen-ts => ../rrpc-gen-ts

require (
	github.com/alecthomas/kingpin/v2 v2.4.0
	github.com/netgarden/rrpc-def v0.0.0
	github.com/netgarden/rrpc-gen-go v0.0.0
	github.com/netgarden/rrpc-gen-ts v0.0.0
)

require (
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
