package config

import "strings"

// The package registry: the client side (`ailang pkg …`) and the registry
// validator service (cmd/registry-validator).
const (
	EnvRegistry              = "AILANG_REGISTRY"
	EnvRegistryAPIKey        = "AILANG_REGISTRY_API_KEY"
	EnvRegistryValidator     = "AILANG_REGISTRY_VALIDATOR"
	EnvRegistryAPI           = "AILANG_REGISTRY_API"
	EnvRegistryObject        = "AILANG_REGISTRY_OBJECT"
	EnvRegistryBucket        = "REGISTRY_BUCKET"
	EnvRegistryServiceAPIKey = "REGISTRY_API_KEY"
	EnvFirestoreDatabase     = "FIRESTORE_DATABASE"
	EnvFirestoreKeysColl     = "FIRESTORE_KEYS_COLLECTION"
)

// DefaultRegistryURL is the public package registry bucket.
const DefaultRegistryURL = "https://storage.googleapis.com/ailang-registry"

// DefaultRegistryValidatorURL is the public registry validator service.
const DefaultRegistryValidatorURL = "https://registry.ailang.sunholo.com"

// DefaultRegistryObject is the object `models publish` writes.
const DefaultRegistryObject = "registry.yml"

var registryVars = []Var{
	{EnvRegistry, DefaultRegistryURL, AreaRegistry, "Base URL of the package registry index the pkg commands read."},
	{EnvRegistryAPIKey, "", AreaRegistry, "API key sent as X-API-Key to the validator for publish, unpublish and key management."},
	{EnvRegistryValidator, DefaultRegistryValidatorURL, AreaRegistry, "Registry validator base URL (trailing slash trimmed)."},
	{EnvRegistryAPI, "", AreaRegistry, "Deprecated alias of AILANG_REGISTRY_VALIDATOR, read after it."},
	{EnvRegistryObject, DefaultRegistryObject, AreaRegistry, "Object name `models publish` writes the model registry to."},
	{EnvRegistryBucket, "", AreaRegistry, "GCS bucket the registry validator serves; required, the service refuses to start without it."},
	{EnvRegistryServiceAPIKey, "", AreaRegistry, "Superuser API key the registry validator accepts."},
	{EnvFirestoreDatabase, "", AreaRegistry, "Firestore database holding scoped registry keys; unset runs the validator superuser-only."},
	{EnvFirestoreKeysColl, "ailang_registry_keys", AreaRegistry, "Firestore collection of scoped registry keys."},
}

// RegistryURL returns AILANG_REGISTRY, default DefaultRegistryURL.
func RegistryURL() string { return getOr(EnvRegistry) }

// RegistryAPIKey returns AILANG_REGISTRY_API_KEY, "" when unset.
func RegistryAPIKey() string { return get(EnvRegistryAPIKey) }

// RegistryValidatorURL returns the validator base URL without a trailing
// slash: AILANG_REGISTRY_VALIDATOR, then the deprecated AILANG_REGISTRY_API,
// then the default.
func RegistryValidatorURL() string {
	if u := get(EnvRegistryValidator); u != "" {
		return strings.TrimRight(u, "/")
	}
	if u := get(EnvRegistryAPI); u != "" {
		return strings.TrimRight(u, "/")
	}
	return defaultOf(EnvRegistryValidator)
}

// RegistryObject returns AILANG_REGISTRY_OBJECT, default registry.yml.
func RegistryObject() string { return getOr(EnvRegistryObject) }

// RegistryBucket returns REGISTRY_BUCKET, "" when unset.
func RegistryBucket() string { return get(EnvRegistryBucket) }

// RegistryServiceAPIKey returns REGISTRY_API_KEY, "" when unset.
func RegistryServiceAPIKey() string { return get(EnvRegistryServiceAPIKey) }

// FirestoreDatabase returns FIRESTORE_DATABASE, "" when unset.
func FirestoreDatabase() string { return get(EnvFirestoreDatabase) }

// FirestoreKeysCollection returns FIRESTORE_KEYS_COLLECTION, default
// ailang_registry_keys.
func FirestoreKeysCollection() string { return getOr(EnvFirestoreKeysColl) }
