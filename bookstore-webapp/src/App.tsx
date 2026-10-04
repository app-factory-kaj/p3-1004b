// ROUTING STRUCTURE is prescribed by thunder-authentication (adapted from
// App.example.tsx):
//   - NoAccess sits ABOVE the shell route and REPLACES it.
//   - Forbidden sits INSIDE the shell, routed at /forbidden.
//   - /forbidden is wired into authz/client once, from the router root.
//   - Every gated route is wrapped in <RequireOperation>, the operation taken
//     from SCREEN_ROUTES — never typed here.
//   - A `public` screen (Catalog, BookDetail) is routed ABOVE the sign-in
//     guard, inside AuthzProvider (so <Can>/useScopes work), with its own
//     lightweight chrome (PublicPage) rather than the signed-in AppShell —
//     there is no session to build a Sidebar from.
//   - /callback is routed OUTSIDE the provider.
import { useEffect, type ReactElement } from "react";
import { BrowserRouter, Navigate, Route, Routes, useNavigate } from "react-router-dom";
import { AuthzProvider, Forbidden, NoAccess, RequireOperation, useAuthz, useScopes } from "./authz/gates";
import { SCREEN_ROUTES, reachableScreens, hasScopedReach } from "./authz/screens";
import { setForbiddenNavigator } from "./authz/client";
import { signIn } from "./authz/session";
import { AppShell } from "./shell/AppShell";
import { PublicPage } from "./shell/PublicHeader";
import { APP_NAME } from "./appName";

import { CallbackPage } from "./pages/Callback";
import { CatalogPage } from "./pages/Catalog";
import { BookDetailPage } from "./pages/BookDetail";
import { CartPage } from "./pages/Cart";
import { CheckoutPage } from "./pages/Checkout";
import { OrderConfirmationPage } from "./pages/OrderConfirmation";
import { OrdersPage } from "./pages/Orders";
import { OrderDetailPage } from "./pages/OrderDetail";
import { NotificationsPage } from "./pages/Notifications";
import { AccountPage } from "./pages/Account";
import { AddAddressPage } from "./pages/AddAddress";
import { StaffCatalogPage } from "./pages/StaffCatalog";
import { StaffBookFormPage } from "./pages/StaffBookForm";
import { StaffInventoryPage } from "./pages/StaffInventory";
import { StaffStockFormPage } from "./pages/StaffStockForm";

const PAGE_BY_KEY: Record<string, ReactElement> = {
  catalog: <CatalogPage />,
  bookdetail: <BookDetailPage />,
  cart: <CartPage />,
  checkout: <CheckoutPage />,
  orderconfirmation: <OrderConfirmationPage />,
  orders: <OrdersPage />,
  orderdetail: <OrderDetailPage />,
  notifications: <NotificationsPage />,
  account: <AccountPage />,
  addaddress: <AddAddressPage />,
  staffcatalog: <StaffCatalogPage />,
  staffbookform: <StaffBookFormPage />,
  staffinventory: <StaffInventoryPage />,
  staffstockform: <StaffStockFormPage />,
};

const PUBLIC_SCREENS = SCREEN_ROUTES.filter((screen) => screen.public);

export function App(): ReactElement {
  return (
    <BrowserRouter>
      <ForbiddenWiring />
      <Routes>
        <Route path="/callback" element={<CallbackPage />} />
        {PUBLIC_SCREENS.map((screen) => (
          <Route
            key={screen.key}
            path={screen.path}
            element={
              <AuthzProvider fallback={<Splash />}>
                <PublicPage>{PAGE_BY_KEY[screen.key]}</PublicPage>
              </AuthzProvider>
            }
          />
        ))}
        <Route
          path="*"
          element={
            <AuthzProvider fallback={<Splash />}>
              <SignedIn />
            </AuthzProvider>
          }
        />
      </Routes>
    </BrowserRouter>
  );
}

function ForbiddenWiring(): null {
  const navigate = useNavigate();
  useEffect(() => {
    setForbiddenNavigator(() => navigate("/forbidden", { replace: true }));
  }, [navigate]);
  return null;
}

function Splash(): ReactElement {
  return (
    <main>
      <h1>{APP_NAME}</h1>
      <p>Checking your session…</p>
    </main>
  );
}

function SignedIn(): ReactElement {
  const { signedIn } = useAuthz();
  const scopes = useScopes();

  useEffect(() => {
    if (!signedIn) void signIn();
  }, [signedIn]);

  if (!signedIn) return <Splash />;

  const reachable = reachableScreens(scopes, signedIn);

  if (!hasScopedReach(scopes, signedIn)) return <NoAccess appName={APP_NAME} />;

  const landing = (reachable.find((s) => !s.public && s.loads !== null) ?? reachable[0]).path;

  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<Navigate to={landing} replace />} />
        {SCREEN_ROUTES.map((screen) => {
          if (screen.public) return null;
          const page = PAGE_BY_KEY[screen.key];
          if (screen.loads === null) {
            return <Route key={screen.key} path={screen.path} element={page} />;
          }
          return (
            <Route key={screen.key} element={<RequireOperation op={screen.loads} screen={screen.label} />}>
              <Route path={screen.path} element={page} />
            </Route>
          );
        })}
        <Route path="/forbidden" element={<Forbidden />} />
        <Route path="*" element={<Navigate to={landing} replace />} />
      </Route>
    </Routes>
  );
}
