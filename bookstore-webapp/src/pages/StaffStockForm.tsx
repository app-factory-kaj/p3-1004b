// screen StaffStockForm "Adjust a book's on-hand quantity"
// heading "Adjust Stock — <Book>"
// input "On-hand quantity"
// row: right, button "Cancel" -> StaffInventory, button "Save" primary -> StaffInventory
import { useEffect, useState, type ReactElement } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Alert, Button, PageContent, PageTitle, Stack, TextField } from "@wso2/oxygen-ui";
import { catalogApi, inventoryApi } from "../api";

export function StaffStockFormPage(): ReactElement {
  const { bookId } = useParams<{ bookId: string }>();
  const navigate = useNavigate();
  const [title, setTitle] = useState("");
  const [quantity, setQuantity] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!bookId) return;
    let live = true;
    void (async () => {
      const [{ data: book }, { data: stock }] = await Promise.all([
        catalogApi.GET("/books/{bookId}", { params: { path: { bookId } } }),
        inventoryApi.GET("/stock-levels/{bookId}", { params: { path: { bookId } } }),
      ]);
      if (!live) return;
      if (book) setTitle(book.title);
      if (stock) setQuantity(String(stock.quantityOnHand));
    })();
    return () => {
      live = false;
    };
  }, [bookId]);

  async function save(): Promise<void> {
    if (!bookId) return;
    setError(null);
    const quantityOnHand = Number(quantity);
    if (!Number.isInteger(quantityOnHand) || quantityOnHand < 0) {
      setError("Enter a non-negative whole number.");
      return;
    }
    setSaving(true);
    const { error: err } = await inventoryApi.PUT("/stock-levels/{bookId}", {
      params: { path: { bookId } },
      body: { quantityOnHand },
    });
    setSaving(false);
    if (err) {
      setError(err.message || "Could not update stock.");
      return;
    }
    navigate("/staff/inventory");
  }

  return (
    <PageContent maxWidth={480}>
      <PageTitle>
        <PageTitle.Header>Adjust Stock{title ? ` — ${title}` : ""}</PageTitle.Header>
      </PageTitle>

      <Stack spacing={2} sx={{ mt: 2 }}>
        {error ? <Alert severity="error">{error}</Alert> : null}
        <TextField label="On-hand quantity" value={quantity} onChange={(e) => setQuantity(e.target.value)} />
        <Stack direction="row" justifyContent="flex-end" spacing={2}>
          <Button variant="outlined" onClick={() => navigate("/staff/inventory")}>
            Cancel
          </Button>
          <Button variant="contained" disabled={saving} onClick={() => void save()}>
            Save
          </Button>
        </Stack>
      </Stack>
    </PageContent>
  );
}
