# Web API Demo

Demonstrates `ailang serve-api` — auto-generating REST endpoints from AILANG module exports.

## Structure

```
web_api_demo/
├── api/
│   ├── math.ail     # Pure math functions (add, multiply, factorial, fibonacci)
│   └── greet.ail    # String/JSON functions (hello, farewell, welcome)
├── test.sh          # Automated test script
└── README.md
```

## Running

```bash
# Start the API server
ailang serve-api examples/web_api_demo/api/ --port 8080

# In another terminal, try the endpoints:
curl -X POST http://localhost:8080/api/api/math/add \
  -H "Content-Type: application/json" \
  -d '{"args": [3, 4]}'
# {"result":7,"module":"api/math","func":"add","elapsed_ms":12}

curl -X POST http://localhost:8080/api/api/greet/hello \
  -H "Content-Type: application/json" \
  -d '{"args": ["World"]}'
# {"result":"Hello, World!","module":"api/greet","func":"hello","elapsed_ms":0}

# List all available modules and exports
curl http://localhost:8080/api/_meta/modules

# Health check
curl http://localhost:8080/api/_health
```

## Automated Tests

```bash
./examples/web_api_demo/test.sh
```

Starts the server, runs all endpoint tests, and reports pass/fail.

## Endpoints Generated

### api/math

| Endpoint | Body | Result |
|----------|------|--------|
| `POST /api/api/math/add` | `{"args": [3, 4]}` | `7` |
| `POST /api/api/math/multiply` | `{"args": [5, 6]}` | `30` |
| `POST /api/api/math/factorial` | `{"args": [5]}` | `120` |
| `POST /api/api/math/fibonacci` | `{"args": [10]}` | `55` |

### api/greet

| Endpoint | Body | Result |
|----------|------|--------|
| `POST /api/api/greet/hello` | `{"args": ["World"]}` | `"Hello, World!"` |
| `POST /api/api/greet/farewell` | `{"args": ["Alice"]}` | `"Goodbye, Alice. Until next time!"` |
| `POST /api/api/greet/welcome` | `{"args": ["Bob"]}` | `{"message":"Welcome, Bob!","name":"Bob"}` |
