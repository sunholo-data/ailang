// OIDC verification for the resident agent (v6.40.0).
//
// WHY THIS EXISTS: Cloud Run *instances* (Preview) do not appear to enforce
// run.invoker at the edge — a same-project caller holding roles/run.invoker,
// presenting a valid ID token with the correct audience and email claim, is
// still refused 403. Google's own launch example sidesteps this by exposing
// the instance publicly with a shared password in an env var.
//
// A shared password would be security by obscurity with extra steps. Instead we
// perform exactly the check the edge would have: the caller still mints a
// Google-signed ID token for our URL, and we verify it ourselves. Public
// ingress, private authorisation.
//
// Deliberately the same shape as the platform's internal_tasks/auth.py:
// issuer set, ENFORCED audience, exact caller allowlist, uniform failure, and
// fail-closed when unconfigured. Without the audience check any Google-signed
// token for any service would pass.
import { createPublicKey, createVerify } from "node:crypto";

const GOOGLE_ISSUERS = new Set(["https://accounts.google.com", "accounts.google.com"]);
const JWKS_URL = "https://www.googleapis.com/oauth2/v3/certs";
const JWKS_TTL_MS = 60 * 60 * 1000;

let jwksCache = { keys: null, fetchedAt: 0 };

async function jwks() {
  if (jwksCache.keys && Date.now() - jwksCache.fetchedAt < JWKS_TTL_MS) return jwksCache.keys;
  const res = await fetch(JWKS_URL);
  if (!res.ok) throw new Error(`JWKS fetch failed: ${res.status}`);
  const body = await res.json();
  jwksCache = { keys: body.keys, fetchedAt: Date.now() };
  return body.keys;
}

const b64 = (s) => Buffer.from(s.replace(/-/g, "+").replace(/_/g, "/"), "base64");

export function authConfig() {
  return {
    audience: (process.env.RESIDENT_AUDIENCE || "").trim(),
    allowed: (process.env.RESIDENT_ALLOWED_CALLERS || "")
      .split(",").map((s) => s.trim()).filter(Boolean),
  };
}

/** Resolve the caller's identity, or throw. Never returns a partial result. */
export async function verify(authorizationHeader) {
  const { audience, allowed } = authConfig();
  // Unconfigured is CLOSED, never open: a misdeployed instance must refuse
  // everything rather than serve the world.
  if (!audience || allowed.length === 0) {
    throw new Error("auth unconfigured (RESIDENT_AUDIENCE / RESIDENT_ALLOWED_CALLERS)");
  }
  if (!authorizationHeader?.startsWith("Bearer ")) throw new Error("missing bearer token");
  const token = authorizationHeader.slice(7).trim();
  const [h, p, s] = token.split(".");
  if (!h || !p || !s) throw new Error("malformed token");

  const header = JSON.parse(b64(h).toString());
  const claims = JSON.parse(b64(p).toString());

  // Signature first: nothing in the payload may be trusted before it verifies.
  const key = (await jwks()).find((k) => k.kid === header.kid);
  if (!key) throw new Error("unknown signing key");
  const ok = createVerify("RSA-SHA256")
    .update(`${h}.${p}`)
    .verify(createPublicKey({ key, format: "jwk" }), b64(s));
  if (!ok) throw new Error("bad signature");

  const now = Math.floor(Date.now() / 1000);
  if (typeof claims.exp !== "number" || claims.exp < now) throw new Error("token expired");
  if (claims.nbf && claims.nbf > now + 60) throw new Error("token not yet valid");
  if (!GOOGLE_ISSUERS.has(claims.iss)) throw new Error(`bad issuer ${claims.iss}`);
  // Enforced, not merely inspected: an unaudienced check would accept a token
  // minted for any other service.
  if (claims.aud !== audience) throw new Error("audience mismatch");
  if (claims.email_verified !== true) throw new Error("email not verified");
  if (!allowed.includes(claims.email)) throw new Error(`caller not allowed: ${claims.email}`);
  return { email: claims.email, sub: claims.sub };
}
