# Solc-Go

Golang bindings for the [Solidity compiler](https://github.com/ethereum/solidity).

Uses the Emscripten compiled Solidity found in the [solc-bin repository](https://github.com/ethereum/solc-bin).

#### Example usage

The library now supports dynamic Solidity version selection with automatic binary downloading:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/rxtech-lab/solc-go"
)

func main() {
    // Show embedded versions (instant compilation)
    fmt.Printf("Embedded versions: %v\n", solc.GetEmbeddedVersions())
    
    // Create a compiler instance for a specific Solidity version
    // For embedded versions (0.8.30, 0.8.21), this is instant
    // For other versions, the binary will be downloaded automatically
    compiler, err := solc.NewWithVersion("0.8.30")
    if err != nil {
        log.Fatalf("Failed to create compiler: %v", err)
    }
    defer compiler.Close()

    input := &solc.Input{
        Language: "Solidity",
        Sources: map[string]solc.SourceIn{
            "SimpleContract.sol": {
                Content: `pragma solidity ^0.8.0;
                
                contract SimpleContract {
                    uint256 public value;
                    
                    function setValue(uint256 _value) public {
                        value = _value;
                    }
                }`,
            },
        },
        Settings: solc.Settings{
            Optimizer: solc.Optimizer{
                Enabled: true,
                Runs:    200,
            },
            OutputSelection: map[string]map[string][]string{
                "*": {
                    "*": []string{
                        "abi",
                        "evm.bytecode.object",
                        "evm.deployedBytecode.object",
                    },
                },
            },
        },
    }
    
    output, err := compiler.Compile(input)
    if err != nil {
        log.Fatalf("Compilation failed: %v", err)
    }

    fmt.Printf("Bytecode: %v", output.Contracts["SimpleContract.sol"]["SimpleContract"].EVM.Bytecode.Object)
}
```

#### Features

- **Dynamic Version Support**: Specify any Solidity version (e.g., "0.8.30", "0.7.6", "0.6.12")
- **Embedded Binaries**: Latest Solidity (0.8.30) and LTS (0.8.21) versions are pre-embedded for instant access
- **Automatic Fallback**: Other versions are downloaded automatically from the official Solidity repository
- **Weekly Updates**: Embedded binaries are automatically updated every Sunday via CI/CD
- **Full Compatibility**: Supports all Solidity compiler features and output options

#### Performance

For the best performance, use the embedded versions:
- **0.8.30** (latest) - instant compilation, no download required
- **0.8.21** (LTS) - instant compilation, no download required

Other versions will be downloaded on first use and may take a moment depending on network speed.
