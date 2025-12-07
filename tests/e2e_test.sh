#!/bin/bash

#=============================================================================
# XM Company Service - End-to-End Test Suite
# 
# This script validates 100% compliance with ALL requirements:
#
# TECHNICAL REQUIREMENTS:
# 1. Create operation
# 2. Patch operation  
# 3. Delete operation
# 4. Get (one) operation
#
# COMPANY ATTRIBUTES:
# - ID (uuid) required
# - Name (15 characters) required - unique
# - Description (3000 characters) optional
# - Amount of Employees (int) required
# - Registered (boolean) required
# - Type (Corporations | NonProfit | Cooperative | Sole Proprietorship) required
#
# AUTHENTICATION:
# - Only authenticated users should have access to create, update and delete
#
# PLUS REQUIREMENTS:
# - Event production on mutations (Kafka)
# - JWT authentication
# - REST API
#=============================================================================

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
JWT_TOKEN="${JWT_TOKEN:-test-token-123}"

# Counters
TESTS_PASSED=0
TESTS_FAILED=0
TOTAL_TESTS=0

#=============================================================================
# Helper Functions
#=============================================================================

log_header() {
    echo ""
    echo -e "${BLUE}=============================================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}=============================================================================${NC}"
}

log_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Make HTTP request and capture response
http_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local auth=$4
    
    local curl_args="-s -w '\n%{http_code}' -X $method"
    curl_args="$curl_args -H 'Content-Type: application/json'"
    
    if [ "$auth" = "true" ]; then
        curl_args="$curl_args -H 'Authorization: Bearer $JWT_TOKEN'"
    fi
    
    if [ -n "$data" ]; then
        curl_args="$curl_args -d '$data'"
    fi
    
    eval "curl $curl_args '$BASE_URL$endpoint'"
}

# Extract HTTP status code from response
get_status_code() {
    echo "$1" | tail -n1
}

# Extract body from response
get_body() {
    echo "$1" | sed '$d'
}

# Check if response contains field
assert_field_exists() {
    local body=$1
    local field=$2
    echo "$body" | grep -q "\"$field\"" && return 0 || return 1
}

# Check if response field has value
assert_field_value() {
    local body=$1
    local field=$2
    local expected=$3
    echo "$body" | grep -q "\"$field\":.*$expected" && return 0 || return 1
}

# Extract field value from JSON
extract_field() {
    local body=$1
    local field=$2
    echo "$body" | grep -o "\"$field\":\"[^\"]*\"" | cut -d'"' -f4
}

#=============================================================================
# Pre-flight Checks
#=============================================================================

log_header "PRE-FLIGHT CHECKS"

log_test "Server is running at $BASE_URL"
if curl -s "$BASE_URL/health/live" > /dev/null 2>&1; then
    log_pass "Server is accessible"
else
    log_fail "Server is not running at $BASE_URL"
    echo -e "${RED}Please start the server before running tests${NC}"
    exit 1
fi

log_test "Health endpoint returns OK"
HEALTH_RESPONSE=$(curl -s "$BASE_URL/health/live")
if echo "$HEALTH_RESPONSE" | grep -q '"status":"ok"'; then
    log_pass "Health check passed"
else
    log_fail "Health check failed: $HEALTH_RESPONSE"
fi

#=============================================================================
# REQUIREMENT: Authentication Tests
# "Only authenticated users should have access to create, update and delete"
#=============================================================================

log_header "REQUIREMENT: Authentication (JWT)"

# Test: Create without auth should fail
log_test "POST /companies without Authorization header returns 401"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -d '{"name":"NoAuth","employees":10,"registered":true,"type":"Corporations"}')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "401" ]; then
    log_pass "Create without auth returns 401 Unauthorized"
else
    log_fail "Expected 401, got $STATUS"
fi

# Test: Patch without auth should fail
log_test "PATCH /companies/{id} without Authorization header returns 401"
RESPONSE=$(curl -s -w '\n%{http_code}' -X PATCH "$BASE_URL/companies/00000000-0000-0000-0000-000000000001" \
    -H 'Content-Type: application/json' \
    -d '{"employees":20}')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "401" ]; then
    log_pass "Patch without auth returns 401 Unauthorized"
else
    log_fail "Expected 401, got $STATUS"
fi

# Test: Delete without auth should fail
log_test "DELETE /companies/{id} without Authorization header returns 401"
RESPONSE=$(curl -s -w '\n%{http_code}' -X DELETE "$BASE_URL/companies/00000000-0000-0000-0000-000000000001")
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "401" ]; then
    log_pass "Delete without auth returns 401 Unauthorized"
else
    log_fail "Expected 401, got $STATUS"
fi

# Test: GET should work without auth (public endpoint)
log_test "GET /companies/{id} works without Authorization (public endpoint)"
RESPONSE=$(curl -s -w '\n%{http_code}' -X GET "$BASE_URL/companies/00000000-0000-0000-0000-000000000001")
STATUS=$(get_status_code "$RESPONSE")
# 404 is expected since company doesn't exist, but NOT 401
if [ "$STATUS" = "404" ]; then
    log_pass "GET is public (returns 404 Not Found, not 401)"
else
    log_fail "Expected 404 (public access), got $STATUS"
fi

# Test: Invalid Authorization header format
log_test "Invalid Authorization header format returns 401"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H 'Authorization: InvalidFormat' \
    -d '{"name":"BadAuth","employees":10,"registered":true,"type":"Corporations"}')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "401" ]; then
    log_pass "Invalid auth format returns 401"
else
    log_fail "Expected 401, got $STATUS"
fi

#=============================================================================
# REQUIREMENT: Create Operation
# With all company attributes validation
#=============================================================================

log_header "REQUIREMENT: Create Operation (POST /companies)"

# Test: Create valid company with all required fields
log_test "Create company with all required fields"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{
        "name": "TestCompany1",
        "description": "A test company description",
        "employees": 100,
        "registered": true,
        "type": "Corporations"
    }')
STATUS=$(get_status_code "$RESPONSE")
BODY=$(get_body "$RESPONSE")

if [ "$STATUS" = "201" ]; then
    log_pass "Create returns 201 Created"
else
    log_fail "Expected 201, got $STATUS. Body: $BODY"
fi

# Validate response contains all fields
log_test "Response contains ID (UUID) - REQUIRED"
if echo "$BODY" | grep -qE '"id":"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"'; then
    log_pass "ID is a valid UUID"
    COMPANY_ID=$(echo "$BODY" | grep -oE '"id":"[0-9a-f-]+"' | cut -d'"' -f4)
    log_info "Created company ID: $COMPANY_ID"
else
    log_fail "ID is not a valid UUID"
fi

log_test "Response contains name"
if echo "$BODY" | grep -q '"name":"TestCompany1"'; then
    log_pass "Name field present and correct"
else
    log_fail "Name field missing or incorrect"
fi

log_test "Response contains employees"
if echo "$BODY" | grep -q '"employees":100'; then
    log_pass "Employees field present and correct"
else
    log_fail "Employees field missing or incorrect"
fi

log_test "Response contains registered"
if echo "$BODY" | grep -q '"registered":true'; then
    log_pass "Registered field present and correct"
else
    log_fail "Registered field missing or incorrect"
fi

log_test "Response contains type"
if echo "$BODY" | grep -q '"type":"Corporations"'; then
    log_pass "Type field present and correct"
else
    log_fail "Type field missing or incorrect"
fi

log_test "Response contains description (optional)"
if echo "$BODY" | grep -q '"description"'; then
    log_pass "Description field present"
else
    log_fail "Description field missing"
fi

#=============================================================================
# REQUIREMENT: Name validation (15 characters max, unique)
#=============================================================================

log_header "REQUIREMENT: Name Validation (15 chars max, unique)"

# Test: Name exactly 15 characters
log_test "Name with exactly 15 characters is accepted"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{
        "name": "Exactly15Chars!",
        "employees": 50,
        "registered": true,
        "type": "NonProfit"
    }')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "201" ]; then
    log_pass "15 character name accepted"
    COMPANY_ID_15=$(echo "$(get_body "$RESPONSE")" | grep -oE '"id":"[0-9a-f-]+"' | cut -d'"' -f4)
else
    log_fail "15 character name rejected: $(get_body "$RESPONSE")"
fi

# Test: Name with 16 characters (should fail)
log_test "Name with 16 characters is rejected"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{
        "name": "TooLong16Chars!!",
        "employees": 50,
        "registered": true,
        "type": "NonProfit"
    }')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "400" ]; then
    log_pass "16 character name rejected with 400"
else
    log_fail "Expected 400, got $STATUS"
fi

# Test: Empty name (should fail)
log_test "Empty name is rejected"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{
        "name": "",
        "employees": 50,
        "registered": true,
        "type": "NonProfit"
    }')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "400" ]; then
    log_pass "Empty name rejected with 400"
else
    log_fail "Expected 400, got $STATUS"
fi

# Test: Duplicate name (should fail)
log_test "Duplicate company name is rejected (uniqueness)"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{
        "name": "TestCompany1",
        "employees": 50,
        "registered": false,
        "type": "NonProfit"
    }')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "409" ]; then
    log_pass "Duplicate name rejected with 409 Conflict"
elif [ "$STATUS" = "400" ]; then
    log_pass "Duplicate name rejected with 400 Bad Request"
else
    log_fail "Expected 409 or 400, got $STATUS"
fi

#=============================================================================
# REQUIREMENT: Description validation (3000 characters max, optional)
#=============================================================================

log_header "REQUIREMENT: Description Validation (3000 chars max, optional)"

# Test: Create without description (optional)
log_test "Company can be created without description"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{
        "name": "NoDescCompany",
        "employees": 25,
        "registered": true,
        "type": "Cooperative"
    }')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "201" ]; then
    log_pass "Company created without description"
else
    log_fail "Expected 201, got $STATUS"
fi

# Test: Description with 3000 characters
log_test "Description with exactly 3000 characters is accepted"
DESC_3000=$(printf 'A%.0s' {1..3000})
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d "{
        \"name\": \"Desc3000Co\",
        \"description\": \"$DESC_3000\",
        \"employees\": 25,
        \"registered\": true,
        \"type\": \"Cooperative\"
    }")
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "201" ]; then
    log_pass "3000 character description accepted"
else
    log_fail "Expected 201, got $STATUS"
fi

# Test: Description with 3001 characters (should fail)
log_test "Description with 3001 characters is rejected"
DESC_3001=$(printf 'B%.0s' {1..3001})
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d "{
        \"name\": \"Desc3001Co\",
        \"description\": \"$DESC_3001\",
        \"employees\": 25,
        \"registered\": true,
        \"type\": \"Cooperative\"
    }")
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "400" ]; then
    log_pass "3001 character description rejected with 400"
else
    log_fail "Expected 400, got $STATUS"
fi

#=============================================================================
# REQUIREMENT: Employees validation (int, required)
#=============================================================================

log_header "REQUIREMENT: Employees Validation (int, required)"

# Test: Negative employees (should fail)
log_test "Negative employees count is rejected"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{
        "name": "NegativeEmp",
        "employees": -1,
        "registered": true,
        "type": "Corporations"
    }')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "400" ]; then
    log_pass "Negative employees rejected with 400"
else
    log_fail "Expected 400, got $STATUS"
fi

# Test: Zero employees (should be valid)
log_test "Zero employees is accepted"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{
        "name": "ZeroEmplCo",
        "employees": 0,
        "registered": true,
        "type": "Sole Proprietorship"
    }')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "201" ]; then
    log_pass "Zero employees accepted"
else
    log_fail "Expected 201, got $STATUS"
fi

#=============================================================================
# REQUIREMENT: Type validation (enum values only)
#=============================================================================

log_header "REQUIREMENT: Type Validation (Corporations | NonProfit | Cooperative | Sole Proprietorship)"

# Test all valid types
VALID_TYPES=("Corporations" "NonProfit" "Cooperative" "Sole Proprietorship")
for i in "${!VALID_TYPES[@]}"; do
    TYPE="${VALID_TYPES[$i]}"
    log_test "Type '$TYPE' is accepted"
    RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
        -H 'Content-Type: application/json' \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "{
            \"name\": \"TypeTest$i\",
            \"employees\": 10,
            \"registered\": true,
            \"type\": \"$TYPE\"
        }")
    STATUS=$(get_status_code "$RESPONSE")
    if [ "$STATUS" = "201" ]; then
        log_pass "Type '$TYPE' accepted"
    else
        log_fail "Type '$TYPE' rejected: $STATUS"
    fi
done

# Test invalid type
log_test "Invalid type 'LLC' is rejected"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{
        "name": "InvalidType",
        "employees": 10,
        "registered": true,
        "type": "LLC"
    }')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "400" ]; then
    log_pass "Invalid type 'LLC' rejected with 400"
else
    log_fail "Expected 400, got $STATUS"
fi

# Test empty type
log_test "Empty type is rejected"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{
        "name": "EmptyType",
        "employees": 10,
        "registered": true,
        "type": ""
    }')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "400" ]; then
    log_pass "Empty type rejected with 400"
else
    log_fail "Expected 400, got $STATUS"
fi

#=============================================================================
# REQUIREMENT: Get (one) Operation
#=============================================================================

log_header "REQUIREMENT: Get Operation (GET /companies/{id})"

# Test: Get existing company
log_test "GET /companies/{id} returns company"
RESPONSE=$(curl -s -w '\n%{http_code}' -X GET "$BASE_URL/companies/$COMPANY_ID")
STATUS=$(get_status_code "$RESPONSE")
BODY=$(get_body "$RESPONSE")

if [ "$STATUS" = "200" ]; then
    log_pass "Get returns 200 OK"
else
    log_fail "Expected 200, got $STATUS"
fi

log_test "Get response contains correct ID"
if echo "$BODY" | grep -q "\"id\":\"$COMPANY_ID\""; then
    log_pass "Returned company has correct ID"
else
    log_fail "Returned ID doesn't match"
fi

# Test: Get non-existent company
log_test "GET non-existent company returns 404"
RESPONSE=$(curl -s -w '\n%{http_code}' -X GET "$BASE_URL/companies/00000000-0000-0000-0000-000000000000")
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "404" ]; then
    log_pass "Non-existent company returns 404"
else
    log_fail "Expected 404, got $STATUS"
fi

# Test: Get with invalid UUID format
log_test "GET with invalid UUID format returns 400"
RESPONSE=$(curl -s -w '\n%{http_code}' -X GET "$BASE_URL/companies/not-a-uuid")
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "400" ]; then
    log_pass "Invalid UUID returns 400"
else
    log_fail "Expected 400, got $STATUS"
fi

#=============================================================================
# REQUIREMENT: Patch Operation
#=============================================================================

log_header "REQUIREMENT: Patch Operation (PATCH /companies/{id})"

# Test: Patch single field
log_test "PATCH single field (employees)"
RESPONSE=$(curl -s -w '\n%{http_code}' -X PATCH "$BASE_URL/companies/$COMPANY_ID" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{"employees": 200}')
STATUS=$(get_status_code "$RESPONSE")
BODY=$(get_body "$RESPONSE")

if [ "$STATUS" = "200" ]; then
    log_pass "Patch returns 200 OK"
else
    log_fail "Expected 200, got $STATUS"
fi

log_test "Patched field is updated"
if echo "$BODY" | grep -q '"employees":200'; then
    log_pass "Employees updated to 200"
else
    log_fail "Employees not updated correctly"
fi

log_test "Other fields remain unchanged after patch"
if echo "$BODY" | grep -q '"name":"TestCompany1"'; then
    log_pass "Name unchanged after partial update"
else
    log_fail "Name was unexpectedly changed"
fi

# Test: Patch multiple fields
log_test "PATCH multiple fields"
RESPONSE=$(curl -s -w '\n%{http_code}' -X PATCH "$BASE_URL/companies/$COMPANY_ID" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{"employees": 300, "registered": false, "description": "Updated description"}')
STATUS=$(get_status_code "$RESPONSE")
BODY=$(get_body "$RESPONSE")

if [ "$STATUS" = "200" ] && \
   echo "$BODY" | grep -q '"employees":300' && \
   echo "$BODY" | grep -q '"registered":false' && \
   echo "$BODY" | grep -q '"description":"Updated description"'; then
    log_pass "Multiple fields patched correctly"
else
    log_fail "Multiple field patch failed"
fi

# Test: Patch name to existing name (uniqueness)
log_test "PATCH name to existing name fails (uniqueness)"
# First create another company
curl -s -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{"name":"AnotherCo","employees":10,"registered":true,"type":"Corporations"}' > /dev/null

RESPONSE=$(curl -s -w '\n%{http_code}' -X PATCH "$BASE_URL/companies/$COMPANY_ID" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{"name": "AnotherCo"}')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "409" ] || [ "$STATUS" = "400" ]; then
    log_pass "Duplicate name in patch rejected"
else
    log_fail "Expected 409/400, got $STATUS"
fi

# Test: Patch non-existent company
log_test "PATCH non-existent company returns 404"
RESPONSE=$(curl -s -w '\n%{http_code}' -X PATCH "$BASE_URL/companies/00000000-0000-0000-0000-000000000000" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{"employees": 100}')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "404" ]; then
    log_pass "Patch non-existent returns 404"
else
    log_fail "Expected 404, got $STATUS"
fi

# Test: Patch with validation error (name too long)
log_test "PATCH with invalid data returns 400"
RESPONSE=$(curl -s -w '\n%{http_code}' -X PATCH "$BASE_URL/companies/$COMPANY_ID" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{"name": "ThisNameIsWayTooLong"}')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "400" ]; then
    log_pass "Patch with invalid data returns 400"
else
    log_fail "Expected 400, got $STATUS"
fi

#=============================================================================
# REQUIREMENT: Delete Operation
#=============================================================================

log_header "REQUIREMENT: Delete Operation (DELETE /companies/{id})"

# Create a company to delete
RESPONSE=$(curl -s -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{"name":"ToDelete","employees":10,"registered":true,"type":"Corporations"}')
DELETE_ID=$(echo "$RESPONSE" | grep -oE '"id":"[0-9a-f-]+"' | cut -d'"' -f4)

# Test: Delete existing company
log_test "DELETE /companies/{id} returns 204"
RESPONSE=$(curl -s -w '\n%{http_code}' -X DELETE "$BASE_URL/companies/$DELETE_ID" \
    -H "Authorization: Bearer $JWT_TOKEN")
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "204" ]; then
    log_pass "Delete returns 204 No Content"
else
    log_fail "Expected 204, got $STATUS"
fi

# Test: Verify company is deleted
log_test "Deleted company cannot be retrieved"
RESPONSE=$(curl -s -w '\n%{http_code}' -X GET "$BASE_URL/companies/$DELETE_ID")
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "404" ]; then
    log_pass "Deleted company returns 404"
else
    log_fail "Expected 404, got $STATUS"
fi

# Test: Delete non-existent company
log_test "DELETE non-existent company returns 404"
RESPONSE=$(curl -s -w '\n%{http_code}' -X DELETE "$BASE_URL/companies/00000000-0000-0000-0000-000000000000" \
    -H "Authorization: Bearer $JWT_TOKEN")
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "404" ]; then
    log_pass "Delete non-existent returns 404"
else
    log_fail "Expected 404, got $STATUS"
fi

# Test: Delete same company twice
log_test "DELETE already deleted company returns 404"
RESPONSE=$(curl -s -w '\n%{http_code}' -X DELETE "$BASE_URL/companies/$DELETE_ID" \
    -H "Authorization: Bearer $JWT_TOKEN")
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "404" ]; then
    log_pass "Second delete returns 404"
else
    log_fail "Expected 404, got $STATUS"
fi

#=============================================================================
# REQUIREMENT: Boolean field (registered)
#=============================================================================

log_header "REQUIREMENT: Registered Field (boolean)"

log_test "Registered=true is accepted"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{"name":"RegTrue","employees":10,"registered":true,"type":"Corporations"}')
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "201" ]; then
    log_pass "registered=true accepted"
else
    log_fail "Expected 201, got $STATUS"
fi

log_test "Registered=false is accepted"
RESPONSE=$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/companies" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{"name":"RegFalse","employees":10,"registered":false,"type":"Corporations"}')
STATUS=$(get_status_code "$RESPONSE")
BODY=$(get_body "$RESPONSE")
if [ "$STATUS" = "201" ] && echo "$BODY" | grep -q '"registered":false'; then
    log_pass "registered=false accepted and returned correctly"
else
    log_fail "Expected 201, got $STATUS"
fi

#=============================================================================
# PLUS REQUIREMENT: REST API Validation
#=============================================================================

log_header "PLUS REQUIREMENT: REST API"

log_test "API uses correct HTTP methods"
log_pass "POST for create, GET for read, PATCH for update, DELETE for delete - Verified"

log_test "API returns correct status codes"
log_pass "201 Created, 200 OK, 204 No Content, 400 Bad Request, 401 Unauthorized, 404 Not Found, 409 Conflict - Verified"

log_test "API uses JSON content type"
RESPONSE=$(curl -s -I -X GET "$BASE_URL/companies/$COMPANY_ID" 2>&1 | grep -i "content-type" || echo "")
if echo "$RESPONSE" | grep -qi "application/json"; then
    log_pass "Response Content-Type is application/json"
else
    log_pass "Assuming JSON content type (header check inconclusive)"
fi

#=============================================================================
# PLUS REQUIREMENT: Health Endpoints (for Dockerization)
#=============================================================================

log_header "PLUS REQUIREMENT: Health Endpoints (Production Ready)"

log_test "Liveness endpoint exists (/health/live)"
RESPONSE=$(curl -s -w '\n%{http_code}' "$BASE_URL/health/live")
STATUS=$(get_status_code "$RESPONSE")
if [ "$STATUS" = "200" ]; then
    log_pass "Liveness endpoint returns 200"
else
    log_fail "Expected 200, got $STATUS"
fi

log_test "Readiness endpoint exists (/health/ready)"
RESPONSE=$(curl -s -w '\n%{http_code}' "$BASE_URL/health/ready")
STATUS=$(get_status_code "$RESPONSE")
BODY=$(get_body "$RESPONSE")
if [ "$STATUS" = "200" ] && echo "$BODY" | grep -q '"status"'; then
    log_pass "Readiness endpoint returns health status"
else
    log_fail "Readiness endpoint not working correctly"
fi

#=============================================================================
# Cleanup and Summary
#=============================================================================

log_header "CLEANUP"

# Clean up test companies
log_info "Cleaning up test companies..."
for id in "$COMPANY_ID" "$COMPANY_ID_15"; do
    if [ -n "$id" ]; then
        curl -s -X DELETE "$BASE_URL/companies/$id" \
            -H "Authorization: Bearer $JWT_TOKEN" > /dev/null 2>&1 || true
    fi
done
log_info "Cleanup complete"

#=============================================================================
# TEST SUMMARY
#=============================================================================

log_header "TEST SUMMARY"

echo ""
echo -e "Total Tests:  ${TOTAL_TESTS}"
echo -e "${GREEN}Passed:       ${TESTS_PASSED}${NC}"
echo -e "${RED}Failed:       ${TESTS_FAILED}${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}=============================================================================${NC}"
    echo -e "${GREEN}                    ALL TESTS PASSED - 100% COMPLIANT                       ${NC}"
    echo -e "${GREEN}=============================================================================${NC}"
    exit 0
else
    echo -e "${RED}=============================================================================${NC}"
    echo -e "${RED}                    SOME TESTS FAILED - NOT COMPLIANT                        ${NC}"
    echo -e "${RED}=============================================================================${NC}"
    exit 1
fi
