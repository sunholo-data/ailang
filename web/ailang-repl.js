/**
 * AILANG WASM REPL Wrapper
 * Provides a clean JavaScript API for the AILANG REPL
 */

class AilangREPL {
  constructor() {
    this.ready = false;
    this.onReadyCallbacks = [];
    this._pendingHandlers = {};  // Accumulator for 3-arg setEffectHandler calls
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

  /**
   * Load an AILANG module into the registry (v0.7.2+)
   * @param {string} name - Module name (e.g., 'math', 'invoice_processor')
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
      return { success: false, error: err.message };
    }
  }

  /**
   * List all loaded modules (v0.7.2+)
   * @returns {string[]} Array of module names
   */
  listModules() {
    if (!this.ready) {
      return [];
    }

    try {
      return window.ailangListModules() || [];
    } catch (err) {
      return [];
    }
  }

  /**
   * Import a module's exports into the REPL environment (v0.7.2+)
   * @param {string} moduleName - Name of a loaded module
   * @returns {string} Import result message
   */
  importModule(moduleName) {
    if (!this.ready) {
      return 'Error: REPL not initialized';
    }

    // Check if module is loaded
    const modules = this.listModules();
    if (!modules.includes(moduleName)) {
      return `Error: module '${moduleName}' not loaded (use loadModule first)`;
    }

    // Use REPL's :import command
    return this.eval(`:import ${moduleName}`);
  }

  /**
   * Call a function from a loaded module (v0.7.2+)
   * Uses native ailangCall for direct function invocation.
   * @param {string} moduleName - Module containing the function
   * @param {string} funcName - Function to call
   * @param {...any} args - Arguments (numbers, strings, booleans)
   * @returns {{success: boolean, result?: string, error?: string}}
   */
  call(moduleName, funcName, ...args) {
    if (!this.ready) {
      return { success: false, error: 'REPL not initialized' };
    }

    try {
      // Use native ailangCall which handles type conversion
      return window.ailangCall(moduleName, funcName, ...args);
    } catch (err) {
      return { success: false, error: err.message };
    }
  }

  // ── Effect Handlers (v0.7.2+) ──────────────────────────────────

  /**
   * Register JS function(s) as effect handler(s) (v0.7.2+)
   *
   * Supports two calling conventions:
   *   2-arg: setEffectHandler('Stream', { connect: fn, send: fn, ... })
   *   3-arg: setEffectHandler('Stream', 'connect', fn)
   *
   * The 3-arg form accumulates handlers per capability and registers them
   * as a single object on each call, so multiple 3-arg calls build up the
   * full handler set. Auto-grants the capability.
   *
   * @param {string} capability - Effect capability (e.g., 'IO', 'Net', 'Stream')
   * @param {string|Object} operationOrHandlers - Operation name (3-arg) or handlers object (2-arg)
   * @param {Function} [handler] - JS callback for 3-arg form
   * @returns {{success: boolean, error?: string}}
   */
  setEffectHandler(capability, operationOrHandlers, handler) {
    if (!this.ready) {
      return { success: false, error: 'REPL not initialized' };
    }

    try {
      if (typeof operationOrHandlers === 'object' && operationOrHandlers !== null) {
        // 2-arg form: setEffectHandler('Stream', { connect: fn, send: fn })
        return window.ailangSetEffectHandler(capability, operationOrHandlers);
      }
      // 3-arg form: setEffectHandler('Stream', 'connect', fn)
      if (!this._pendingHandlers[capability]) {
        this._pendingHandlers[capability] = {};
      }
      this._pendingHandlers[capability][operationOrHandlers] = handler;
      return window.ailangSetEffectHandler(capability, this._pendingHandlers[capability]);
    } catch (err) {
      return { success: false, error: err.message };
    }
  }

  /**
   * Register a JS function as the AI completion handler (v0.7.2+)
   * @param {Function} handler - JS callback: (prompt) => response string
   * @returns {{success: boolean, error?: string}}
   */
  setAIHandler(handler) {
    if (!this.ready) {
      return { success: false, error: 'REPL not initialized' };
    }

    try {
      return window.ailangSetAIHandler(handler);
    } catch (err) {
      return { success: false, error: err.message };
    }
  }

  /**
   * Grant an effect capability to the REPL environment (v0.7.2+)
   * @param {string} capability - Capability to grant (e.g., 'IO', 'AI', 'Net')
   * @returns {{success: boolean, error?: string}}
   */
  grantCapability(capability) {
    if (!this.ready) {
      return { success: false, error: 'REPL not initialized' };
    }

    try {
      return window.ailangGrantCapability(capability);
    } catch (err) {
      return { success: false, error: err.message };
    }
  }

  // ── Async Methods (v0.7.2+) ────────────────────────────────────

  /**
   * Evaluate an expression asynchronously (v0.7.2+)
   * Required when the expression triggers effect handlers that return Promises.
   * @param {string} input - AILANG code to evaluate
   * @returns {Promise<string>} Result or error message
   */
  async evalAsync(input) {
    if (!this.ready) {
      throw new Error('REPL not initialized');
    }

    return window.ailangEvalAsync(input);
  }

  /**
   * Call a module function asynchronously (v0.7.2+)
   * Required when the function triggers effect handlers that return Promises.
   * @param {string} moduleName - Module containing the function
   * @param {string} funcName - Function to call
   * @param {...any} args - Arguments (numbers, strings, booleans)
   * @returns {Promise<{success: boolean, result?: string, error?: string}>}
   */
  async callAsync(moduleName, funcName, ...args) {
    if (!this.ready) {
      throw new Error('REPL not initialized');
    }

    return window.ailangCallAsync(moduleName, funcName, ...args);
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
