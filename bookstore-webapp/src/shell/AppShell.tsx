// The app chrome — ONE shell, ONE rail, shared by every signed-in screen
// (Shopper and Store Staff alike). Each nav item is wrapped in <Can>, which
// reproduces the wireframe's per-role navbar/sidebar pictures and also covers
// a caller holding both roles. Mirrors the pinned design system's sample
// AppLayout (AppShell > Navbar/Sidebar/Main/Footer).
import type { ReactElement } from "react";
import { Outlet, useLocation, useNavigate, Link as NavigateLink } from "react-router-dom";
import {
  AppShell as OxygenAppShell,
  ColorSchemeToggle,
  Footer,
  Header,
  Sidebar,
  UserMenu,
} from "@wso2/oxygen-ui";
import {
  BookOpen,
  Bell,
  ClipboardList,
  LogOut,
  ShoppingCart,
  UserCircle,
  Warehouse,
} from "@wso2/oxygen-ui-icons-react";
import { Can, useAuthz, useHeldRoles } from "../authz/gates";
import { signOut } from "../authz/session";
import { APP_NAME } from "../appName";

export function AppShell(): ReactElement {
  const navigate = useNavigate();
  const location = useLocation();
  const { username } = useAuthz();
  const roles = useHeldRoles();

  const activeItem = ((): string => {
    const p = location.pathname;
    if (p.startsWith("/cart")) return "cart";
    if (p.startsWith("/checkout")) return "cart";
    if (p.startsWith("/orders")) return "orders";
    if (p.startsWith("/notifications")) return "notifications";
    if (p.startsWith("/account")) return "account";
    if (p.startsWith("/staff/catalog")) return "staff-catalog";
    if (p.startsWith("/staff/inventory")) return "staff-inventory";
    return "catalog";
  })();

  return (
    <OxygenAppShell>
      <OxygenAppShell.Navbar>
        <Header>
          <Header.Toggle />
          <Header.Brand onClick={() => navigate("/")}>
            <Header.BrandLogo>
              <BookOpen />
            </Header.BrandLogo>
            <Header.BrandTitle>{APP_NAME}</Header.BrandTitle>
          </Header.Brand>
          <Header.Spacer />
          <Header.Actions>
            <ColorSchemeToggle />
            <UserMenu>
              <UserMenu.Trigger name={username || "Signed in"} />
              <UserMenu.Header
                name={username || "Signed in"}
                email=""
                role={roles[0] ?? undefined}
              />
              <UserMenu.Item icon={<UserCircle />} label="My Account" onClick={() => navigate("/account")} />
              <UserMenu.Divider />
              <UserMenu.Logout icon={<LogOut />} onClick={() => void signOut()} />
            </UserMenu>
          </Header.Actions>
        </Header>
      </OxygenAppShell.Navbar>

      <OxygenAppShell.Sidebar>
        <Sidebar activeItem={activeItem}>
          <Sidebar.Nav>
            <Sidebar.Category>
              <Sidebar.Item id="catalog" link={<NavigateLink to="/" />}>
                <Sidebar.ItemIcon>
                  <BookOpen />
                </Sidebar.ItemIcon>
                <Sidebar.ItemLabel>Catalog</Sidebar.ItemLabel>
              </Sidebar.Item>
              <Can op="GET /me/cart">
                <Sidebar.Item id="cart" link={<NavigateLink to="/cart" />}>
                  <Sidebar.ItemIcon>
                    <ShoppingCart />
                  </Sidebar.ItemIcon>
                  <Sidebar.ItemLabel>My Cart</Sidebar.ItemLabel>
                </Sidebar.Item>
              </Can>
              <Can op="GET /me/orders">
                <Sidebar.Item id="orders" link={<NavigateLink to="/orders" />}>
                  <Sidebar.ItemIcon>
                    <ClipboardList />
                  </Sidebar.ItemIcon>
                  <Sidebar.ItemLabel>My Orders</Sidebar.ItemLabel>
                </Sidebar.Item>
              </Can>
              <Can op="GET /me/notifications">
                <Sidebar.Item id="notifications" link={<NavigateLink to="/notifications" />}>
                  <Sidebar.ItemIcon>
                    <Bell />
                  </Sidebar.ItemIcon>
                  <Sidebar.ItemLabel>Notifications</Sidebar.ItemLabel>
                </Sidebar.Item>
              </Can>
              <Can op="GET /me/profile">
                <Sidebar.Item id="account" link={<NavigateLink to="/account" />}>
                  <Sidebar.ItemIcon>
                    <UserCircle />
                  </Sidebar.ItemIcon>
                  <Sidebar.ItemLabel>My Account</Sidebar.ItemLabel>
                </Sidebar.Item>
              </Can>
            </Sidebar.Category>

            <Can op="PUT /books/{bookId}">
              <Sidebar.Category>
                <Sidebar.CategoryLabel>Store Staff</Sidebar.CategoryLabel>
                <Sidebar.Item id="staff-catalog" link={<NavigateLink to="/staff/catalog" />}>
                  <Sidebar.ItemIcon>
                    <BookOpen />
                  </Sidebar.ItemIcon>
                  <Sidebar.ItemLabel>Catalog</Sidebar.ItemLabel>
                </Sidebar.Item>
                <Can op="GET /reservations">
                  <Sidebar.Item id="staff-inventory" link={<NavigateLink to="/staff/inventory" />}>
                    <Sidebar.ItemIcon>
                      <Warehouse />
                    </Sidebar.ItemIcon>
                    <Sidebar.ItemLabel>Inventory</Sidebar.ItemLabel>
                  </Sidebar.Item>
                </Can>
              </Sidebar.Category>
            </Can>
          </Sidebar.Nav>
        </Sidebar>
      </OxygenAppShell.Sidebar>

      <OxygenAppShell.Main>
        <Outlet />
      </OxygenAppShell.Main>

      <OxygenAppShell.Footer>
        <Footer>
          <Footer.Copyright>© {new Date().getFullYear()} WSO2 LLC.</Footer.Copyright>
        </Footer>
      </OxygenAppShell.Footer>
    </OxygenAppShell>
  );
}
