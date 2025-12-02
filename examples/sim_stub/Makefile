# Sim Stub Example Makefile
# Demonstrates the AILANG -> Go code generation workflow

.PHONY: all generate build run test clean

# Default target
all: generate build

# Generate Go code from AILANG source
generate:
	@echo "Generating Go code from world.ail..."
	ailang compile --emit-go --out gen --package-name game world.ail

# Build the example
build: generate
	@echo "Building sim_stub..."
	go build -o sim_stub .

# Run the example
run: build
	./sim_stub

# Test deterministic output
test: build
	@echo "Testing deterministic output..."
	./sim_stub > actual_output.txt
	diff expected_output.txt actual_output.txt && echo "PASS: Output matches expected" || (echo "FAIL: Output differs"; exit 1)

# Clean generated files
clean:
	rm -rf gen/
	rm -f sim_stub
	rm -f actual_output.txt
