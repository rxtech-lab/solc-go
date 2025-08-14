package solc

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"rogchap.com/v8go"
)

// Solc represents a Solidity compiler interface.
type Solc interface {
	// License returns the license information of the compiler.
	License() string
	// Version returns the version information of the compiler.
	Version() string
	// Compile compiles Solidity source code with the given input configuration.
	Compile(input *Input) (*Output, error)
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
	compile *v8go.Function

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

	// Bind compile function
	compileVal, err := s.ctx.RunScript("Module.cwrap('solidity_compile', 'string', ['string', 'number', 'number'])", "wrap_compile.js")
	if err != nil {
		return fmt.Errorf("failed to bind compile function: %w", err)
	}
	s.compile, err = compileVal.AsFunction()
	if err != nil {
		return fmt.Errorf("compile binding is not a function: %w", err)
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

// Compile compiles Solidity source code with the given input configuration.
func (s *baseSolc) Compile(input *Input) (*Output, error) {
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

	if s.compile == nil {
		return nil, fmt.Errorf("compile function not available")
	}

	valInput, err := v8go.NewValue(s.ctx.Isolate(), string(inputJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create input value: %w", err)
	}

	valOne, err := v8go.NewValue(s.ctx.Isolate(), int32(1))
	if err != nil {
		return nil, fmt.Errorf("failed to create parameter value: %w", err)
	}

	valOutput, err := s.compile.Call(v8go.Undefined(s.ctx.Isolate()), valInput, valOne, valOne)
	if err != nil {
		return nil, fmt.Errorf("compilation failed: %w", err)
	}

	output := &Output{}
	if err := json.Unmarshal([]byte(valOutput.String()), output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %w", err)
	}

	return output, nil
}
