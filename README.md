# Travel Split

เว็บแอป (PWA) หารค่าใช้จ่ายตอนเที่ยวกับเพื่อน อ่านสลิปด้วย AI และรองรับหลายสกุลเงิน
สเปกเต็มอยู่ที่ `TRAVEL_SPLIT_SPEC.md`

> **สถานะ:** Milestone 3 (AI อ่านสลิป) เสร็จแล้ว สร้างทริป ชวนเพื่อน (หน้าแรกจำทริปที่เคยเข้าไว้ในเครื่อง) บันทึกรายจ่ายแบบหารเท่ากันหรือระบุยอด สแกนสลิปหลายรูปให้ AI อ่านแล้วตรวจก่อนบันทึก และดูว่าใครต้องโอนให้ใครได้ ยังรองรับเฉพาะสกุลเงินหลักของทริป (หลายสกุลเงินมาใน M4)

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

## API (M2–M3)

Base path `/api/v1` ทุก endpoint ใต้ `/trips/{tripId}` ยกเว้น `members` ต้องส่ง header `X-Member-Id` ของสมาชิกในทริปนั้น (ไม่ส่ง → 401, ไม่ใช่สมาชิก → 403)

| Method | Path | หมายเหตุ |
|---|---|---|
| POST | `/trips` | `{name, base_currency?}` คืนทริปพร้อม `invite_code` |
| GET | `/trips/{inviteCode}` | ไม่สนตัวพิมพ์เล็กใหญ่ |
| GET/POST | `/trips/{tripId}/members` | `{name, promptpay?}` ชื่อซ้ำในทริป → 409 |
| GET | `/trips/{tripId}/expenses?category=&member=` | `member` = คนจ่ายหรือคนที่หารด้วย |
| POST/PUT | `/trips/{tripId}/expenses[/{id}]` | ดูตัวอย่างด้านล่าง |
| GET/DELETE | `/trips/{tripId}/expenses/{id}` | |
| GET | `/trips/{tripId}/balances` | `net` บวก = ได้คืน, ลบ = ต้องจ่าย |
| GET | `/trips/{tripId}/settlement` | รายการโอนที่น้อยที่สุด |
| POST | `/trips/{tripId}/settlements/paid` | `{from, to, amount_base}` |
| POST | `/trips/{tripId}/slips` | multipart ไฟล์รูปได้สูงสุด 10 รูป คืน `results[]` ที่มี `slip_id`, `result` (ผลดิบ), `suggestion` (ค่าแนะนำสำหรับฟอร์ม) ส่ง `slip_id` ตอนสร้างรายจ่ายเพื่อผูกสลิป |

ตัวอย่างรายจ่าย (`amount` เป็นหน่วยย่อย 125000 = ฿1,250.00):

```json
{ "title": "ราเมง", "category": "food", "paid_by": "<id>", "amount": 125000,
  "split_type": "equal", "member_ids": ["<id>", "<id>", "<id>"] }

{ "title": "แท็กซี่", "category": "transport", "paid_by": "<id>", "amount": 45000,
  "split_type": "exact", "shares": [{"member_id": "<id>", "amount": 15000}, {"member_id": "<id>", "amount": 30000}] }
```

เศษจากการหารเท่ากันจะเพิ่มทีละ 1 หน่วยย่อยให้สมาชิกตามลำดับที่เข้าทริป ผลรวม `shares` จึงเท่ากับ `amount_base` เสมอ

## Environment variables

ดู `.env.example` ทั้ง backend และ frontend อ่านจากไฟล์ `.env` ที่ root (ไฟล์นี้อยู่ใน `.gitignore` แล้ว ห้าม commit)

## ข้อตกลงในโค้ด

- เงินเก็บเป็นจำนวนเต็มในหน่วยย่อยตามมาตรฐาน ISO 4217 (THB 2 ตำแหน่ง, JPY 0 ตำแหน่ง) ห้ามใช้ float
- Error ของ API ใช้รูปแบบ `{ "error": { "code": "...", "message": "..." } }` เสมอ
