package solc

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"rogchap.com/v8go"
)

// ImportResult represents the result of an import callback.
type ImportResult struct {
	// Contents holds the file contents if import was successful.
	Contents string `json:"contents,omitempty"`
	// Error holds the error message if import failed.
	Error string `json:"error,omitempty"`
}

// ImportCallback is a function that resolves import statements.
// It receives the import URL and returns the file contents or an error.
type ImportCallback func(url string) ImportResult

// CompileOptions holds additional options for compilation.
type CompileOptions struct {
	// ImportCallback handles import resolution.
	ImportCallback ImportCallback
}

// Solc represents a Solidity compiler interface.
type Solc interface {
	// License returns the license information of the compiler.
	License() string
	// Version returns the version information of the compiler.
	Version() string
	// CompileWithOptions compiles Solidity source code with additional options like import callbacks.
	// Pass nil for options to use default compilation without import callbacks.
	CompileWithOptions(input *Input, options *CompileOptions) (*Output, error)
	// Close releases all resources associated with the compiler instance.
	Close() error
}

// baseSolc implements the Solc interface using V8 JavaScript engine.
type baseSolc struct {
	isolate *v8go.Isolate
	ctx     *v8go.Context

	// mu protects the underlying v8 context from concurrent access
	mu sync.Mutex

	version *v8go.Function
	license *v8go.Function

	closed bool
}

// New creates a new Solc binding using the provided soljson.js emscripten binary.
func New(soljsonjs string) (Solc, error) {
	return newBaseSolc(soljsonjs)
}

// newBaseSolc creates and initializes a new baseSolc instance.
func newBaseSolc(soljsonjs string) (*baseSolc, error) {
	if soljsonjs == "" {
		return nil, fmt.Errorf("soljsonjs cannot be empty")
	}
	// Create v8go JS execution context
	isolate := v8go.NewIsolate()
	ctx := v8go.NewContext(isolate)

	// Create Solc object
	solc := &baseSolc{
		isolate: isolate,
		ctx:     ctx,
	}

	// Initialize solc
	if err := solc.init(soljsonjs); err != nil {
		solc.cleanup()
		return nil, fmt.Errorf("failed to initialize compiler: %w", err)
	}

	return solc, nil
}

// init initializes the Solidity compiler by executing the soljson.js script
// and binding the necessary functions.
func (s *baseSolc) init(soljsonjs string) error {
	// Execute soljson.js script
	if _, err := s.ctx.RunScript(soljsonjs, "soljson.js"); err != nil {
		return fmt.Errorf("failed to execute soljson.js: %w", err)
	}

	// Set up debug logging function
	debugLogFunc := v8go.NewFunctionTemplate(s.isolate, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			fmt.Printf("JS DEBUG: %s\n", args[0].String())
		}
		return v8go.Undefined(s.isolate)
	})

	debugLog := debugLogFunc.GetFunction(s.ctx)
	if err := s.ctx.Global().Set("debugLog", debugLog); err != nil {
		return fmt.Errorf("failed to set debug function: %w", err)
	}

	// Bind version function
	versionFunc := "version"
	if strings.Contains(soljsonjs, "_solidity_version") {
		versionFunc = "solidity_version"
	}
	var err error
	versionVal, err := s.ctx.RunScript(fmt.Sprintf("Module.cwrap('%s', 'string', [])", versionFunc), "wrap_version.js")
	if err != nil {
		return fmt.Errorf("failed to bind version function: %w", err)
	}
	s.version, err = versionVal.AsFunction()
	if err != nil {
		return fmt.Errorf("version binding is not a function: %w", err)
	}

	// Bind license function
	if strings.Contains(soljsonjs, "_solidity_license") {
		licenseVal, err := s.ctx.RunScript("Module.cwrap('solidity_license', 'string', [])", "wrap_license.js")
		if err != nil {
			return fmt.Errorf("failed to bind license function: %w", err)
		}
		s.license, err = licenseVal.AsFunction()
		if err != nil {
			return fmt.Errorf("license binding is not a function: %w", err)
		}
	} else if strings.Contains(soljsonjs, "_license") {
		licenseVal, err := s.ctx.RunScript("Module.cwrap('license', 'string', [])", "wrap_license.js")
		if err != nil {
			return fmt.Errorf("failed to bind license function: %w", err)
		}
		s.license, err = licenseVal.AsFunction()
		if err != nil {
			return fmt.Errorf("license binding is not a function: %w", err)
		}
	}

	// Implement the correct solc.js-compatible wrapper
	setupScript := `
		// Create the core compile function binding
		var nativeCompile = Module.cwrap('solidity_compile', 'string', ['string']);
		debugLog('DEBUG: Native compile function bound successfully');
		
		// Implement solc.js wrapper with iterative import resolution
		function wrapper(input, importCallback, smtSolverCallback) {
			debugLog('DEBUG: Wrapper called with importCallback: ' + (importCallback ? 'present' : 'null'));
			
			if (!importCallback) {
				debugLog('DEBUG: No import callback, direct compilation');
				return nativeCompile(input);
			}
			
			// Parse the input to work with it
			var inputObj;
			try {
				inputObj = JSON.parse(input);
			} catch (e) {
				debugLog('DEBUG: Failed to parse input JSON: ' + e.toString());
				return JSON.stringify({errors: [{severity: "error", formattedMessage: "Invalid input JSON"}]});
			}
			
			// Iteratively resolve imports by compiling and handling import errors
			var maxIterations = 10;
			var iteration = 0;
			
			while (iteration < maxIterations) {
				debugLog('DEBUG: Compilation iteration ' + (iteration + 1));
				
				// Try compilation
				var result = nativeCompile(JSON.stringify(inputObj));
				var resultObj;
				
				try {
					resultObj = JSON.parse(result);
				} catch (e) {
					debugLog('DEBUG: Failed to parse result JSON: ' + e.toString());
					return result; // Return as-is if we can't parse it
				}
				
				// Check if we have import errors
				var hasImportErrors = false;
				var importToResolve = null;
				
				if (resultObj.errors) {
					for (var i = 0; i < resultObj.errors.length; i++) {
						var error = resultObj.errors[i];
						if (error.type === 'ParserError' && error.message && 
							error.message.indexOf('not found') !== -1 && 
							error.message.indexOf('File not supplied initially') !== -1) {
							
							// Extract import path from error message or source location
							var match = error.formattedMessage.match(/import\s+.*?from\s+["']([^"']+)["']/);
							if (match) {
								importToResolve = match[1];
								hasImportErrors = true;
								debugLog('DEBUG: Found import to resolve: ' + importToResolve);
								break;
							}
						}
					}
				}
				
				if (!hasImportErrors || !importToResolve) {
					debugLog('DEBUG: No more import errors, returning result');
					return result;
				}
				
				// Try to resolve the import
				debugLog('DEBUG: Resolving import: ' + importToResolve);
				try {
					var importResult = importCallback(importToResolve);
					debugLog('DEBUG: Import resolved: ' + JSON.stringify(importResult));
					
					if (importResult.error) {
						debugLog('DEBUG: Import resolution failed: ' + importResult.error);
						return result; // Return the compilation result with error
					}
					
					if (importResult.contents) {
						// Add the resolved source to our input
						if (!inputObj.sources) {
							inputObj.sources = {};
						}
						inputObj.sources[importToResolve] = { content: importResult.contents };
						debugLog('DEBUG: Added resolved source for: ' + importToResolve);
					}
				} catch (e) {
					debugLog('DEBUG: Import callback threw error: ' + e.toString());
					return result; // Return the compilation result with error
				}
				
				iteration++;
			}
			
			debugLog('DEBUG: Max iterations reached, returning last result');
			return nativeCompile(JSON.stringify(inputObj));
		}
		
		// Create solc interface using the wrapper
		var solc = {
			compile: function(input, callbacks) {
				debugLog('DEBUG: solc.compile called with callbacks: ' + (callbacks ? 'present' : 'null'));
				
				var importCallback = null;
				if (callbacks && callbacks.import && typeof callbacks.import === 'function') {
					importCallback = callbacks.import;
					debugLog('DEBUG: Import callback extracted from callbacks object');
				}
				
				// Use the wrapper function
				return wrapper(input, importCallback, null);
			}
		};
		
		globalThis.solc = solc;
		globalThis.compileWithImports = function(inputJson, importCallback) {
			debugLog('DEBUG: compileWithImports called');
			return solc.compile(inputJson, importCallback ? { import: importCallback } : null);
		};
		
		solc;
	`

	_, err = s.ctx.RunScript(setupScript, "compile_wrapper.js")
	if err != nil {
		return fmt.Errorf("failed to create compile wrapper: %w", err)
	}

	// Validate that the setup worked by checking if solc is available
	solcVal, err := s.ctx.Global().Get("solc")
	if err != nil {
		return fmt.Errorf("solc object not created: %w", err)
	}

	if solcVal.IsUndefined() || solcVal.IsNull() {
		return fmt.Errorf("solc object is undefined or null")
	}

	return nil
}

// Close releases all resources associated with the compiler instance.
func (s *baseSolc) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.cleanup()
	s.closed = true
	return nil
}

// cleanup releases V8 resources without acquiring the mutex.
func (s *baseSolc) cleanup() {
	if s.ctx != nil {
		s.ctx.Close()
		s.ctx = nil
	}
	if s.isolate != nil {
		s.isolate.Dispose()
		s.isolate = nil
	}
}

// License returns the license information of the compiler.
func (s *baseSolc) License() string {
	if s.license == nil {
		return ""
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ""
	}

	val, err := s.license.Call(v8go.Undefined(s.ctx.Isolate()))
	if err != nil {
		return ""
	}
	return val.String()
}

// Version returns the version information of the compiler.
func (s *baseSolc) Version() string {
	if s.version == nil {
		return ""
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ""
	}

	val, err := s.version.Call(v8go.Undefined(s.ctx.Isolate()))
	if err != nil {
		return ""
	}
	return val.String()
}

// CompileWithOptions compiles Solidity source code with additional options like import callbacks.
func (s *baseSolc) CompileWithOptions(input *Input, options *CompileOptions) (*Output, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	// Marshal Solc Compiler Input
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Run Compilation
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("compiler has been closed")
	}

	// Get the compile wrapper function
	compileWrapperVal, err := s.ctx.Global().Get("compileWithImports")
	if err != nil {
		return nil, fmt.Errorf("compile wrapper not available: %w", err)
	}

	compileWrapper, err := compileWrapperVal.AsFunction()
	if err != nil {
		return nil, fmt.Errorf("compile wrapper is not a function: %w", err)
	}

	// Create input value
	valInput, err := v8go.NewValue(s.ctx.Isolate(), string(inputJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create input value: %w", err)
	}

	var valOutput *v8go.Value

	// Check if we have an import callback
	if options != nil && options.ImportCallback != nil {
		// Create JavaScript callback function that calls back to Go
		callbackFunc := v8go.NewFunctionTemplate(s.ctx.Isolate(), func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) < 1 {
				fmt.Printf("DEBUG: Import callback called with no arguments\n")
				// Create error object
				errorObj, _ := s.ctx.RunScript(`({"error": "No import path provided"})`, "error_obj.js")
				return errorObj
			}

			importPath := args[0].String()
			result := options.ImportCallback(importPath)
			var responseScript string
			if result.Error != "" {
				responseScript = fmt.Sprintf(`({"error": %q})`, result.Error)
			} else {
				responseScript = fmt.Sprintf(`({"contents": %q})`, result.Contents)
			}

			responseVal, err := s.ctx.RunScript(responseScript, "callback_response.js")
			if err != nil {
				errorObj, _ := s.ctx.RunScript(`({"error": "Failed to create response"})`, "error_obj.js")
				return errorObj
			}
			return responseVal
		})

		callbackInstance := callbackFunc.GetFunction(s.ctx)

		// Call the compile wrapper with the callback
		valOutput, err = compileWrapper.Call(v8go.Undefined(s.ctx.Isolate()), valInput, callbackInstance)
		if err != nil {
			return nil, fmt.Errorf("compilation with import callback failed: %w", err)
		}
	} else {
		// Standard compilation without import callback
		valOutput, err = compileWrapper.Call(v8go.Undefined(s.ctx.Isolate()), valInput, v8go.Null(s.ctx.Isolate()))
		if err != nil {
			return nil, fmt.Errorf("compilation failed: %w", err)
		}
	}

	output := &Output{}
	if err := json.Unmarshal([]byte(valOutput.String()), output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %w", err)
	}

	return output, nil
}
