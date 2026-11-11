# Travel Split

เว็บแอป (PWA) หารค่าใช้จ่ายตอนเที่ยวกับเพื่อน อ่านสลิปด้วย AI และรองรับหลายสกุลเงิน
สเปกเต็มอยู่ที่ `TRAVEL_SPLIT_SPEC.md`

> **สถานะ:** Milestone 2 (ทริปและการหารเงิน) เสร็จแล้ว สร้างทริป ชวนเพื่อน (หน้าแรกจำทริปที่เคยเข้าไว้ในเครื่อง) บันทึกรายจ่ายแบบหารเท่ากันหรือระบุยอด และดูว่าใครต้องโอนให้ใครได้ ยังรองรับเฉพาะสกุลเงินหลักของทริป (หลายสกุลเงินมาใน M4)

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

ตรวจ backend ตรงๆ:

```bash
curl localhost:8081/healthz          # {"status":"ok"}
curl localhost:8081/api/v1/healthz   # เหมือนกัน
```

## คำสั่ง make

| คำสั่ง | หน้าที่ |
|---|---|
| `make dev` | รันทุกอย่างพร้อมกัน (Ctrl+C หยุดทั้งหมด) |
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
  internal/money/     ตารางสกุลเงินและจำนวนทศนิยม
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

## API (M2)

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
