# Feature: Profil

**Route:** `/profile`  
**Status:** `done`  
**Domain:** Manajemen akun admin yang sedang login.

---

## User Stories

### US-01 Mengubah nama profil
> Sebagai admin, saya ingin mengubah nama tampilan akun saya, agar informasi profil tetap akurat.

**Acceptance Criteria:**
- [ ] Input nama yang bisa diedit (pre-filled dengan nama saat ini)
- [ ] Tombol Simpan aktif hanya jika nama berubah
- [ ] Loading state selama menyimpan
- [ ] Toast sukses/gagal setelah simpan
- [ ] Email tidak bisa diubah (read-only)

### US-02 Mengubah password (mode produksi)
> Sebagai admin, saya ingin mengubah password saya, agar keamanan akun terjaga.

**Acceptance Criteria:**
- [ ] Section password hanya muncul jika `VITE_USE_MOCK=false` (Supabase mode)
- [ ] Input: password baru + konfirmasi password baru
- [ ] Validasi: min 6 karakter, kedua input harus match
- [ ] Pesan error inline jika tidak valid
- [ ] Toast sukses setelah berhasil, input dikosongkan

### US-03 Logout
> Sebagai admin, saya ingin logout dari sistem, agar sesi saya berakhir dengan aman.

**Acceptance Criteria:**
- [ ] Tombol Logout tersedia di halaman profil
- [ ] Setelah logout: session dihapus, redirect ke `/login`

---

## State & Data

| State | Tipe | Keterangan |
|---|---|---|
| `user` | `AuthUser` | Dari `AuthContext` |
| `name` | `string` | Form state untuk edit nama |
| `pw, pw2` | `string` | Form state untuk ganti password |
| `savingName, savingPw` | `boolean` | Loading states |
