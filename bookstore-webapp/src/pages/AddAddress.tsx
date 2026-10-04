// screen AddAddress "Add a new shipping address"
// heading, inputs (line1, line2, city, region, postal code, country),
// checkbox "Set as default", row: right, button "Save address" primary
import { useState, type ReactElement } from "react";
import { useNavigate } from "react-router-dom";
import {
  Alert,
  Button,
  Checkbox,
  FormControlLabel,
  PageContent,
  PageTitle,
  Stack,
  TextField,
} from "@wso2/oxygen-ui";
import { customersApi } from "../api";

export function AddAddressPage(): ReactElement {
  const navigate = useNavigate();
  const [line1, setLine1] = useState("");
  const [line2, setLine2] = useState("");
  const [city, setCity] = useState("");
  const [region, setRegion] = useState("");
  const [postalCode, setPostalCode] = useState("");
  const [country, setCountry] = useState("");
  const [isDefault, setIsDefault] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function save(): Promise<void> {
    setError(null);
    if (!line1 || !city || !region || !postalCode || !country) {
      setError("Fill in every required field.");
      return;
    }
    setSaving(true);
    const { error: err } = await customersApi.POST("/me/addresses", {
      body: { line1, line2: line2 || undefined, city, region, postalCode, country, isDefault },
    });
    setSaving(false);
    if (err) {
      setError(err.message || "Could not save that address.");
      return;
    }
    navigate("/account");
  }

  return (
    <PageContent maxWidth={560}>
      <PageTitle>
        <PageTitle.Header>Add Shipping Address</PageTitle.Header>
      </PageTitle>

      <Stack spacing={2} sx={{ mt: 2 }}>
        {error ? <Alert severity="error">{error}</Alert> : null}
        <TextField label="Address line 1" value={line1} onChange={(e) => setLine1(e.target.value)} />
        <TextField label="Address line 2" value={line2} onChange={(e) => setLine2(e.target.value)} />
        <TextField label="City" value={city} onChange={(e) => setCity(e.target.value)} />
        <TextField label="Region" value={region} onChange={(e) => setRegion(e.target.value)} />
        <TextField label="Postal code" value={postalCode} onChange={(e) => setPostalCode(e.target.value)} />
        <TextField label="Country" value={country} onChange={(e) => setCountry(e.target.value)} />
        <FormControlLabel
          control={<Checkbox checked={isDefault} onChange={(e) => setIsDefault(e.target.checked)} />}
          label="Set as default"
        />
        <Stack direction="row" justifyContent="flex-end">
          <Button variant="contained" disabled={saving} onClick={() => void save()}>
            Save address
          </Button>
        </Stack>
      </Stack>
    </PageContent>
  );
}
