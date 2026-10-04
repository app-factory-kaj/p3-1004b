// screen Checkout "Review shipping and place the order"
// heading "Checkout"
// select "Shipping address"
// input "Payment method reference"
// row: right, button "Place order" primary -> OrderConfirmation
//
// Story 9: when POST /me/orders answers 400 (insufficient stock), that refusal
// is shown ON THIS SCREEN, plainly, rather than a generic failure — the Alert
// below, built from the Error schema's `message`/`description`.
import { useEffect, useState, type ReactElement } from "react";
import { useNavigate } from "react-router-dom";
import { Alert, Button, MenuItem, PageContent, PageTitle, Stack, TextField } from "@wso2/oxygen-ui";
import { customersApi, ordersApi } from "../api";
import type { components as CustomersComponents } from "../generated/customers-service";

type Address = CustomersComponents["schemas"]["Address"];

export function CheckoutPage(): ReactElement {
  const navigate = useNavigate();
  const [addresses, setAddresses] = useState<Address[]>([]);
  const [addressId, setAddressId] = useState("");
  const [paymentMethodRef, setPaymentMethodRef] = useState("");
  const [placing, setPlacing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let live = true;
    void (async () => {
      const { data } = await customersApi.GET("/me/addresses", { params: { query: {} } });
      if (!live) return;
      const rows = data?.data ?? [];
      setAddresses(rows);
      const def = rows.find((a) => a.isDefault) ?? rows[0];
      if (def) setAddressId(def.id);
    })();
    return () => {
      live = false;
    };
  }, []);

  async function placeOrder(): Promise<void> {
    setError(null);
    if (!addressId || !paymentMethodRef) {
      setError("Choose a shipping address and enter a payment method reference.");
      return;
    }
    setPlacing(true);
    const { data, error: err } = await ordersApi.POST("/me/orders", {
      body: { addressId, paymentMethodRef },
    });
    setPlacing(false);
    if (err) {
      // The out-of-stock refusal (400) and any other checkout failure both
      // surface here, on the Checkout screen, with the service's own message.
      setError(err.message || err.description || "The order could not be placed.");
      return;
    }
    if (data) navigate(`/orders/${data.id}/confirmation`);
  }

  return (
    <PageContent maxWidth={560}>
      <PageTitle>
        <PageTitle.Header>Checkout</PageTitle.Header>
      </PageTitle>

      <Stack spacing={3} sx={{ mt: 2 }}>
        {error ? <Alert severity="error">{error}</Alert> : null}

        <TextField
          select
          label="Shipping address"
          value={addressId}
          onChange={(e) => setAddressId(e.target.value)}
        >
          {addresses.length === 0 ? (
            <MenuItem value="" disabled>
              No saved addresses — add one in My Account
            </MenuItem>
          ) : null}
          {addresses.map((a) => (
            <MenuItem key={a.id} value={a.id}>
              {a.line1}, {a.city}
              {a.isDefault ? " (default)" : ""}
            </MenuItem>
          ))}
        </TextField>

        <TextField
          label="Payment method reference"
          value={paymentMethodRef}
          onChange={(e) => setPaymentMethodRef(e.target.value)}
        />

        <Stack direction="row" justifyContent="flex-end">
          <Button variant="contained" disabled={placing} onClick={() => void placeOrder()}>
            Place order
          </Button>
        </Stack>
      </Stack>
    </PageContent>
  );
}
