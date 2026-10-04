// screen StaffBookForm "Create or edit a book"
// heading "Edit Book"
// input Title, Author, ISBN, textarea Description, input Price
// row: right, button "Cancel" -> StaffCatalog, button "Save book" primary -> StaffCatalog
//
// One screen serves both "Add book" (bookId param is the literal "new") and
// per-row "edit" (an existing id), per the wireframe's single StaffBookForm.
import { useEffect, useState, type ReactElement } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Alert, Button, PageContent, PageTitle, Stack, TextField } from "@wso2/oxygen-ui";
import { catalogApi } from "../api";

export function StaffBookFormPage(): ReactElement {
  const { bookId } = useParams<{ bookId: string }>();
  const navigate = useNavigate();
  const isNew = !bookId || bookId === "new";

  const [title, setTitle] = useState("");
  const [author, setAuthor] = useState("");
  const [isbn, setIsbn] = useState("");
  const [description, setDescription] = useState("");
  const [price, setPrice] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isNew || !bookId) return;
    let live = true;
    void (async () => {
      const { data } = await catalogApi.GET("/books/{bookId}", { params: { path: { bookId } } });
      if (!live || !data) return;
      setTitle(data.title);
      setAuthor(data.author);
      setIsbn(data.isbn ?? "");
      setDescription(data.description ?? "");
      setPrice(String(data.price));
    })();
    return () => {
      live = false;
    };
  }, [isNew, bookId]);

  async function save(): Promise<void> {
    setError(null);
    const priceNumber = Number(price);
    if (!title || !author || !Number.isFinite(priceNumber)) {
      setError("Title, author and a numeric price are required.");
      return;
    }
    setSaving(true);
    const body = { title, author, isbn: isbn || undefined, description: description || undefined, price: priceNumber };
    const { error: err } = isNew
      ? await catalogApi.POST("/books", { body })
      : await catalogApi.PUT("/books/{bookId}", { params: { path: { bookId: bookId! } }, body });
    setSaving(false);
    if (err) {
      setError(err.message || "Could not save this book.");
      return;
    }
    navigate("/staff/catalog");
  }

  return (
    <PageContent maxWidth={560}>
      <PageTitle>
        <PageTitle.Header>Edit Book</PageTitle.Header>
      </PageTitle>

      <Stack spacing={2} sx={{ mt: 2 }}>
        {error ? <Alert severity="error">{error}</Alert> : null}
        <TextField label="Title" value={title} onChange={(e) => setTitle(e.target.value)} />
        <TextField label="Author" value={author} onChange={(e) => setAuthor(e.target.value)} />
        <TextField label="ISBN" value={isbn} onChange={(e) => setIsbn(e.target.value)} />
        <TextField
          label="Description"
          multiline
          minRows={3}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        <TextField label="Price" value={price} onChange={(e) => setPrice(e.target.value)} />

        <Stack direction="row" justifyContent="flex-end" spacing={2}>
          <Button variant="outlined" onClick={() => navigate("/staff/catalog")}>
            Cancel
          </Button>
          <Button variant="contained" disabled={saving} onClick={() => void save()}>
            Save book
          </Button>
        </Stack>
      </Stack>
    </PageContent>
  );
}
