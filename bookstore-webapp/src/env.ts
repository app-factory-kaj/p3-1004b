// Typed read of window._env_, mounted by the platform at /env-config.js at
// request time. Never read import.meta.env.VITE_* / process.env.* here — the
// platform does not use build-time config.
type Env = {
  // The `user-auth` platform-resource dependency (thunder-app). Not JWKS_URL:
  // the browser never validates a token — the API gateway does — so no asset
  // reads it.
  USER_AUTH_CLIENT_ID: string;
  USER_AUTH_ISSUER: string;
  USER_AUTH_SCOPES: string;
  USER_AUTH_RESOURCE: string;
};

declare global {
  interface Window {
    _env_: Env;
  }
}

if (!window._env_) {
  throw new Error(
    "window._env_ not set — /env-config.js failed to load. " +
      "The platform mounts this file; if you see this locally, host " +
      "/env-config.js from your dev server.",
  );
}

export const env: Env = window._env_;
