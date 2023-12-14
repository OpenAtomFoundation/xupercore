module github.com/OpenAtomFoundation/xupercore/crypto-dll-go

go 1.19

require (
	github.com/OpenAtomFoundation/xupercore/crypto-rust/x-crypto-ffi v0.0.0-20231214063800-d385e2c4dd24
	github.com/stretchr/testify v1.8.4
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/OpenAtomFoundation/xupercore/crypto-rust/x-crypto-ffi => ../crypto-rust/x-crypto-ffi
