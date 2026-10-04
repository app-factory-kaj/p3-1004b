// The one registered redirect URI, for both the redirect leg and the silent
// renew's hidden iframe — handleCallback() dispatches on stored request_type.
import { useEffect, type ReactElement } from "react";
import { useNavigate } from "react-router-dom";
import { handleCallback } from "../authz/session";

export function CallbackPage(): ReactElement {
  const navigate = useNavigate();

  useEffect(() => {
    void handleCallback().then(() => navigate("/", { replace: true }));
  }, [navigate]);

  return (
    <main>
      <p>Signing you in…</p>
    </main>
  );
}
