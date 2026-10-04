// Chrome for the two screens reachable with no sign-in (Catalog, BookDetail).
// They are routed ABOVE the sign-in guard (thunder-authentication §6) and so
// have no session to build a Sidebar from — the wireframe's "My Cart / My
// Orders / Notifications" navbar links still appear (same literal copy), but
// as plain links: clicking one while signed out falls through to the
// catch-all route, which signs the visitor in, exactly as clicking a gated
// link should.
import type { ReactElement, ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { Box, Button, Header, Stack } from "@wso2/oxygen-ui";
import { BookOpen, Bell, ClipboardList, ShoppingCart } from "@wso2/oxygen-ui-icons-react";
import { useAuthz } from "../authz/gates";
import { signIn, signOut } from "../authz/session";
import { APP_NAME } from "../appName";

export function PublicHeader(): ReactElement {
  const navigate = useNavigate();
  const { signedIn, username } = useAuthz();

  return (
    <Header>
      <Header.Brand onClick={() => navigate("/")}>
        <Header.BrandLogo>
          <BookOpen />
        </Header.BrandLogo>
        <Header.BrandTitle>{APP_NAME}</Header.BrandTitle>
      </Header.Brand>
      <Header.Spacer />
      <Header.Actions>
        <Stack direction="row" spacing={1} alignItems="center">
          <Button
            variant="text"
            startIcon={<ShoppingCart size={18} />}
            onClick={() => navigate("/cart")}
          >
            My Cart
          </Button>
          <Button
            variant="text"
            startIcon={<ClipboardList size={18} />}
            onClick={() => navigate("/orders")}
          >
            My Orders
          </Button>
          <Button
            variant="text"
            startIcon={<Bell size={18} />}
            onClick={() => navigate("/notifications")}
          >
            Notifications
          </Button>
          {signedIn ? (
            <Button variant="outlined" onClick={() => void signOut()}>
              Sign out ({username})
            </Button>
          ) : (
            <Button variant="contained" onClick={() => void signIn()}>
              Sign in
            </Button>
          )}
        </Stack>
      </Header.Actions>
    </Header>
  );
}

export function PublicPage({ children }: { children: ReactNode }): ReactElement {
  return (
    <Box sx={{ minHeight: "100vh", bgcolor: "background.default" }}>
      <PublicHeader />
      <Box sx={{ p: 3 }}>{children}</Box>
    </Box>
  );
}
