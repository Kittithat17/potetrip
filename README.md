# Travel Split

เว็บแอป (PWA) หารค่าใช้จ่ายตอนเที่ยวกับเพื่อน อ่านสลิปด้วย AI และรองรับหลายสกุลเงิน
สเปกเต็มอยู่ที่ `TRAVEL_SPLIT_SPEC.md`

> **สถานะ:** Milestone 1 (โครงโปรเจกต์) เสร็จแล้ว ตอนนี้มีแค่ `/healthz` และหน้าเว็บที่บอกสถานะเซิร์ฟเวอร์

## สิ่งที่ต้องมีในเครื่อง

| เครื่องมือ | เวอร์ชัน | หมายเหตุ |
|---|---|---|
| Go | 1.26+ | |
| Node.js | 20.19+ | |
| pnpm | 10+ | |
| Docker Desktop | | ใช้รัน Postgres ตอน dev (พอร์ต **5440** เพื่อไม่ชนกับ Postgres ที่อาจติดตั้งไว้ในเครื่อง) |
| golangci-lint | 2.x | `brew install golangci-lint` |

`sqlc` กับ `goose` ไม่ต้องติดตั้งเอง เพราะล็อกเวอร์ชันไว้ใน `backend/tools/go.mod` แล้ว และสั่งผ่าน `make` (ข้างในเรียก `go tool -modfile=tools/go.mod ...`)

## เริ่มใช้งาน

```bash
make setup     # สร้าง .env จาก .env.example และติดตั้ง dependency
make dev       # เปิด postgres, รัน migration, แล้วรัน backend (:8081) + frontend (:5173)
```

เปิด http://localhost:5173 ถ้าขึ้นป้ายสีเขียว "เซิร์ฟเวอร์พร้อมใช้งาน" แปลว่า frontend → backend → Postgres เชื่อมกันครบ

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
| `make test` | `go test ./...` + `vitest` |
| `make lint` | `go vet` + `golangci-lint` + `tsc` + `oxlint` |
| `make build` | build binary และ `frontend/dist` |

## โครงสร้าง

```
backend/
  cmd/api/            จุดเริ่มโปรแกรม
  internal/config/    โหลด env
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

## Environment variables

ดู `.env.example` ทั้ง backend และ frontend อ่านจากไฟล์ `.env` ที่ root (ไฟล์นี้อยู่ใน `.gitignore` แล้ว ห้าม commit)

## ข้อตกลงในโค้ด

- เงินเก็บเป็นจำนวนเต็มในหน่วยย่อยตามมาตรฐาน ISO 4217 (THB 2 ตำแหน่ง, JPY 0 ตำแหน่ง) ห้ามใช้ float
- Error ของ API ใช้รูปแบบ `{ "error": { "code": "...", "message": "..." } }` เสมอ
