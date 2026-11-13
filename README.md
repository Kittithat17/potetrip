# Travel Split

เว็บแอป (PWA) หารค่าใช้จ่ายตอนเที่ยวกับเพื่อน อ่านสลิปด้วย AI และรองรับหลายสกุลเงิน
สเปกเต็มอยู่ที่ `TRAVEL_SPLIT_SPEC.md`

> **สถานะ:** Milestone 4 (หลายสกุลเงิน + รายเมนู + QR) และระบบเข้าสู่ระบบด้วย Google เสร็จแล้ว สร้างทริป ชวนเพื่อน บันทึกรายจ่ายสกุลไหนก็ได้ (แปลงเป็นสกุลหลักด้วยเรตของวันที่จ่าย หรือใส่เรตเอง) หารเท่ากัน ระบุยอด หรือรายเมนู สแกนสลิปให้ AI อ่าน (ดึงรายการเมนูจากใบเสร็จได้) เคลียร์เงินพร้อม QR พร้อมเพย์ และดูหน้าสรุปพร้อมกราฟ

## สิ่งที่ต้องมีในเครื่อง

| เครื่องมือ | เวอร์ชัน | หมายเหตุ |
|---|---|---|
| Go | 1.26+ | |
| Node.js | 20.19+ | แนะนำ 22 LTS (Node 20 หมดระยะซัพพอร์ตแล้ว และบาง lib เวอร์ชันใหม่ต้องใช้ 22) |
| pnpm | 10+ | |
| Docker Desktop | | ใช้รัน Postgres ตอน dev (พอร์ต **5440** เพื่อไม่ชนกับ Postgres ที่อาจติดตั้งไว้ในเครื่อง) |
| golangci-lint | 2.x | `brew install golangci-lint` |

`sqlc` กับ `goose` ไม่ต้องติดตั้งเอง เพราะล็อกเวอร์ชันไว้ใน `backend/tools/go.mod` แล้ว และสั่งผ่าน `make` (ข้างในเรียก `go tool -modfile=tools/go.mod ...`)

## เริ่มใช้งาน

```bash
make setup     # สร้าง .env จาก .env.example และติดตั้ง dependency
make dev       # เปิด postgres, รัน migration, แล้วรัน backend (:8081) + frontend (:5173)
```

เปิด http://localhost:5173 แล้วสร้างทริป จากนั้นกด "ชวนเพื่อน" เพื่อคัดลอกลิงก์เชิญ (`/t/<รหัส 10 ตัว>`) ถ้าจะลองเป็นหลายคนในเครื่องเดียว ให้เปิดลิงก์ในหน้าต่าง Incognito แล้วเลือกชื่อหรือเพิ่มชื่อใหม่

> แก้โค้ด backend แล้วต้องหยุด `make dev` แล้วรันใหม่ (frontend reload ให้เอง)

### ลองบนมือถือ (Wi-Fi เดียวกัน)

```bash
make dev-lan
```

terminal จะพิมพ์ลิงก์เช่น `http://192.168.1.112:5173` ให้เปิดบนมือถือที่ต่อ Wi-Fi เดียวกับคอม ถ้าเปิดไม่ได้ ให้เช็กว่า macOS Firewall อนุญาต `node` (System Settings → Network → Firewall) ข้อจำกัดของ http: ปุ่มแชร์จะแสดงลิงก์ให้คัดลอกเองแทน และยังติดตั้งเป็นแอปไม่ได้จนกว่าจะ deploy บน https อย่าใช้ `dev-lan` บน Wi-Fi สาธารณะ

ตรวจ backend ตรงๆ:

```bash
curl localhost:8081/healthz          # {"status":"ok"}
curl localhost:8081/api/v1/healthz   # เหมือนกัน
```

## คำสั่ง make

| คำสั่ง | หน้าที่ |
|---|---|
| `make dev` | รันทุกอย่างพร้อมกัน (Ctrl+C หยุดทั้งหมด) |
| `make dev-lan` | เหมือน `dev` แต่ให้มือถือใน Wi-Fi เดียวกันเปิดได้ |
| `make dev-backend` / `make dev-frontend` | รันทีละฝั่ง |
| `make db-up` / `make db-down` | เปิด/ปิด Postgres |
| `make db-reset` | ลบข้อมูล DB dev ทั้งหมดแล้วสร้างใหม่ |
| `make migrate` / `make migrate-down` / `make migrate-status` | จัดการ migration |
| `make migrate-new name=add_x` | สร้างไฟล์ migration ใหม่ |
| `make sqlc` | generate โค้ด Go จาก `backend/queries/*.sql` |
| `make test` | `go test ./...` (รวม integration test กับ Postgres) + `vitest` |
| `make test-unit` | test backend เฉพาะส่วนที่ไม่ใช้ DB |
| `make lint` | `go vet` + `golangci-lint` + `tsc` + `oxlint` |
| `make build` | build binary และ `frontend/dist` |

## โครงสร้าง

```
backend/
  cmd/api/            จุดเริ่มโปรแกรม
  internal/config/    โหลด env
  internal/trip/      ทริปและสมาชิก
  internal/expense/   รายจ่าย + การแบ่งเงิน (Allocate/SplitEqual)
  internal/settlement/ คำนวณยอดสุทธิและรายการโอน (มี unit test ตามสเปกหัวข้อ 7)
  internal/money/     ตารางสกุลเงิน จำนวนทศนิยม และ ParseDecimal
  internal/fx/        อัตราแลกเปลี่ยน: Provider (Frankfurter) + cache ใน DB
  internal/slip/      Reader interface, GeminiReader, MockReader, RetryReader, ประมวลผลขนาน
  internal/apperr/    error ของ domain ที่ map เป็น HTTP status
  internal/testdb/    สร้าง schema ชั่วคราวสำหรับ integration test
  internal/http/      routes, middleware, handlers
  internal/db/        โค้ดที่ sqlc generate (ห้ามแก้มือ)
  migrations/         goose migrations (schema ทั้งหมดตามสเปกหัวข้อ 4)
  queries/            SQL สำหรับ sqlc
  tools/go.mod        ล็อกเวอร์ชัน sqlc และ goose
frontend/
  src/api/            API client (แปลง error format เป็น ApiError)
  src/components/     คอมโพเนนต์
  src/pages/          หน้าจอ
docker-compose.yml    Postgres 17 สำหรับ dev
```

## AI อ่านสลิป (M3)

1. ขอ API key ฟรีที่ [Google AI Studio](https://aistudio.google.com/apikey)
2. ใส่ใน `.env`:
   ```
   SLIP_PROVIDER=gemini
   GEMINI_API_KEY=<key ของคุณ>
   GEMINI_MODEL=<ชื่อรุ่น>
   ```
   ดูชื่อรุ่นและโควตาใน AI Studio (เมนู Rate Limit) ทดสอบเมื่อ 12 พ.ย. 2569: `gemini-3.5-flash-lite` อ่านสลิปไทยและใบเสร็จได้ถูกต้องใน ~2 วินาที (15 RPM ใน free tier) ส่วน `gemini-3.8-flash` (5 RPM) เจอ 503 "high demand" บ่อย เปลี่ยนรุ่นได้โดยแก้แค่ `GEMINI_MODEL` ไม่ต้องแก้โค้ด
3. รัน `make dev` ใหม่ log จะขึ้น `slip reading enabled`

ถ้ายังไม่มี key ตั้ง `SLIP_PROVIDER=mock` เพื่อลองหน้าจอด้วยผลตัวอย่าง (ร้าน "ตัวอย่าง (mock)" ฿1,250) ห้ามใช้ใน production ถ้าไม่ได้ตั้งค่าอะไรเลย แอปยังใช้งานได้ปกติ แค่ปุ่มสแกนจะแจ้งว่ายังไม่ได้ตั้งค่า AI

**การทำงาน:** frontend ย่อรูปให้ด้านยาวไม่เกิน 1600px (และตัดข้อมูล EXIF เช่น GPS ออก) แล้วอัปโหลดทีละรูป ครั้งละไม่เกิน 3 รูปพร้อมกัน เพื่อแสดงสถานะของแต่ละรูป backend ตรวจชนิดไฟล์จากเนื้อไฟล์จริง (JPEG/PNG/WebP/HEIC ไม่เกิน 8MB) แล้วส่งให้ AI อ่านแบบขนาน จำกัดด้วย `SLIP_CONCURRENCY` ถ้าเจอ 429 (เกินโควตา) หรือ 503 (โมเดลไม่ว่าง) จะ retry ด้วยระยะรอ 2s → 4s → 8s รูปไม่ถูกเก็บลงดิสก์หรือ DB เก็บแค่ผลที่อ่านได้ใน `slips.raw_result` ผู้ใช้ต้องกดบันทึกหรือข้ามทีละรายการ ระบบไม่บันทึกเอง และสลิปหนึ่งใบผูกกับรายจ่ายได้รายการเดียว

## อัตราแลกเปลี่ยน (แท็บ "แลกเงิน")

ใช้เรตอ้างอิงของธนาคารกลางยุโรป (ECB) ผ่าน [Frankfurter](https://frankfurter.dev) ซึ่งฟรีและไม่ต้องใช้ key เรตอัปเดตวันทำการละครั้ง ไม่ใช่ realtime ร้านแลกเงินหรือบัตรเครดิตจะต่างไปเล็กน้อย รองรับ 30 สกุล (ดูได้ที่ `GET /fx/currencies`) **ไม่มี** TWD, VND, LAK, KHR, MMK

backend เก็บเรตไว้ในตาราง `fx_rates`: เรตของวันที่ผ่านมาเกิน 7 วันเก็บถาวร ส่วนเรตล่าสุด refresh ทุกชั่วโมง ถ้า Frankfurter ล่มจะใช้เรตใน cache แทนและตอบ `stale: true`

## รายจ่ายหลายสกุลเงินและการหารรายเมนู (M4)

- **หลายสกุลเงิน:** รายจ่ายเลือกสกุลเองได้ backend แปลงเป็นสกุลหลักด้วยเรตอ้างอิงของ **วันที่จ่าย** (ถ้าเป็นวันในอนาคตใช้เรตล่าสุด) เก็บเรตไว้ใน `expenses.fx_rate` (ทศนิยม 8 ตำแหน่ง) แล้วปัดครึ่งขึ้นเป็นหน่วยย่อย ผู้ใช้ส่ง `fx_rate` ของตัวเองได้ (เช่นเรตร้านแลกเงิน) ซึ่งจำเป็นสำหรับสกุลที่ไม่มีเรตอ้างอิง เช่น VND หรือ TWD ตอนแก้ไขรายการ ฟอร์มจะใช้เรตเดิมที่บันทึกไว้ เว้นแต่กดกลับไปใช้เรตอ้างอิง
- **รายเมนู (`split_type: items`):** แต่ละรายการมีชื่อ ราคา (สกุลของรายจ่าย) และคนที่กิน ยอดรวมของบิลไม่จำเป็นต้องเท่ากับผลรวมรายการ ส่วนต่าง (VAT ค่าบริการ ส่วนลด) จะเฉลี่ยตามราคาของแต่ละรายการ แล้วแต่ละรายการหารเท่ากันในคนที่กิน เศษให้ตามลำดับที่เข้าทริป ผลรวมจึงเท่ากับ `amount_base` เสมอ
- **สแกนใบเสร็จ:** ถ้า AI อ่านรายการเมนูได้ ฟอร์มจะเปิดเป็นแบบรายเมนูพร้อมรายการให้เลย และใช้สกุลเงินตามใบเสร็จ
- **QR พร้อมเพย์:** ใส่พร้อมเพย์ของตัวเองได้ในตั้งค่าทริป หน้าเคลียร์เงินจะมีปุ่ม QR (ใช้ `promptpay-qr` + `qrcode` สร้างในเครื่อง) ถ้าทริปไม่ได้ใช้บาท ยอดใน QR เป็นค่าประมาณจากเรตอ้างอิงล่าสุด
- **หน้าสรุป:** ยอดรวม ตามหมวดหมู่ รายคน (จ่ายแทนกลุ่มเทียบกับส่วนของตัวเอง) และสกุลเงินที่ใช้

## การเข้าสู่ระบบและสิทธิ์เข้าทริป

**ทุกคนต้องเข้าสู่ระบบด้วย Google ก่อน** ถ้ายังไม่เข้าสู่ระบบ จะเปิดหน้าไหนหรือเรียก API ไหนไม่ได้เลย ยกเว้น `/healthz` และ `/api/v1/auth/*` หลังเข้าสู่ระบบ แอปจะพากลับไปหน้าที่เปิดไว้ เช่นลิงก์เชิญ

- **Session อยู่ในคุกกี้ HttpOnly** JavaScript อ่านไม่ได้ จึงขโมยผ่าน XSS ไม่ได้ ตั้งค่า `SameSite=Lax` ถ้า `PUBLIC_URL` เป็น https จะใช้ชื่อ `__Host-ts_session` และเป็น `Secure` ฝั่ง DB เก็บแค่ SHA-256 ของ token อายุ 60 วัน และต่ออายุอัตโนมัติเมื่อมีการใช้งาน ส่วนการออกจากระบบจะลบ session ที่เซิร์ฟเวอร์ด้วย
- **ไม่มีข้อมูลลับใน localStorage** localStorage เหลือแค่ภาษาและคู่สกุลเงินที่เลือกไว้
- **กัน CSRF:** คำขอที่แก้ข้อมูลต้องมี header `X-Requested-With: travel-split` และถ้า `Sec-Fetch-Site` เป็น `cross-site` จะถูกปฏิเสธ (403 `csrf`)
- **Google OAuth:** ใช้ authorization code + PKCE + `state` ในคุกกี้ HttpOnly อายุ 10 นาที `client_secret` อยู่แค่ฝั่งเซิร์ฟเวอร์ และตรวจ `iss`, `aud`, `exp` กับ `email_verified` ของ ID token ส่วนหน้าที่จะกลับไปหลังล็อกอินต้องเป็น path ในเว็บเราเท่านั้น (กัน open redirect)
- **ห้ามใช้ `AUTH_PROVIDER=mock` บนเว็บจริง** โหมดนี้ใช้ตอน dev เท่านั้น พิมพ์ชื่ออะไรก็เข้าได้ และเซิร์ฟเวอร์จะไม่ยอมเริ่มถ้า `PUBLIC_URL` เป็น https
- **เครื่องที่เคยใช้แอปก่อนมีระบบล็อกอิน:** ล็อกอินครั้งแรกแล้ว แอปจะส่ง token เดิมใน localStorage ไปผูกทริปเข้ากับบัญชี (`POST /auth/claim`) จากนั้นเซิร์ฟเวอร์ทำให้ token นั้นใช้ไม่ได้อีก และแอปลบออกจากเครื่อง

สิทธิ์ในทริป:
- **คนสร้างทริป** เป็นเจ้าของ ได้รับอนุมัติอัตโนมัติ และออกจากทริปไม่ได้
- **คนที่ได้ลิงก์เชิญ** ทำได้แค่ส่งคำขอ (ใช้ชื่อใหม่ หรืออ้างว่าเป็นสมาชิกเดิม) แล้วรอ ระหว่างรอเรียกได้แค่ `/me` ถ้าถูกปฏิเสธจะขอใหม่ได้
- **เฉพาะเจ้าของทริป** เห็นและอนุมัติหรือปฏิเสธคำขอได้
- รายชื่อที่คนถือรหัสเชิญเห็นมีแค่ชื่อ ไม่มีเบอร์พร้อมเพย์
- ทริปผูกกับบัญชี ล็อกอินบัญชีเดิมบนเครื่องไหนก็ได้ทริปคืนครบ

### ตั้งค่า Google (ทำครั้งเดียว)

1. เข้า [Google Cloud Console](https://console.cloud.google.com) แล้วสร้าง Project
2. ไปที่ **Google Auth Platform** (หรือ **OAuth consent screen**) ใส่ชื่อแอปและอีเมล แล้วเลือกประเภทผู้ใช้เป็น **External**
3. ไปที่ **Clients** แล้วสร้าง **OAuth client ID** ประเภท **Web application**
4. ช่อง **Authorized redirect URIs** ใส่ `{PUBLIC_URL}/api/v1/auth/google/callback` เช่น `http://localhost:5173/api/v1/auth/google/callback` ตอน deploy ให้เพิ่มของโดเมนจริงที่เป็น https
5. ใส่ค่าในไฟล์ `.env` ได้แก่ `AUTH_PROVIDER=google`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` และ `PUBLIC_URL`
6. ตอนที่แอปยังอยู่ในโหมด Testing ต้องเพิ่มอีเมลของคนที่จะทดสอบไว้ใน **Test users** ก่อน ถ้าจะเปิดให้ทุกคนใช้ ต้องกด **Publish app**

Google ไม่รับ redirect ไปที่ IP ในวง LAN เช่น `192.168.x.x` ถ้าจะลองบนมือถือก่อน deploy ให้ใช้ `AUTH_PROVIDER=mock` หรือใช้ tunnel ที่เป็น https

## API

Base path `/api/v1` ทุก endpoint (ยกเว้น `healthz` และ `auth/*`) ต้องเข้าสู่ระบบก่อน

รหัสสถานะ:
- ยังไม่เข้าสู่ระบบ → 401 `login_required`
- ไม่ได้อยู่ในทริปนี้ → 403 `not_a_member`
- รออนุมัติ → 403 `pending_approval`
- ถูกปฏิเสธ → 403 `join_rejected`
- คำขอแก้ข้อมูลที่ไม่มี header กัน CSRF → 403 `csrf`

| Method | Path | สิทธิ์ | หมายเหตุ |
|---|---|---|---|
| GET | `/auth/config` | ทุกคน | `{provider: "google" \| "mock"}` |
| GET | `/auth/google/start?next=/path` | ทุกคน | redirect ไป Google |
| GET | `/auth/google/callback` | ทุกคน | Google ส่งกลับมาที่นี่ แล้วตั้งคุกกี้และ redirect ไป `next` |
| POST | `/auth/mock` | ทุกคน (เฉพาะโหมด mock) | `{name}` |
| POST | `/auth/logout` | ทุกคน | ลบ session |
| GET | `/auth/me` | เข้าสู่ระบบ | `{id, email, name, picture}` |
| POST | `/auth/claim` | เข้าสู่ระบบ | `{tokens: [...]}` ย้ายทริปจาก token แบบเก่ามาไว้ในบัญชี |
| GET | `/me/trips` | เข้าสู่ระบบ | ทริปที่อยู่หรือรออนุมัติ `[{id, name, invite_code, status, member_name, is_owner}]` |
| POST | `/trips` | เข้าสู่ระบบ | `{name, creator_name?, base_currency?, promptpay?}` คืน `{trip, member}` (ถ้าไม่ส่ง `creator_name` จะใช้ชื่อบัญชี) |
| GET | `/trips/{inviteCode}` | เข้าสู่ระบบ | `{trip, members: [{id, name}]}` |
| POST | `/trips/{tripId}/join` | เข้าสู่ระบบ | `{name?, promptpay?}` หรือ `{member_id}` คืน `{request}` (รอได้สูงสุด 20 คำขอต่อทริป) |
| GET | `/trips/{tripId}/me` | อยู่ในทริปหรือรออนุมัติ | `{status, member, is_owner, request}` |
| DELETE | `/trips/{tripId}/me` | อยู่ในทริปหรือรออนุมัติ | ออกจากทริปหรือยกเลิกคำขอ (เจ้าของทริปทำไม่ได้: 409 `owner_cannot_leave`) |
| GET | `/trips/{tripId}/members` | สมาชิก | |
| GET | `/trips/{tripId}/expenses?category=&member=` | สมาชิก | `member` = คนจ่ายหรือคนที่หารด้วย |
| POST/PUT | `/trips/{tripId}/expenses[/{id}]` | สมาชิก | ดูตัวอย่างด้านล่าง |
| GET/DELETE | `/trips/{tripId}/expenses/{id}` | สมาชิก | |
| GET | `/trips/{tripId}/balances` | สมาชิก | `net` บวก = ได้คืน, ลบ = ต้องจ่าย |
| GET | `/trips/{tripId}/settlement` | สมาชิก | รายการโอนที่น้อยที่สุด |
| POST | `/trips/{tripId}/settlements/paid` | สมาชิก | `{from, to, amount_base}` |
| POST | `/trips/{tripId}/slips` | สมาชิก | multipart ไฟล์รูปได้สูงสุด 10 รูป |
| GET | `/trips/{tripId}/summary` | สมาชิก | `{total, count, by_category, by_currency, by_member: [{member_id, paid, consumed}]}` |
| PATCH | `/trips/{tripId}/me` | สมาชิก | `{promptpay}` แก้พร้อมเพย์ของตัวเอง (`null` หรือ `""` = ลบ) |
| PATCH | `/trips/{tripId}` | เจ้าของ | `{name?, base_currency?}` สกุลเงินหลักล็อกเมื่อมีรายจ่ายหรือการโอนคืนแล้ว (409 `currency_locked`) |
| GET | `/trips/{tripId}/join-requests` | เจ้าของ | คำขอที่รออยู่ |
| POST | `/trips/{tripId}/join-requests/{id}/approve` \| `reject` | เจ้าของ | |
| GET | `/fx?from=CNY&to=THB[&date=YYYY-MM-DD]` | เข้าสู่ระบบ | อัตราแลกเปลี่ยน |
| GET | `/fx/currencies` | เข้าสู่ระบบ | สกุลที่รองรับ |

ตัวอย่างรายจ่าย (`amount` เป็นหน่วยย่อย 125000 = ฿1,250.00):

```json
{ "title": "ราเมง", "category": "food", "paid_by": "<id>", "amount": 125000,
  "split_type": "equal", "member_ids": ["<id>", "<id>", "<id>"] }

{ "title": "แท็กซี่", "category": "transport", "paid_by": "<id>", "amount": 45000,
  "split_type": "exact", "shares": [{"member_id": "<id>", "amount": 15000}, {"member_id": "<id>", "amount": 30000}] }

{ "title": "Ichiran", "category": "food", "paid_by": "<id>", "amount": 4500, "currency": "JPY",
  "spent_at": "2026-11-10T19:30:00+07:00", "split_type": "equal", "member_ids": ["<id>", "<id>"] }

{ "title": "Pho", "category": "food", "paid_by": "<id>", "amount": 150000, "currency": "VND", "fx_rate": "0.00136",
  "split_type": "items", "items": [{"name": "Pho bo", "amount": 100000, "member_ids": ["<id>"]},
                                   {"name": "Coffee", "amount": 50000, "member_ids": ["<id>", "<id>"]}] }
```

ใน `exact` และ `items` ยอดของแต่ละคน/แต่ละรายการเป็นสกุลของรายจ่าย (`currency`) ส่วน `shares` ที่ API คืนมาเป็นสกุลหลักของทริปเสมอ

เศษจากการหารเท่ากันจะเพิ่มทีละ 1 หน่วยย่อยให้สมาชิกตามลำดับที่เข้าทริป ผลรวม `shares` จึงเท่ากับ `amount_base` เสมอ

## Environment variables

ดู `.env.example` ทั้ง backend และ frontend อ่านจากไฟล์ `.env` ที่ root (ไฟล์นี้อยู่ใน `.gitignore` แล้ว ห้าม commit)

## ภาษา

UI มี 2 ภาษา: **English (ค่าเริ่มต้น)** และ **ไทย** เลือกได้ในหน้าตั้งค่า แอปจำภาษาที่เลือกไว้ใน `localStorage` ข้อความทั้งหมดอยู่ที่ `frontend/src/i18n/en.ts` และ `th.ts` ซึ่ง TypeScript และ test บังคับให้ทั้งสองไฟล์มีคำครบเท่ากัน error จาก API จะแปลตาม `error.code` (ข้อความจาก server ใช้เมื่อไม่มีคำแปล) ส่วนวันที่, ตัวเลข และชื่อสกุลเงินใช้ `Intl` ตามภาษาที่เลือก

## ข้อตกลงในโค้ด

- เงินเก็บเป็นจำนวนเต็มในหน่วยย่อยตามมาตรฐาน ISO 4217 (THB 2 ตำแหน่ง, JPY 0 ตำแหน่ง) ห้ามใช้ float
- Error ของ API ใช้รูปแบบ `{ "error": { "code": "...", "message": "..." } }` เสมอ เพิ่ม code ใหม่แล้วให้เพิ่มคำแปลใน `errors` ของทั้งสองภาษาด้วย
