


**rfhEVM** is an open-source library used to easily integrate the into an EVM-compatible blockchain.

## Main features

fhEVM-go gives your EVM the ability to compute on encrypted data using fully homomorphic encryption by:

- a collection of operations on encrypted data via precompiled contracts
- various additional EVM components that support encrypted computation

## Getting started

In order to use the library, you need to clone the repository and build it. This is required because the library depends on the `tfhe-rs` library that needs to be built from source (for now), and Go doesn't support such a build.

```bash
$ git clone --recurse-submodules https://github.com/lukads12345/rfhevm
$ cd rfhevm
$ make build
```

That's it! You can now use it in your project by adding it to `go.mod`, and adding a `replace` to point to your local build. An example using `rfhevm` v1.0.0:

```
...
require(
    ...
    https://github.com/lukads12345/rfhevm v1.0.0
    ...
)

replace(
    ...
    https://github.com/lukads12345/rfhevm v1.0.0 => /path/to/your/local/fhevm-go
    ...
)
...
```

> [!NOTE]
> The replace in necessary for now as Go build system can't build the `tfhe-rs` library that `rfhevm` needs. It's therefore necessary that we build it manually as mentioned above, then point to our ready-to-use directory in `go.mod`.

## Regenerate protobuff files

To re-generate these files, install `protoc`, `protoc-gen-go` and `protoc-gen-go-grpc` and run protoc
`cd proto && protoc --go_out=../fhevm/kms --go_opt=paths=source_relative --go-grpc_out=../fhevm/kms --go-grpc_opt=paths=source_relative kms.proto && cd ..`.


## Target users

The library helps EVM maintainers to extend their EVM with the power of FHE. If you are looking for a library to deploy and use smart contracts on an rfhEVM.


## License


This software is distributed under the BSD-3-Clause-Clear license. .
