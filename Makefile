# Load .env (if present) and export every variable to recipes.
-include .env
export

GO_TOOL  := go tool -modfile=tools/go.mod
GOOSE    := cd backend && $(GO_TOOL) goose -dir migrations postgres "$(DATABASE_URL)"

.PHONY: help setup dev dev-backend dev-frontend db-up db-down db-reset \
        migrate migrate-down migrate-status migrate-new sqlc test test-backend \
        test-frontend test-unit lint build

help: ## แสดงคำสั่งทั้งหมด
	@grep -hE '^[a-z-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

setup: ## ติดตั้ง dependency ครั้งแรก (สร้าง .env ถ้ายังไม่มี)
	@test -f .env || cp .env.example .env
	cd backend && go mod download && go mod download -modfile=tools/go.mod
	cd frontend && pnpm install

dev: db-up migrate ## รัน postgres + backend + frontend พร้อมกัน
	@trap 'kill 0' INT TERM EXIT; \
	  $(MAKE) --no-print-directory dev-backend & \
	  $(MAKE) --no-print-directory dev-frontend & \
	  wait

dev-backend: ## รัน backend อย่างเดียว
	cd backend && go run ./cmd/api

dev-frontend: ## รัน frontend อย่างเดียว
	cd frontend && pnpm dev

db-up: ## เปิด postgres (Docker) แล้วรอจนพร้อม
	docker compose up -d --wait postgres

db-down: ## ปิด postgres
	docker compose down

db-reset: ## ลบข้อมูลทั้งหมดใน DB dev แล้วสร้างใหม่
	docker compose down -v
	$(MAKE) --no-print-directory db-up migrate

migrate: ## รัน migration ทั้งหมด
	$(GOOSE) up

migrate-down: ## ย้อน migration ล่าสุด 1 ขั้น
	$(GOOSE) down

migrate-status: ## ดูสถานะ migration
	$(GOOSE) status

migrate-new: ## สร้างไฟล์ migration ใหม่: make migrate-new name=add_x
	cd backend && $(GO_TOOL) goose -dir migrations create $(name) sql

sqlc: ## generate โค้ด Go จาก queries/*.sql
	cd backend && $(GO_TOOL) sqlc generate

test: test-backend test-frontend ## รัน test ทั้งหมด

test-backend: db-up ## test backend (integration test ใช้ schema ชั่วคราวใน DB dev แล้วลบทิ้ง)
	cd backend && TEST_DATABASE_URL="$(DATABASE_URL)" go test ./...

test-unit: ## test backend เฉพาะส่วนที่ไม่ต้องใช้ DB
	cd backend && TEST_DATABASE_URL= go test ./...

test-frontend:
	cd frontend && pnpm test

lint: ## go vet + golangci-lint + tsc + oxlint
	cd backend && go vet ./... && golangci-lint run ./...
	cd frontend && pnpm typecheck && pnpm lint

build: ## build backend binary + frontend dist
	cd backend && go build -o bin/api ./cmd/api
	cd frontend && pnpm build
