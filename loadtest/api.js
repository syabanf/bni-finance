// Uji beban k6 untuk backend BNI Finance.
//
//   BASE_URL=http://127.0.0.1:8123 ADMIN_EMAIL=… ADMIN_PASSWORD=… \
//     k6 run loadtest/api.js
//
// Dua skenario yang meniru dua jenis lalu lintas nyata, berjalan BERSAMAAN
// karena begitulah produksi: admin membaca dashboard sementara batch penagihan
// berjalan.
//
//   baca     admin menjelajah: daftar invoice, member, dashboard, laporan
//   tulis    penerbitan invoice — jalur yang dulu punya balapan penomoran
//
// Skenario "publik" dihapus bersama permukaan bayar publik: tidak ada lagi
// endpoint tanpa autentikasi untuk dibebani.
//
// Threshold di bawah MENGGAGALKAN run (exit code ≠ 0), jadi skrip ini bisa
// dipasang di CI sebagai gerbang, bukan sekadar penghasil angka.
//
// Angka threshold-nya sengaja longgar: p95 < 500 ms itu batas "jelas rusak",
// bukan target performa. Batas ketat milik pengukuran di perangkat produksi —
// pelajaran dari benchmark pool kemarin, saat ragam antar-run di mesin
// pengembang lebih besar daripada efek yang diukur.

import http from 'k6/http';
import { check, fail } from 'k6';
import { Counter } from 'k6/metrics';

const BASE = __ENV.BASE_URL || 'http://127.0.0.1:8123';
// Kredensial WAJIB lewat env — tidak ada default. Kata sandi yang ter-commit
// sebagai fallback akan berakhir dipakai di tempat yang tidak seharusnya.
const EMAIL = __ENV.ADMIN_EMAIL;
const PASSWORD = __ENV.ADMIN_PASSWORD;

// Nomor invoice ganda dari penerbitan paralel — bug yang SUDAH diperbaiki;
// counter ini menjaga agar tidak kambuh tanpa ketahuan.
const duplicateNumbers = new Counter('invoice_duplicate_numbers');

export const options = {
  scenarios: {
    baca: {
      executor: 'ramping-vus',
      exec: 'baca',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 30 },
        { duration: '30s', target: 30 },
        { duration: '5s', target: 0 },
      ],
    },
    tulis: {
      executor: 'constant-arrival-rate',
      exec: 'tulis',
      rate: 10, timeUnit: '1s', duration: '45s',
      preAllocatedVUs: 20, maxVUs: 60,
    },
  },
  thresholds: {
    // Gagalkan run bila salah satu dilanggar.
    http_req_failed: ['rate<0.01'],                          // <1% error
    'http_req_duration{scenario:baca}': ['p(95)<500'],
    'http_req_duration{scenario:tulis}': ['p(95)<800'],      // tulis = transaksi + lock
    invoice_duplicate_numbers: ['count==0'],
    checks: ['rate>0.99'],

    // --- lantai volume: "tidak ada data" harus GAGAL, bukan lulus -------------
    //
    // k6 menilai threshold atas metrik KOSONG sebagai lulus. Terbukti: sebuah
    // skrip yang tidak mengirim satu request pun tetap keluar dengan exit code
    // 0 dan menampilkan ✓ untuk semuanya — termasuk `checks rate>0.99` yang
    // dilaporkan rate=0.00%, dan `p(95)<500` pada {scenario:baca} di skrip yang
    // bahkan tidak punya skenario bernama baca.
    //
    // Konsekuensinya bukan teoretis. Dua threshold latensi di atas di-tag per NAMA
    // skenario. Salah ketik atau rename satu nama membuat metriknya tidak
    // pernah terisi, thresholdnya ✓ selamanya, dan latensi jalur itu berhenti
    // dijaga tanpa ada yang merah. Gerbang CI-nya hijau justru karena tidak
    // mengukur apa-apa.
    //
    // Ini kelas bug yang sama dengan benchmark yang dulu terbaca "9.606 req/s"
    // padahal seluruh 1.000 request-nya 401: angka tanpa bukti bahwa kerjanya
    // benar-benar terjadi tidak bernilai.
    //
    // Angkanya sengaja rendah — tugasnya menangkap NOL, bukan menetapkan target
    // throughput. baca 30 VU selama 45 detik menghasilkan ratusan ribu iterasi
    // di mesin pengembang; mesin CI paling lambat pun jauh melewati 100.
    // tulis memakai constant-arrival-rate 10/detik × 45 detik =
    // 450 iterasi yang deterministik, jadi lantainya bisa lebih dekat.
    'iterations{scenario:baca}': ['count>100'],
    'iterations{scenario:tulis}': ['count>300'],
    http_reqs: ['count>500'],
  },
};

// setup berjalan sekali: login, ambil id yang benar-benar ada. VU tidak boleh
// login sendiri-sendiri — 100 VU × login berarti 100 PBKDF2 600k iterasi, dan
// yang terukur jadi hashing, bukan API.
export function setup() {
  if (!EMAIL || !PASSWORD) {
    fail('set ADMIN_EMAIL dan ADMIN_PASSWORD — pakai kredensial SEED_ADMIN dari backend/.env');
  }
  const login = http.post(`${BASE}/api/v1/auth/login`,
    JSON.stringify({ email: EMAIL, password: PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } });
  if (login.status !== 200) {
    fail(`login gagal (${login.status}): ${login.body} — periksa ADMIN_EMAIL/ADMIN_PASSWORD`);
  }
  const token = login.json('token');
  const auth = { headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' } };

  const members = http.get(`${BASE}/api/v1/members?status=active&limit=1`, auth);
  const member = members.json('data.0');
  if (!member) fail('tidak ada member aktif — jalankan `make db-reset` dulu');

  return { token, memberId: member.id, chapterId: member.chapterId };
}

const authHeaders = (data) => ({
  headers: { Authorization: `Bearer ${data.token}`, 'Content-Type': 'application/json' },
});

// --- skenario ------------------------------------------------------------------

export function baca(data) {
  const h = authHeaders(data);
  const paths = [
    '/api/v1/invoices?limit=20',
    '/api/v1/invoices?status=outstanding',
    '/api/v1/members',
    '/api/v1/chapters',
    '/api/v1/payments',
    '/api/v1/dashboard/summary',
  ];
  const res = http.get(BASE + paths[Math.floor(Math.random() * paths.length)], h);
  check(res, { 'baca 200': (r) => r.status === 200 });
}

// Satu Set nomor per VU cukup: k6 tidak berbagi memori antar-VU, tetapi
// indeks unik di database-lah penjaga sebenarnya — 500 di sini berarti
// balapan penomoran kambuh, dan itu tertangkap threshold http_req_failed.
const seen = new Set();

export function tulis(data) {
  const h = authHeaders(data);
  const body = JSON.stringify({
    memberId: data.memberId,
    chapterId: data.chapterId,
    type: 'renewal',
    amount: 250000,
    dueDate: '2026-12-31',
    periodStart: '2026-08-03',
    periodEnd: '2027-08-03',
  });
  const res = http.post(`${BASE}/api/v1/invoices`, body, h);
  const ok = check(res, { 'tulis 201': (r) => r.status === 201 });
  if (ok) {
    const number = res.json('number');
    if (seen.has(number)) duplicateNumbers.add(1);
    seen.add(number);
  }
}
