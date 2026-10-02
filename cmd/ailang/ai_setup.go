package main

import (
	"github.com/sunholo-data/ailang/internal/ai/factory"
	"github.com/sunholo-data/ailang/internal/config"
)

// aiSetup is everything the AI flags of `ailang run` / `serve-api` say about
// the handler, bundled so a new flag does not widen runFile's parameter list.
type aiSetup struct {
	// Stub selects the built-in deterministic stub handler (--ai-stub).
	Stub bool
	// Model is the --ai model name; "" means no real provider.
	Model string
	// NoADC disables Google's Application Default Credentials lane
	// (--ai-no-adc; AILANG_AI_NO_ADC=1 does the same process-wide).
	NoADC bool
	// KeyFile is the --ai-key-file path; "" falls back to AILANG_AI_KEY_FILE.
	KeyFile string
}

// aiAuth is the resolved #1499 credential policy handed to the factory.
type aiAuth struct {
	options     []factory.Option
	fromKeyFile bool
}

// resolveAuth reads the key file (flag, else AILANG_AI_KEY_FILE) once and
// turns the flags into factory options. The key lives only in the returned
// option closure: it is never logged, put in an error or exported to the
// environment, and the file is read before the program's first instruction.
func (s aiSetup) resolveAuth() (aiAuth, error) {
	var a aiAuth
	if s.NoADC {
		a.options = append(a.options, factory.WithNoADC(true))
	}
	path := s.KeyFile
	if path == "" {
		path = config.AIKeyFile()
	}
	if path != "" {
		key, err := factory.ReadKeyFile(path)
		if err != nil {
			return aiAuth{}, err
		}
		a.options = append(a.options, factory.WithAPIKey(key))
		a.fromKeyFile = true
	}
	return a, nil
}

// firstAuth returns the single optional aiAuth argument, or the zero value.
func firstAuth(auth []aiAuth) aiAuth {
	if len(auth) == 0 {
		return aiAuth{}
	}
	return auth[0]
}
