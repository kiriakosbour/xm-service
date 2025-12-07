# XM Company Service - Requirements Compliance Matrix

## Overview

This document maps each requirement from the XM Golang Exercise v22.0.0 to its implementation and corresponding test cases.

---

## Technical Requirements (MUST HAVE)

### 1. CRUD Operations

| Requirement | Implementation | E2E Test | Status |
|-------------|---------------|----------|--------|
| **Create** | `POST /companies` → `handler.Create()` → `service.Create()` | `REQUIREMENT: Create Operation` section | ✅ |
| **Patch** | `PATCH /companies/{id}` → `handler.Patch()` → `service.Patch()` | `REQUIREMENT: Patch Operation` section | ✅ |
| **Delete** | `DELETE /companies/{id}` → `handler.Delete()` → `service.Delete()` | `REQUIREMENT: Delete Operation` section | ✅ |
| **Get (one)** | `GET /companies/{id}` → `handler.Get()` → `service.Get()` | `REQUIREMENT: Get Operation` section | ✅ |

### 2. Company Attributes

| Attribute | Constraints | Implementation | E2E Test | Status |
|-----------|------------|----------------|----------|--------|
| **ID** | UUID, required, auto-generated | `uuid.New()` in `service.Create()` | Validates UUID format in response | ✅ |
| **Name** | 15 chars max, required, unique | `Validate()` + DB unique constraint | Name validation section | ✅ |
| **Description** | 3000 chars max, optional | `*string` type + `Validate()` | Description validation section | ✅ |
| **Employees** | int, required, ≥0 | `Validate()` check | Employees validation section | ✅ |
| **Registered** | boolean, required | Direct boolean field | Boolean field tests | ✅ |
| **Type** | Enum (4 values), required | `CompanyType` enum + `IsValid()` | Type validation section | ✅ |

### 3. Authentication

| Requirement | Implementation | E2E Test | Status |
|-------------|---------------|----------|--------|
| Create requires auth | `middleware.JWTAuth` on POST | POST without auth → 401 | ✅ |
| Patch requires auth | `middleware.JWTAuth` on PATCH | PATCH without auth → 401 | ✅ |
| Delete requires auth | `middleware.JWTAuth` on DELETE | DELETE without auth → 401 | ✅ |
| Get is public | No middleware on GET | GET without auth → 200/404 | ✅ |

---

## Plus Requirements (BONUS)

| Requirement | Implementation | E2E Test | Status |
|-------------|---------------|----------|--------|
| **Events on mutations** | `kafka.Producer.Publish()` called on Create/Patch/Delete | Event publishing verified in unit tests | ✅ |
| **Dockerized app** | Multi-stage `Dockerfile` | Docker build succeeds | ✅ |
| **Docker for external services** | `docker-compose.yml` with PostgreSQL, Kafka, Zookeeper | Services start correctly | ✅ |
| **REST API** | Chi router with proper HTTP methods/status codes | REST validation section | ✅ |
| **JWT authentication** | `middleware.JWTAuth` (mock implementation) | Auth tests | ✅ |
| **Kafka for events** | `segmentio/kafka-go` producer | Unit tests verify event publishing | ✅ |
| **Database** | PostgreSQL with `lib/pq` driver | Integration tests | ✅ |
| **Integration tests** | `tests/integration_test.go` with test suite | Run with `-tags=integration` | ✅ |
| **Linter** | `.golangci.yml` configuration | `make lint` | ✅ |
| **Configuration file** | `internal/config/config.go` with env vars | Environment-based config | ✅ |

---

## Test Coverage Matrix

### E2E Test Script (`tests/e2e_test.sh`)

| Test Category | Tests | Assertions |
|---------------|-------|------------|
| **Pre-flight** | 2 | Server running, health endpoint |
| **Authentication** | 5 | No auth → 401, invalid auth → 401, GET public |
| **Create** | 8 | 201 status, all fields in response, UUID format |
| **Name validation** | 4 | 15 chars OK, 16 chars fail, empty fail, unique |
| **Description validation** | 3 | Optional, 3000 chars OK, 3001 fail |
| **Employees validation** | 2 | Negative fail, zero OK |
| **Type validation** | 6 | All 4 valid types OK, invalid fail, empty fail |
| **Get** | 3 | 200 with data, 404 not found, 400 invalid UUID |
| **Patch** | 6 | Single field, multiple fields, uniqueness, 404, validation |
| **Delete** | 4 | 204 success, verify deleted, 404 not found, idempotent |
| **Boolean field** | 2 | true OK, false OK |
| **REST API** | 3 | HTTP methods, status codes, JSON content type |
| **Health endpoints** | 2 | Liveness, readiness |
| **Total** | ~50 | Comprehensive coverage |

---

## Validation Rules Implementation

### Name Validation
```go
// internal/core/domain.go
if c.Name == "" {
    return errors.New("name is required")
}
if len(c.Name) > 15 {
    return errors.New("name must be 15 characters or fewer")
}
```

### Description Validation
```go
// internal/core/domain.go
if c.Description != nil && len(*c.Description) > 3000 {
    return errors.New("description must be 3000 characters or fewer")
}
```

### Employees Validation
```go
// internal/core/domain.go
if c.Employees < 0 {
    return errors.New("employees cannot be negative")
}
```

### Type Validation
```go
// internal/core/domain.go
const (
    TypeCorporations       CompanyType = "Corporations"
    TypeNonProfit          CompanyType = "NonProfit"
    TypeCooperative        CompanyType = "Cooperative"
    TypeSoleProprietorship CompanyType = "Sole Proprietorship"
)

func (ct CompanyType) IsValid() bool {
    switch ct {
    case TypeCorporations, TypeNonProfit, TypeCooperative, TypeSoleProprietorship:
        return true
    default:
        return false
    }
}
```

---

## HTTP Status Codes

| Operation | Success | Client Error | Auth Error | Not Found |
|-----------|---------|--------------|------------|-----------|
| Create | 201 | 400, 409 | 401 | - |
| Get | 200 | 400 | - | 404 |
| Patch | 200 | 400, 409 | 401 | 404 |
| Delete | 204 | 400 | 401 | 404 |

---

## Event Types (Kafka)

| Operation | Event Type | Payload |
|-----------|-----------|---------|
| Create | `CompanyCreated` | Full company object |
| Patch | `CompanyUpdated` | Full updated company object |
| Delete | `CompanyDeleted` | `{id, name}` |

---

## How to Run Tests

### Unit Tests
```bash
make test
# or
go test -v ./...
```

### Integration Tests
```bash
# Start database first
docker-compose up -d db

# Run integration tests
TEST_DB_URL="postgres://xm_user:xm_password@localhost:5432/xm_test?sslmode=disable" \
    go test -v -tags=integration ./tests/...
```

### E2E Tests
```bash
# Option 1: Full docker environment
docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit

# Option 2: Manual (start services first)
docker-compose up -d
./tests/e2e_test.sh
```

---

## Compliance Checklist

- [x] Create operation implemented and tested
- [x] Patch operation implemented and tested
- [x] Delete operation implemented and tested
- [x] Get (one) operation implemented and tested
- [x] ID is UUID, auto-generated
- [x] Name max 15 chars, required, unique
- [x] Description max 3000 chars, optional
- [x] Employees is int, required, ≥0
- [x] Registered is boolean, required
- [x] Type is enum with 4 valid values, required
- [x] Authentication on Create/Patch/Delete
- [x] Get is public (no auth required)
- [x] Events produced on mutations
- [x] Application dockerized
- [x] External services in docker-compose
- [x] REST API with proper HTTP semantics
- [x] JWT authentication (mock)
- [x] Kafka for events
- [x] PostgreSQL database
- [x] Integration tests
- [x] Linter configuration
- [x] Configuration management

**COMPLIANCE STATUS: 100%**
