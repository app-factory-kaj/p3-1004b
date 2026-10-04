// screen Account "The signed-in Shopper's profile and addresses"
// heading "My Account"
// card "Profile": input "Full name", input "Email", button "Save profile"
// heading "Shipping Addresses"
// table "Address | City | Default"
// button "Add address" primary -> AddAddress
import { useEffect, useState, type ReactElement } from "react";
import { useNavigate } from "react-router-dom";
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  Chip,
  ListingTable,
  PageContent,
  PageTitle,
  Stack,
  TextField,
  Typography,
} from "@wso2/oxygen-ui";
import { customersApi } from "../api";
import { Can } from "../authz/gates";
import type { components as CustomersComponents } from "../generated/customers-service";

type Customer = CustomersComponents["schemas"]["Customer"];
type Address = CustomersComponents["schemas"]["Address"];

export function AccountPage(): ReactElement {
  const navigate = useNavigate();
  const [profile, setProfile] = useState<Customer | null>(null);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [addresses, setAddresses] = useState<Address[]>([]);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    let live = true;
    void (async () => {
      const [{ data: profileData }, { data: addressData }] = await Promise.all([
        customersApi.GET("/me/profile", {}),
        customersApi.GET("/me/addresses", { params: { query: {} } }),
      ]);
      if (!live) return;
      if (profileData) {
        setProfile(profileData);
        setName(profileData.name);
        setEmail(profileData.email);
      }
      setAddresses(addressData?.data ?? []);
    })();
    return () => {
      live = false;
    };
  }, []);

  async function saveProfile(): Promise<void> {
    setSaving(true);
    setSaved(false);
    const { data } = await customersApi.PUT("/me/profile", { body: { name, email } });
    setSaving(false);
    if (data) {
      setProfile(data);
      setSaved(true);
    }
  }

  return (
    <PageContent maxWidth={720}>
      <PageTitle>
        <PageTitle.Header>My Account</PageTitle.Header>
      </PageTitle>

      <Card sx={{ mt: 2 }}>
        <CardHeader title="Profile" />
        <CardContent>
          <Stack spacing={2}>
            <TextField label="Full name" value={name} onChange={(e) => setName(e.target.value)} />
            <TextField label="Email" value={email} onChange={(e) => setEmail(e.target.value)} />
            {saved ? <Typography color="success.main">Profile saved.</Typography> : null}
            <Stack direction="row">
              <Can op="PUT /me/profile">
                <Button variant="contained" disabled={saving} onClick={() => void saveProfile()}>
                  Save profile
                </Button>
              </Can>
            </Stack>
          </Stack>
        </CardContent>
      </Card>

      <Typography variant="h6" sx={{ mt: 4, mb: 2 }}>
        Shipping Addresses
      </Typography>
      <ListingTable.Container>
        <ListingTable>
          <ListingTable.Head>
            <ListingTable.Row>
              <ListingTable.Cell>Address</ListingTable.Cell>
              <ListingTable.Cell>City</ListingTable.Cell>
              <ListingTable.Cell>Default</ListingTable.Cell>
            </ListingTable.Row>
          </ListingTable.Head>
          <ListingTable.Body>
            {addresses.map((a) => (
              <ListingTable.Row key={a.id}>
                <ListingTable.Cell>{a.line1}</ListingTable.Cell>
                <ListingTable.Cell>{a.city}</ListingTable.Cell>
                <ListingTable.Cell>{a.isDefault ? <Chip label="Yes" color="success" size="small" /> : "No"}</ListingTable.Cell>
              </ListingTable.Row>
            ))}
          </ListingTable.Body>
        </ListingTable>
        {addresses.length === 0 ? (
          <ListingTable.EmptyState title="No addresses yet" description="Add a shipping address below." />
        ) : null}
      </ListingTable.Container>

      <Stack direction="row" justifyContent="flex-end" sx={{ mt: 2 }}>
        <Can op="POST /me/addresses">
          <Button variant="contained" onClick={() => navigate("/account/addresses/new")}>
            Add address
          </Button>
        </Can>
      </Stack>
    </PageContent>
  );
}
