/**
 * AILANG WASM REPL Wrapper
 * Provides a clean JavaScript API for the AILANG REPL
 */

class AilangREPL {
  constructor() {
    this.ready = false;
    this.onReadyCallbacks = [];
  }

  /**
   * Initialize the WASM module
   * @param {string} wasmPath - Path to ailang.wasm file
   */
  async init(wasmPath = '/wasm/ailang.wasm') {
    if (!('WebAssembly' in window)) {
      throw new Error('WebAssembly not supported in this browser');
    }

    // Load Go's WASM support
    const go = new Go();

    try {
      const result = await WebAssembly.instantiateStreaming(
        fetch(wasmPath),
        go.importObject
      );

      // Run the Go program (this will register the functions)
      go.run(result.instance);

      this.ready = true;
      this.onReadyCallbacks.forEach(cb => cb());

      return this;
    } catch (err) {
      console.error('Failed to load AILANG WASM:', err);
      throw err;
    }
  }

  /**
   * Register callback for when REPL is ready
   */
  onReady(callback) {
    if (this.ready) {
      callback();
    } else {
      this.onReadyCallbacks.push(callback);
    }
  }

  /**
   * Evaluate an AILANG expression
   * @param {string} input - AILANG code to evaluate
   * @returns {string} Result or error message
   */
  eval(input) {
    if (!this.ready) {
      return 'Error: REPL not initialized';
    }

    try {
      return window.ailangEval(input);
    } catch (err) {
      return `Error: ${err.message}`;
    }
  }

  /**
   * Execute a REPL command (e.g., :type, :help)
   * @param {string} command - Command to execute
   * @returns {string} Command output
   */
  command(command) {
    return this.eval(command);
  }

  /**
   * Reset the REPL environment
   */
  reset() {
    if (!this.ready) {
      return 'Error: REPL not initialized';
    }

    try {
      return window.ailangReset();
    } catch (err) {
      return `Error: ${err.message}`;
    }
  }

  /**
   * Get version information
   */
  getVersion() {
    if (!this.ready) {
      return null;
    }

    try {
      return window.ailangVersion();
    } catch (err) {
      return null;
    }
  }

  /**
   * Check if a line needs continuation (for multi-line input)
   */
  needsContinuation(line) {
    return line.trim().endsWith('in') ||
           line.trim().endsWith('let') ||
           line.trim().endsWith('=');
  }
}

// Export for use in modules
if (typeof module !== 'undefined' && module.exports) {
  module.exports = AilangREPL;
}

// Also make available globally
if (typeof window !== 'undefined') {
  window.AilangREPL = AilangREPL;
}
