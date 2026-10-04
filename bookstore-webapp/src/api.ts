// Same-origin typed clients, one per sibling service, all through the ONE
// authorization rule in src/authz/client.ts: the bearer is attached and a
// 401/403 is classified into Forbidden vs. a fresh sign-in. Nothing here
// re-implements that decision.
//
// catalog-service is the PRIMARY dependency — nginx proxies it at bare /api
// (it is the one public, unauthenticated entry point: browsing). The other
// five are "extra siblings", proxied at /api/<component-name>/ per
// react-webapp's nginx drop-in.
import createClient, { type Middleware } from "openapi-fetch";
import type { paths as CatalogPaths } from "./generated/catalog-service";
import type { paths as CustomersPaths } from "./generated/customers-service";
import type { paths as CartPaths } from "./generated/cart-service";
import type { paths as OrdersPaths } from "./generated/orders-service";
import type { paths as InventoryPaths } from "./generated/inventory-service";
import type { paths as NotificationsPaths } from "./generated/notifications-service";
import { authorizationHeader, classifyResponse, ForbiddenError } from "./authz/client";

const authMiddleware: Middleware = {
  async onRequest({ request }) {
    const header = await authorizationHeader();
    if (header) request.headers.set("Authorization", header);
    return request;
  },
  async onResponse({ response }) {
    if ((await classifyResponse(response.status)) === "forbidden") {
      throw new ForbiddenError(response.status);
    }
    return response;
  },
};

export const catalogApi = createClient<CatalogPaths>({ baseUrl: "/api" });
catalogApi.use(authMiddleware);

export const customersApi = createClient<CustomersPaths>({ baseUrl: "/api/customers-service/" });
customersApi.use(authMiddleware);

export const cartApi = createClient<CartPaths>({ baseUrl: "/api/cart-service/" });
cartApi.use(authMiddleware);

export const ordersApi = createClient<OrdersPaths>({ baseUrl: "/api/orders-service/" });
ordersApi.use(authMiddleware);

export const inventoryApi = createClient<InventoryPaths>({ baseUrl: "/api/inventory-service/" });
inventoryApi.use(authMiddleware);

export const notificationsApi = createClient<NotificationsPaths>({
  baseUrl: "/api/notifications-service/",
});
notificationsApi.use(authMiddleware);
