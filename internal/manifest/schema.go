// Package manifest provides the JSON schema definition for AILANG manifests.
package manifest

// ManifestSchemaJSON defines the JSON schema for ailang.manifest/v1
const ManifestSchemaJSON = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "ailang.manifest/v1",
  "title": "AILANG Example Manifest",
  "description": "Manifest for tracking example status and validation",
  "type": "object",
  "required": ["schema", "examples", "statistics"],
  "additionalProperties": false,
  "properties": {
    "schema": {
      "type": "string",
      "const": "ailang.manifest/v1",
      "description": "Schema identifier"
    },
    "schema_version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+\\.\\d+$",
      "description": "Schema semantic version"
    },
    "schema_digest": {
      "type": "string",
      "pattern": "^sha256:[a-f0-9]{16}",
      "description": "Schema integrity digest"
    },
    "generated_at": {
      "type": "string",
      "format": "date-time",
      "description": "Timestamp of manifest generation"
    },
    "generator": {
      "type": "string",
      "description": "Tool that generated the manifest"
    },
    "examples": {
      "type": "array",
      "description": "List of example files",
      "items": {
        "type": "object",
        "required": ["path", "status", "mode"],
        "additionalProperties": false,
        "properties": {
          "path": {
            "type": "string",
            "pattern": "^[^/].*\\.ail$",
            "description": "Relative path to example file"
          },
          "status": {
            "type": "string",
            "enum": ["working", "broken", "experimental"],
            "description": "Current status of the example"
          },
          "mode": {
            "type": "string",
            "enum": ["file", "repl"],
            "description": "Execution mode for the example"
          },
          "tags": {
            "type": "array",
            "items": {"type": "string"},
            "description": "Categorization tags"
          },
          "description": {
            "type": "string",
            "description": "Human-readable description"
          },
          "expected": {
            "type": "object",
            "description": "Expected output for validation",
            "additionalProperties": false,
            "properties": {
              "stdout": {
                "type": "string",
                "description": "Expected standard output"
              },
              "stderr": {
                "type": "string",
                "description": "Expected standard error"
              },
              "exit_code": {
                "type": "integer",
                "minimum": 0,
                "maximum": 255,
                "description": "Expected exit code"
              },
              "error_pattern": {
                "type": "string",
                "description": "Regex pattern for expected error"
              }
            }
          },
          "environment": {
            "type": "object",
            "description": "Environment settings for deterministic execution",
            "additionalProperties": false,
            "properties": {
              "seed": {
                "type": "integer",
                "description": "Random seed"
              },
              "locale": {
                "type": "string",
                "description": "Locale setting"
              },
              "timezone": {
                "type": "string",
                "description": "Timezone setting"
              }
            }
          },
          "broken": {
            "type": "object",
            "description": "Information about why example is broken",
            "required": ["reason", "error_code"],
            "additionalProperties": false,
            "properties": {
              "reason": {
                "type": "string",
                "description": "Human-readable explanation"
              },
              "error_code": {
                "type": "string",
                "pattern": "^[A-Z]{2,3}\\d{3}$",
                "description": "Error code (e.g., PAR001)"
              },
              "requires": {
                "type": "array",
                "items": {"type": "string"},
                "description": "Required features to work"
              },
              "tracked_issue": {
                "type": "string",
                "description": "Issue tracker URL"
              }
            }
          },
          "requires_features": {
            "type": "array",
            "items": {"type": "string"},
            "description": "Feature flags required"
          },
          "skip_reason": {
            "type": "string",
            "description": "Why example is skipped in CI"
          }
        }
      }
    },
    "statistics": {
      "type": "object",
      "required": ["total", "working", "broken", "experimental", "coverage"],
      "additionalProperties": false,
      "properties": {
        "total": {
          "type": "integer",
          "minimum": 0,
          "description": "Total number of examples"
        },
        "working": {
          "type": "integer",
          "minimum": 0,
          "description": "Number of working examples"
        },
        "broken": {
          "type": "integer",
          "minimum": 0,
          "description": "Number of broken examples"
        },
        "experimental": {
          "type": "integer",
          "minimum": 0,
          "description": "Number of experimental examples"
        },
        "coverage": {
          "type": "number",
          "minimum": 0.0,
          "maximum": 1.0,
          "description": "Fraction of examples that work"
        }
      }
    }
  }
}`

// ExampleHeaderSchema defines the schema for example file headers
const ExampleHeaderSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "ailang.example-header/v1",
  "title": "AILANG Example Header",
  "description": "Machine-parseable header in example files",
  "type": "object",
  "required": ["status"],
  "additionalProperties": false,
  "properties": {
    "status": {
      "type": "string",
      "enum": ["working", "broken", "experimental"]
    },
    "error_code": {
      "type": "string",
      "pattern": "^[A-Z]{2,3}\\d{3}$"
    },
    "reason": {
      "type": "string"
    },
    "requires": {
      "type": "array",
      "items": {"type": "string"}
    },
    "manifest_path": {
      "type": "string"
    }
  }
}`
