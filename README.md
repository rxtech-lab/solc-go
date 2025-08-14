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
    
    // Compile with default options (no import resolution)
    output, err := compiler.CompileWithOptions(input, nil)
    if err != nil {
        log.Fatalf("Compilation failed: %v", err)
    }

    fmt.Printf("Bytecode: %v", output.Contracts["SimpleContract.sol"]["SimpleContract"].EVM.Bytecode.Object)
}
```

#### Import Resolution

Solc-Go supports resolving import statements through custom import callbacks. This is useful when your Solidity contracts import from external libraries or files that need to be resolved at compile time.

```go
package main

import (
    "fmt"
    "log"
    "path/filepath"
    "strings"
    
    "github.com/rxtech-lab/solc-go"
)

func main() {
    compiler, err := solc.NewWithVersion("0.8.21")
    if err != nil {
        log.Fatalf("Failed to create compiler: %v", err)
    }
    defer compiler.Close()

    // Contract that imports external dependencies
    input := &solc.Input{
        Language: "Solidity",
        Sources: map[string]solc.SourceIn{
            "Calculator.sol": {
                Content: `pragma solidity ^0.8.0;
                
                import "./lib/Math.sol";
                import "@openzeppelin/contracts/utils/Context.sol";
                
                contract Calculator {
                    function add(uint256 a, uint256 b) public pure returns (uint256) {
                        return Math.add(a, b);
                    }
                }`,
            },
        },
        Settings: solc.Settings{
            OutputSelection: map[string]map[string][]string{
                "*": {"*": []string{"abi", "evm.bytecode"}},
            },
        },
    }

    // Define import resolution callback
    options := &solc.CompileOptions{
        ImportCallback: func(url string) solc.ImportResult {
            // Handle relative imports
            if strings.HasPrefix(url, "./lib/") {
                switch url {
                case "./lib/Math.sol":
                    return solc.ImportResult{
                        Contents: `pragma solidity ^0.8.0;
                        
                        library Math {
                            function add(uint256 a, uint256 b) internal pure returns (uint256) {
                                return a + b;
                            }
                        }`,
                    }
                }
            }
            
            // Handle node_modules style imports
            if strings.HasPrefix(url, "@openzeppelin/") {
                // Simulate reading from node_modules
                switch url {
                case "@openzeppelin/contracts/utils/Context.sol":
                    return solc.ImportResult{
                        Contents: `pragma solidity ^0.8.0;
                        
                        abstract contract Context {
                            function _msgSender() internal view virtual returns (address) {
                                return msg.sender;
                            }
                        }`,
                    }
                }
            }
            
            // Return error for unresolved imports
            return solc.ImportResult{
                Error: fmt.Sprintf("File not found: %s", url),
            }
        },
    }

    // Compile with import resolution
    output, err := compiler.CompileWithOptions(input, options)
    if err != nil {
        log.Fatalf("Compilation failed: %v", err)
    }

    fmt.Printf("Compilation successful! Contracts: %v\n", len(output.Contracts))
}
```

##### File System Import Example

Here's a more practical example that reads imports from the file system:

```go
// File system based import resolver
func createFileSystemImportCallback(basePath string) solc.ImportCallback {
    return func(url string) solc.ImportResult {
        var fullPath string
        
        // Handle relative imports
        if strings.HasPrefix(url, "./") || strings.HasPrefix(url, "../") {
            fullPath = filepath.Join(basePath, url)
        } else {
            // Handle node_modules style imports
            fullPath = filepath.Join(basePath, "node_modules", url)
        }
        
        content, err := os.ReadFile(fullPath)
        if err != nil {
            return solc.ImportResult{
                Error: fmt.Sprintf("Failed to read %s: %v", url, err),
            }
        }
        
        return solc.ImportResult{
            Contents: string(content),
        }
    }
}

// Usage
options := &solc.CompileOptions{
    ImportCallback: createFileSystemImportCallback("./contracts"),
}
output, err := compiler.CompileWithOptions(input, options)
```

#### Features

- **Dynamic Version Support**: Specify any Solidity version (e.g., "0.8.30", "0.7.6", "0.6.12")
- **Import Resolution**: Custom import callbacks for resolving external dependencies, libraries, and node_modules
- **Embedded Binaries**: Latest Solidity (0.8.30) and LTS (0.8.21) versions are pre-embedded for instant access
- **Automatic Fallback**: Other versions are downloaded automatically from the official Solidity repository
- **Weekly Updates**: Embedded binaries are automatically updated every Sunday via CI/CD
- **Full Compatibility**: Supports all Solidity compiler features and output options

#### Import Callback Interface

The `ImportCallback` function signature is:

```go
type ImportCallback func(url string) ImportResult

type ImportResult struct {
    Contents string `json:"contents,omitempty"` // File contents if successful
    Error    string `json:"error,omitempty"`    // Error message if failed
}
```

**Import Resolution Process:**
1. When the compiler encounters an `import` statement, it calls your `ImportCallback` with the import URL
2. Your callback should return either the file contents or an error message
3. The compiler includes the resolved content in the compilation
4. Supports any import pattern: relative paths (`./lib/Math.sol`), absolute paths, or package imports (`@openzeppelin/...`)

#### Performance

For the best performance, use the embedded versions:
- **0.8.30** (latest) - instant compilation, no download required
- **0.8.21** (LTS) - instant compilation, no download required

Other versions will be downloaded on first use and may take a moment depending on network speed.
