-- =============================================================================
-- BNI Finance — RLS hardening
-- Replaces the permissive "allow all (using true)" policies with
-- AUTHENTICATED-ONLY access. After this, the public anon key alone can no
-- longer read/write any data — a logged-in Supabase session is required.
--
-- Run once in the Supabase SQL Editor. Safe to re-run (idempotent).
-- The `auto-create-invoices` Edge Function uses the service-role key, which
-- bypasses RLS, so it keeps working.
-- =============================================================================

do $$
declare
  t    text;
  pol  record;
  tables text[] := array[
    'chapters', 'members', 'fee_settings', 'invoices',
    'payments', 'invoice_audit_log', 'app_settings'
  ];
begin
  foreach t in array tables loop
    if to_regclass('public.' || t) is not null then
      -- ensure RLS is on
      execute format('alter table public.%I enable row level security', t);

      -- drop EVERY existing policy on the table (handles unknown/legacy names)
      for pol in
        select policyname from pg_policies
        where schemaname = 'public' and tablename = t
      loop
        execute format('drop policy %I on public.%I', pol.policyname, t);
      end loop;

      -- recreate as authenticated-only (anon role gets no policy => denied)
      execute format(
        'create policy %I on public.%I for all to authenticated using (true) with check (true)',
        'auth_all_' || t, t
      );
    end if;
  end loop;
end $$;

-- NOTE: `app_settings` holds the BNI VM token. Authenticated-only is a big
-- improvement over world-readable, but the token is still sent to logged-in
-- browsers. For real hardening, move the BNI VM sync into the Edge Function
-- (token in function env/secrets) and stop exposing it via the client API.
