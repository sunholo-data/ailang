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
   * @returns {string|null} Version string (e.g., "v0.5.6") or null if not ready
   */
  getVersion() {
    if (!this.ready) {
      return null;
    }

    try {
      const info = window.ailangVersion();
      if (info && info.version) {
        // Ensure version starts with 'v'
        const ver = info.version;
        return ver.startsWith('v') ? ver : `v${ver}`;
      }
      return null;
    } catch (err) {
      return null;
    }
  }

  /**
   * Get full version info object
   * @returns {Object|null} Version info with version, buildTime, platform
   */
  getVersionInfo() {
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

  // ============================================
  // Module Loading API (v0.7.2+)
  // ============================================

  /**
   * Load an AILANG module into the registry
   * @param {string} name - Module name (e.g., "math", "invoice_processor")
   * @param {string} code - AILANG source code
   * @returns {{success: boolean, exports?: string[], error?: string}}
   */
  loadModule(name, code) {
    if (!this.ready) {
      return { success: false, error: 'REPL not initialized' };
    }

    try {
      return window.ailangLoadModule(name, code);
    } catch (err) {
      return { success: false, error: `Internal error: ${err.message}` };
    }
  }

  /**
   * List all loaded modules
   * @returns {string[]} Array of module names
   */
  listModules() {
    if (!this.ready) {
      return [];
    }

    try {
      return window.ailangListModules() || [];
    } catch (err) {
      console.error('Failed to list modules:', err);
      return [];
    }
  }

  /**
   * Import a module's exports into the REPL environment
   * @param {string} moduleName - Name of the loaded module
   * @returns {string} Import result message
   */
  importModule(moduleName) {
    return this.command(`:import ${moduleName}`);
  }

  /**
   * Call a function from a loaded module
   * This is a convenience method that:
   * 1. Imports the module if not already imported
   * 2. Evaluates the function call
   * @param {string} moduleName - Module containing the function
   * @param {string} funcName - Function to call
   * @param {...any} args - Arguments to pass (will be converted to AILANG syntax)
   * @returns {string} Result of the function call
   */
  call(moduleName, funcName, ...args) {
    if (!this.ready) {
      return 'Error: REPL not initialized';
    }

    // Check if module is loaded
    const modules = this.listModules();
    if (!modules.includes(moduleName)) {
      return `Error: module ${moduleName} not loaded (use loadModule first)`;
    }

    // Import the module to make exports available
    this.importModule(moduleName);

    // Build the function call expression
    // Convert JS args to AILANG syntax
    const ailangArgs = args.map(arg => {
      if (typeof arg === 'string') return `"${arg}"`;
      if (typeof arg === 'number') return String(arg);
      if (typeof arg === 'boolean') return arg ? 'true' : 'false';
      if (Array.isArray(arg)) return `[${arg.map(a => typeof a === 'string' ? `"${a}"` : String(a)).join(', ')}]`;
      return String(arg);
    });

    // Apply arguments one at a time (curried)
    let expr = funcName;
    for (const arg of ailangArgs) {
      expr = `(${expr})(${arg})`;
    }

    return this.eval(expr);
  }
}

// Make available globally (priority for browser usage)
if (typeof window !== 'undefined') {
  window.AilangREPL = AilangREPL;
  console.log('AilangREPL loaded and available globally');
}

// Export for use in modules (Node.js)
if (typeof module !== 'undefined' && module.exports) {
  module.exports = AilangREPL;
}
