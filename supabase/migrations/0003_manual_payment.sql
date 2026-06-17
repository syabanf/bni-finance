-- ===========================================================================
-- Manual payment — record offline payments with an uploaded proof.
-- Apply via the Supabase SQL editor.
-- ===========================================================================

-- payments: proof + note
alter table payments add column if not exists proof_url text;
alter table payments add column if not exists note      text;

-- Storage bucket for uploaded payment proofs (public read for easy viewing).
insert into storage.buckets (id, name, public)
values ('payment-proofs', 'payment-proofs', true)
on conflict (id) do nothing;

-- Storage policies: authenticated users upload, anyone can read (bucket public).
do $$
begin
  if not exists (
    select 1 from pg_policies
    where schemaname = 'storage' and tablename = 'objects' and policyname = 'payment_proofs_insert'
  ) then
    create policy "payment_proofs_insert" on storage.objects
      for insert to authenticated with check (bucket_id = 'payment-proofs');
  end if;

  if not exists (
    select 1 from pg_policies
    where schemaname = 'storage' and tablename = 'objects' and policyname = 'payment_proofs_read'
  ) then
    create policy "payment_proofs_read" on storage.objects
      for select using (bucket_id = 'payment-proofs');
  end if;
end $$;

-- Reload PostgREST schema cache
select pg_notify('pgrst', 'reload schema');
