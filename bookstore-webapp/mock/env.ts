// window._env_ for mock mode — exactly the keys the platform actually emits
// for this component: the user-auth OIDC keys (no JWKS_URL — the browser
// never validates a token). No sibling service URL: all six are same-origin
// /api / /api/<name>/.
export const mockEnv = {
  USER_AUTH_CLIENT_ID: "mock-client",
  USER_AUTH_ISSUER: "https://mock-idp.test",
  USER_AUTH_SCOPES:
    "openid profile email group ou account:read account:manage cart:read cart:manage " +
    "orders:read orders:create notifications:read notifications:create stock:reserve " +
    "books:manage stock:manage stock:view-reservations",
  USER_AUTH_RESOURCE: "https://mock-idp.test/resources/mock-project",
};
