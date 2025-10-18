/**
 * AILANG REPL Component for Docusaurus
 *
 * Usage in MDX:
 * ```mdx
 * import AilangRepl from '@site/src/components/AilangRepl';
 *
 * <AilangRepl />
 * ```
 */

import React, { useEffect, useRef, useState } from 'react';

// Basic terminal-like styling (customize to match your theme)
const styles = {
  container: {
    backgroundColor: '#1e1e1e',
    color: '#d4d4d4',
    fontFamily: 'Consolas, Monaco, "Courier New", monospace',
    fontSize: '14px',
    padding: '16px',
    borderRadius: '8px',
    marginBottom: '20px',
    maxHeight: '600px',
    overflow: 'auto',
  },
  input: {
    backgroundColor: 'transparent',
    border: 'none',
    color: '#d4d4d4',
    fontFamily: 'inherit',
    fontSize: 'inherit',
    outline: 'none',
    width: '100%',
    padding: '4px 0',
  },
  output: {
    whiteSpace: 'pre-wrap',
    marginBottom: '8px',
  },
  prompt: {
    color: '#4ec9b0',
    marginRight: '8px',
  },
  error: {
    color: '#f48771',
  },
  success: {
    color: '#ce9178',
  },
  loading: {
    color: '#808080',
    fontStyle: 'italic',
  },
};

export default function AilangRepl() {
  const [repl, setRepl] = useState(null);
  const [history, setHistory] = useState([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const inputRef = useRef(null);
  const containerRef = useRef(null);

  // Initialize WASM REPL
  useEffect(() => {
    let mounted = true;

    async function initRepl() {
      try {
        // Wait a bit for scripts to fully execute
        await new Promise(resolve => setTimeout(resolve, 100));

        // Check if AilangREPL is available globally (loaded via script tag)
        if (typeof window.AilangREPL === 'undefined') {
          throw new Error('AilangREPL not loaded. Make sure ailang-repl.js is loaded.');
        }

        const replInstance = new window.AilangREPL();
        await replInstance.init('/ailang/wasm/ailang.wasm');

        if (mounted) {
          setRepl(replInstance);
          setLoading(false);

          // Get version from WASM
          const versionInfo = window.ailangVersion ? window.ailangVersion() : { version: 'dev' };
          const version = versionInfo.version || 'dev';

          // Add welcome message with dynamic version
          setHistory([
            { type: 'output', content: `AILANG ${version} - WebAssembly REPL` },
            { type: 'output', content: 'Type :help for help, or try: 1 + 2' },
            { type: 'output', content: '' },
          ]);
        }
      } catch (err) {
        console.error('Failed to initialize AILANG REPL:', err);
        if (mounted) {
          setError(err.message);
          setLoading(false);
        }
      }
    }

    initRepl();

    return () => {
      mounted = false;
    };
  }, []);

  // Auto-scroll to bottom when history updates
  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [history]);

  // Handle input submission
  const handleSubmit = (e) => {
    e.preventDefault();
    if (!input.trim() || !repl) return;

    // Add input to history
    const newHistory = [...history, { type: 'input', content: input }];

    // Evaluate
    const result = repl.eval(input);

    // Add result to history
    newHistory.push({ type: 'output', content: result });

    setHistory(newHistory);
    setInput('');

    // Focus back on input
    if (inputRef.current) {
      inputRef.current.focus();
    }
  };

  if (loading) {
    return (
      <div style={styles.container}>
        <div style={styles.loading}>Loading AILANG REPL...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div style={styles.container}>
        <div style={styles.error}>
          Error loading REPL: {error}
          <br />
          <br />
          Make sure you have:
          <br />
          1. Built the WASM file: <code>make build-wasm</code>
          <br />
          2. Copied ailang.wasm to your static/wasm/ directory
          <br />
          3. Included wasm_exec.js in your HTML
        </div>
      </div>
    );
  }

  return (
    <div style={styles.container} ref={containerRef}>
      {/* History */}
      {history.map((item, idx) => (
        <div key={idx} style={styles.output}>
          {item.type === 'input' && (
            <>
              <span style={styles.prompt}>λ&gt;</span>
              {item.content}
            </>
          )}
          {item.type === 'output' && item.content}
        </div>
      ))}

      {/* Input */}
      <form onSubmit={handleSubmit} style={{ display: 'flex' }}>
        <span style={styles.prompt}>λ&gt;</span>
        <input
          ref={inputRef}
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          style={styles.input}
          autoFocus
          placeholder="Enter AILANG code..."
          spellCheck={false}
        />
      </form>
    </div>
  );
}
