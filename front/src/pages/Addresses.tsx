import { useState, type FormEvent } from "react";
import { MapPin, Pencil, Plus, Star, Trash2 } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Badge } from "@lib/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { Skeleton } from "@lib/components/ui/skeleton";
import {
  useAddresses,
  useCreateAddress,
  useDeleteAddress,
  useSetDefaultAddress,
  useUpdateAddress,
  type AddressInput,
} from "@lib/hooks/useAddresses";
import type { Address } from "@lib/types";

const empty: AddressInput = {
  recipient: "",
  phone: "",
  province: "",
  city: "",
  district: "",
  line1: "",
  postalCode: "",
  default: false,
};

export function AddressesPage() {
  const { data, isLoading } = useAddresses();
  const create = useCreateAddress();
  const update = useUpdateAddress();
  const remove = useDeleteAddress();
  const setDefault = useSetDefaultAddress();

  const [editing, setEditing] = useState<Address | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<AddressInput>(empty);

  const openCreate = () => {
    setEditing(null);
    setForm(empty);
    setShowForm(true);
  };
  const openEdit = (a: Address) => {
    setEditing(a);
    setForm({ ...a });
    setShowForm(true);
  };
  const close = () => {
    setShowForm(false);
    setEditing(null);
  };

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (editing) {
      update.mutate({ id: editing.id, input: form }, { onSuccess: close });
    } else {
      create.mutate(form, { onSuccess: close });
    }
  }

  const busy = create.isPending || update.isPending;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Addresses</h1>
          <p className="text-muted-foreground">Manage your shipping addresses.</p>
        </div>
        {!showForm && (
          <Button onClick={openCreate}>
            <Plus className="size-4" />
            New address
          </Button>
        )}
      </div>

      {showForm && (
        <Card>
          <CardHeader>
            <CardTitle>{editing ? "Edit address" : "New address"}</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={onSubmit} className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="recipient">Recipient</Label>
                <Input
                  id="recipient"
                  value={form.recipient}
                  onChange={(e) => setForm({ ...form, recipient: e.target.value })}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="phone">Phone</Label>
                <Input
                  id="phone"
                  value={form.phone}
                  onChange={(e) => setForm({ ...form, phone: e.target.value })}
                />
              </div>
              <div className="space-y-2 sm:col-span-2">
                <Label htmlFor="line1">Address line</Label>
                <Input
                  id="line1"
                  value={form.line1}
                  onChange={(e) => setForm({ ...form, line1: e.target.value })}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="province">Province / State</Label>
                <Input
                  id="province"
                  value={form.province}
                  onChange={(e) => setForm({ ...form, province: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="city">City</Label>
                <Input
                  id="city"
                  value={form.city}
                  onChange={(e) => setForm({ ...form, city: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="district">District</Label>
                <Input
                  id="district"
                  value={form.district}
                  onChange={(e) => setForm({ ...form, district: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="postalCode">Postal code</Label>
                <Input
                  id="postalCode"
                  value={form.postalCode}
                  onChange={(e) => setForm({ ...form, postalCode: e.target.value })}
                />
              </div>
              <label className="flex items-center gap-2 text-sm sm:col-span-2">
                <input
                  type="checkbox"
                  checked={form.default ?? false}
                  onChange={(e) => setForm({ ...form, default: e.target.checked })}
                />
                Set as default address
              </label>
              <div className="flex gap-2 sm:col-span-2">
                <Button type="submit" disabled={busy}>
                  {busy ? "Saving…" : editing ? "Save changes" : "Add address"}
                </Button>
                <Button type="button" variant="outline" onClick={close}>
                  Cancel
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      {isLoading ? (
        <Skeleton className="h-24 w-full" />
      ) : !data || data.length === 0 ? (
        <div className="text-muted-foreground rounded-lg border border-dashed py-16 text-center">
          <MapPin className="mx-auto size-8" />
          <p className="mt-2">No addresses yet.</p>
        </div>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2">
          {data.map((a) => (
            <Card key={a.id}>
              <CardContent className="space-y-2">
                <div className="flex items-center gap-2">
                  <span className="font-medium">{a.recipient}</span>
                  {a.default && <Badge variant="success">Default</Badge>}
                </div>
                <p className="text-muted-foreground text-sm">{a.phone}</p>
                <p className="text-sm">
                  {[a.province, a.city, a.district, a.line1, a.postalCode]
                    .filter(Boolean)
                    .join(" ")}
                </p>
                <div className="flex gap-1 pt-2">
                  <Button variant="ghost" size="sm" onClick={() => openEdit(a)}>
                    <Pencil className="size-4" />
                    Edit
                  </Button>
                  {!a.default && (
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={setDefault.isPending}
                      onClick={() => setDefault.mutate(a.id)}
                    >
                      <Star className="size-4" />
                      Set default
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={remove.isPending}
                    onClick={() => remove.mutate(a.id)}
                  >
                    <Trash2 className="size-4" />
                    Delete
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
