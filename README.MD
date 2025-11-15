to install:
go install github.com/mdtosif/icarus@latest

to uninstall:
rm $(go env GOPATH)/bin/icarus

to get available flags:
icarus -h

to run:
icarus -mnemonic "your mnemonic phrase here" -rpc-url "https://your.ethereum.node" -wallets 10 -txns 100 -wait 10ms -log-level 1