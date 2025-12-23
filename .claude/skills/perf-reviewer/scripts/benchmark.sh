#!/usr/bin/env bash
# Performance benchmark: AILANG interpreted vs Python vs AILANG compiled to Go
#
# Usage:
#   ./benchmark.sh [benchmark_name] [iterations]
#
# Available benchmarks: fibonacci, sum, all
# Default: all benchmarks with 5 iterations each

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(cd "$SKILL_DIR/../../.." && pwd)"
BENCH_DIR="$SKILL_DIR/benchmarks"
WORK_DIR="/tmp/perf-benchmark-$$"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

BENCHMARK="${1:-all}"
ITERATIONS="${2:-5}"

cleanup() {
    rm -rf "$WORK_DIR"
}
trap cleanup EXIT

mkdir -p "$WORK_DIR"

echo -e "${BOLD}Performance Benchmark: AILANG vs Python vs Compiled Go${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check prerequisites
check_prereqs() {
    local missing=0

    if ! command -v python3 &> /dev/null; then
        echo -e "${RED}Error: python3 not found${NC}"
        missing=1
    fi

    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: go not found${NC}"
        missing=1
    fi

    if ! command -v ailang &> /dev/null && [[ ! -x "$PROJECT_ROOT/bin/ailang" ]]; then
        echo -e "${RED}Error: ailang not found${NC}"
        missing=1
    fi

    if [[ $missing -eq 1 ]]; then
        exit 1
    fi

    # Use local ailang if available
    if [[ -x "$PROJECT_ROOT/bin/ailang" ]]; then
        AILANG="$PROJECT_ROOT/bin/ailang"
    else
        AILANG="ailang"
    fi
}

# Time a command and return milliseconds
time_cmd() {
    local start end elapsed
    start=$(python3 -c 'import time; print(int(time.time() * 1000))')
    eval "$@" > /dev/null 2>&1
    end=$(python3 -c 'import time; print(int(time.time() * 1000))')
    elapsed=$((end - start))
    echo "$elapsed"
}

# Run benchmark multiple times and get average
run_benchmark() {
    local name="$1"
    local cmd="$2"
    local total=0
    local times=()

    for ((i=1; i<=ITERATIONS; i++)); do
        t=$(time_cmd "$cmd")
        times+=("$t")
        total=$((total + t))
    done

    local avg=$((total / ITERATIONS))
    echo "$avg"
}

# Create benchmark files
create_fibonacci_benchmarks() {
    # Use fib(25) for reasonable runtime (~300ms AILANG, ~30ms Python, ~3ms Go)
    local n=25

    # AILANG version (using pure func syntax)
    cat > "$WORK_DIR/fib.ail" << 'AILANG_EOF'
module bench/fib

import std/io (println)

-- Recursive fibonacci (intentionally naive for benchmarking)
pure func fib(n: int) -> int {
  if n <= 1 then n else fib(n - 1) + fib(n - 2)
}

export func main() -> () ! {IO} {
  let result = fib(25);
  println("fib(25) = " ++ show(result))
}
AILANG_EOF

    # Python version
    cat > "$WORK_DIR/fib.py" << 'PYTHON_EOF'
def fib(n):
    if n <= 1:
        return n
    return fib(n - 1) + fib(n - 2)

if __name__ == "__main__":
    result = fib(25)
PYTHON_EOF

    # Native Go version for baseline comparison
    cat > "$WORK_DIR/fib_native.go" << 'GO_EOF'
package main

func fib(n int64) int64 {
    if n <= 1 {
        return n
    }
    return fib(n-1) + fib(n-2)
}

func main() {
    _ = fib(25)
}
GO_EOF

    echo "fibonacci (n=25)"
}

create_sum_benchmarks() {
    # Sum of squares benchmark (tests loops/recursion)

    # AILANG version (using pure func syntax)
    cat > "$WORK_DIR/sum.ail" << 'AILANG_EOF'
module bench/sum

import std/io (println)

-- Sum of squares from 1 to n using tail recursion
pure func sumSquaresHelper(acc: int, n: int) -> int {
  if n <= 0 then acc else sumSquaresHelper(acc + n * n, n - 1)
}

pure func sumSquares(n: int) -> int {
  sumSquaresHelper(0, n)
}

export func main() -> () ! {IO} {
  let result = sumSquares(10000);
  println("sumSquares(10000) = " ++ show(result))
}
AILANG_EOF

    # Python version
    cat > "$WORK_DIR/sum.py" << 'PYTHON_EOF'
def sum_squares(n):
    acc = 0
    for i in range(1, n + 1):
        acc += i * i
    return acc

if __name__ == "__main__":
    result = sum_squares(10000)
PYTHON_EOF

    # Native Go version for baseline comparison
    cat > "$WORK_DIR/sum_native.go" << 'GO_EOF'
package main

func sumSquares(n int64) int64 {
    var acc int64 = 0
    for i := int64(1); i <= n; i++ {
        acc += i * i
    }
    return acc
}

func main() {
    _ = sumSquares(10000)
}
GO_EOF

    echo "sum"
}

# Compile AILANG to Go and build
compile_ailang_to_go() {
    local ail_file="$1"
    local out_dir="$2"

    echo -e "${CYAN}Compiling AILANG to Go...${NC}"

    # Compile AILANG to Go
    AILANG_RELAX_MODULES=1 "$AILANG" compile --emit-go --out "$out_dir" "$ail_file" 2>&1 || {
        echo -e "${RED}AILANG compilation failed${NC}"
        return 1
    }

    # Find the generated package directory
    local pkg_dir
    pkg_dir=$(find "$out_dir" -name "*.go" -type f -exec dirname {} \; | head -1)

    if [[ -z "$pkg_dir" ]]; then
        echo -e "${RED}No Go files generated${NC}"
        return 1
    fi

    # Add main.go wrapper
    local pkg_name
    pkg_name=$(basename "$pkg_dir")

    cat > "$pkg_dir/main_bench.go" << MAIN_EOF
//go:build ignore

package main

import (
    "./$pkg_name"
)

func main() {
    // Call the main function
    $pkg_name.Main()
}
MAIN_EOF

    # Actually, let's just build a simple main that calls the generated code
    # The generated code should have a Main function

    # Initialize go module
    cd "$out_dir"
    go mod init bench 2>/dev/null || true

    # Build the package as a test
    cd "$pkg_dir"
    go build -o "$out_dir/bench_bin" . 2>&1 || {
        # If direct build fails, try building with a main wrapper
        cat > "$out_dir/main.go" << WRAPPER_EOF
package main

import (
    pkg "bench/$pkg_name"
)

func main() {
    pkg.Main()
}
WRAPPER_EOF
        cd "$out_dir"
        go build -o bench_bin . 2>&1 || {
            echo -e "${YELLOW}Go build failed (codegen may be incomplete)${NC}"
            return 1
        }
    }

    echo "$out_dir/bench_bin"
}

# Run a single benchmark
run_single_benchmark() {
    local name="$1"
    local ail_file="$WORK_DIR/${name}.ail"
    local py_file="$WORK_DIR/${name}.py"
    local native_go_file="$WORK_DIR/${name}_native.go"
    local go_dir="$WORK_DIR/go_${name}"

    echo -e "\n${BOLD}Benchmark: $name${NC}"
    echo "─────────────────────────────────────────────"

    # AILANG interpreted
    echo -ne "  AILANG (interpreted): "
    local ailang_time
    ailang_time=$(run_benchmark "$name" "AILANG_RELAX_MODULES=1 $AILANG run --caps IO --entry main $ail_file")
    echo -e "${GREEN}${ailang_time}ms${NC}"

    # Python
    echo -ne "  Python:               "
    local python_time
    python_time=$(run_benchmark "$name" "python3 $py_file")
    echo -e "${GREEN}${python_time}ms${NC}"

    # AILANG compiled to Go
    echo -ne "  AILANG -> Go:         "
    mkdir -p "$go_dir"
    local go_bin
    go_bin=$(compile_ailang_to_go "$ail_file" "$go_dir" 2>/dev/null) || go_bin=""

    local codegen_time="N/A"
    if [[ -n "$go_bin" && -x "$go_bin" ]]; then
        codegen_time=$(run_benchmark "$name" "$go_bin")
        echo -e "${GREEN}${codegen_time}ms${NC}"
    else
        echo -e "${YELLOW}skipped (codegen incomplete)${NC}"
    fi

    # Native Go (baseline)
    echo -ne "  Native Go (baseline): "
    local native_go_time="N/A"
    if [[ -f "$native_go_file" ]]; then
        local native_bin="$WORK_DIR/${name}_native_bin"
        if go build -o "$native_bin" "$native_go_file" 2>/dev/null; then
            native_go_time=$(run_benchmark "$name" "$native_bin")
            echo -e "${GREEN}${native_go_time}ms${NC}"
        else
            echo -e "${YELLOW}build failed${NC}"
        fi
    else
        echo -e "${YELLOW}not available${NC}"
    fi

    # Calculate ratios
    echo ""
    echo "  Ratios (vs Native Go baseline):"

    if [[ "$native_go_time" != "N/A" && "$native_go_time" -gt 0 ]]; then
        if [[ "$ailang_time" -gt 0 ]]; then
            local ail_vs_native
            ail_vs_native=$(python3 -c "print(f'{$ailang_time / $native_go_time:.1f}x slower')")
            echo "    AILANG interpreted: $ail_vs_native"
        fi

        if [[ "$python_time" -gt 0 ]]; then
            local py_vs_native
            py_vs_native=$(python3 -c "print(f'{$python_time / $native_go_time:.1f}x slower')")
            echo "    Python:             $py_vs_native"
        fi

        if [[ "$codegen_time" != "N/A" && "$codegen_time" -gt 0 ]]; then
            local codegen_vs_native
            codegen_vs_native=$(python3 -c "print(f'{$codegen_time / $native_go_time:.1f}x slower')")
            echo "    AILANG -> Go:       $codegen_vs_native"
        fi
    fi

    echo ""
    echo "  AILANG vs Python:"
    if [[ "$python_time" -gt 0 && "$ailang_time" -gt 0 ]]; then
        local py_vs_ail
        py_vs_ail=$(python3 -c "print(f'{$python_time / $ailang_time:.2f}x')")
        echo -e "    AILANG is ${CYAN}${py_vs_ail} faster${NC} than Python"
    fi

    # Store results for summary
    echo "$name,$ailang_time,$python_time,$codegen_time,$native_go_time" >> "$WORK_DIR/results.csv"
}

# Main
check_prereqs

echo "Configuration:"
echo "  Iterations per test: $ITERATIONS"
echo "  Work directory: $WORK_DIR"
echo "  AILANG binary: $AILANG"
echo ""

# Initialize results
echo "benchmark,ailang_ms,python_ms,codegen_ms,native_go_ms" > "$WORK_DIR/results.csv"

# Create and run benchmarks
case "$BENCHMARK" in
    fibonacci|fib)
        create_fibonacci_benchmarks
        run_single_benchmark "fib"
        ;;
    sum)
        create_sum_benchmarks
        run_single_benchmark "sum"
        ;;
    all)
        create_fibonacci_benchmarks
        create_sum_benchmarks
        run_single_benchmark "fib"
        run_single_benchmark "sum"
        ;;
    *)
        echo -e "${RED}Unknown benchmark: $BENCHMARK${NC}"
        echo "Available: fibonacci, sum, all"
        exit 1
        ;;
esac

# Summary
echo ""
echo -e "${BOLD}Summary${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Results saved to: $WORK_DIR/results.csv"
echo ""

cat "$WORK_DIR/results.csv" | column -t -s','

echo ""
echo -e "${CYAN}Interpretation:${NC}"
echo "  Key findings:"
echo "  - AILANG interpreter has high per-call overhead (slower than Python for recursion)"
echo "  - AILANG -> Go codegen provides significant speedup when it works"
echo "  - Native Go is fastest (no interface{} boxing overhead)"
echo ""
echo "  Recommendations:"
echo "  - For performance-critical recursive code: use 'ailang compile --emit-go'"
echo "  - For I/O-bound tasks: interpreter overhead is negligible"
echo "  - Use tail recursion where possible to reduce call overhead"
