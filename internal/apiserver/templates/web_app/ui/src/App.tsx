import { useState } from 'react'

interface ApiResponse {
  result: unknown
  module: string
  func: string
  elapsed_ms: number
  error?: string
}

function App() {
  const [name, setName] = useState('')
  const [response, setResponse] = useState<ApiResponse | null>(null)
  const [loading, setLoading] = useState(false)

  const callApi = async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/api/handlers/hello', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ args: [name || 'World'] }),
      })
      const data = await res.json()
      setResponse(data)
    } catch (err) {
      setResponse({ result: null, module: '', func: '', elapsed_ms: 0, error: String(err) })
    }
    setLoading(false)
  }

  return (
    <div style={{ fontFamily: 'system-ui', maxWidth: 600, margin: '2rem auto', padding: '0 1rem' }}>
      <h1>AILANG Web App</h1>
      <p>This React app calls AILANG functions via the auto-generated REST API.</p>

      <div style={{ display: 'flex', gap: '0.5rem', marginTop: '1rem' }}>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Enter a name"
          style={{ padding: '0.5rem', flex: 1, fontSize: '1rem' }}
          onKeyDown={(e) => e.key === 'Enter' && callApi()}
        />
        <button
          onClick={callApi}
          disabled={loading}
          style={{ padding: '0.5rem 1rem', fontSize: '1rem' }}
        >
          {loading ? 'Calling...' : 'Call hello()'}
        </button>
      </div>

      {response && (
        <pre style={{
          background: '#f5f5f5',
          padding: '1rem',
          borderRadius: '4px',
          marginTop: '1rem',
          overflow: 'auto',
        }}>
          {JSON.stringify(response, null, 2)}
        </pre>
      )}

      <details style={{ marginTop: '2rem' }}>
        <summary>API Info</summary>
        <p>
          The AILANG API server auto-generates REST endpoints from module exports.
          Click below to see all available endpoints.
        </p>
        <button onClick={async () => {
          const res = await fetch('/api/_meta/modules')
          const data = await res.json()
          setResponse(data as ApiResponse)
        }}>
          List all modules
        </button>
      </details>
    </div>
  )
}

export default App
