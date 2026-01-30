# Testing Guide

## Running Tests

### Quick Test
```bash
just test
```

### Short Tests (skip long-running tests)
```bash
just test-short
```

### Verbose Output
```bash
go test -v
```

### With Coverage
```bash
go test -cover
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Structure

### Unit Tests (`main_test.go`)
Tests individual components in isolation:

- **AuthService Tests**
  - User registration (valid, invalid usernames, duplicate users)
  - User login (correct/incorrect credentials)
  - Session creation, retrieval, and deletion

- **PasteService Tests**
  - Paste creation (public, private, anonymous)
  - Paste retrieval (access control, privacy)
  - Paste updates (ownership verification)
  - Paste deletion (ownership verification)
  - Get user pastes (filtering, ordering)
  - Can edit checks (permissions)
  - Content validation (empty, too large)

- **Utility Function Tests**
  - Hash computation (consistency, uniqueness)
  - Random filename generation

- **HTTP Handler Tests**
  - Registration endpoint
  - Login endpoint (includes cookie verification)
  - Upload endpoint (JSON and plain text)
  - Delete endpoint (with and without auth)
  - My pastes endpoint (with and without auth)

### Integration Tests (`integration_test.go`)
Tests complete user workflows:

- **Complete User Workflow**
  1. Register account
  2. Create public paste
  3. Create private paste
  4. View own pastes
  5. Edit paste
  6. Verify access control
  7. Logout

- **Delete Paste Workflow**
  - Create and delete paste
  - Verify deleted paste is inaccessible
  - Access control (only owner can delete)
  - Anonymous users cannot delete

- **Anonymous User Workflow**
  - Create public paste
  - View paste
  - Verify cannot create private paste

- **Deduplication Test**
  - Verify duplicate content handling

- **UI Features**
  - Raw paste view (?raw=1 and Accept header)
  - Multiple programming languages support
  - Edit page loading
  - My-pastes page loading
  - Session persistence

- **Error Handling**
  - Empty paste content
  - Extremely large pastes (>10MB)
  - Invalid HTTP methods
  - Non-existent pastes (404)
  - Invalid JSON handling

- **Access Control**
  - Private paste visibility (owner vs others)
  - Public paste accessibility
  - Edit permissions
  - Edit page access control

- **Legacy Upload Format**
  - Plain text uploads (backward compatibility)
  - Query parameter language selection

## Test Coverage

Current test coverage includes:
- ✅ User authentication and session management
- ✅ Paste CRUD operations (create, read, update, delete)
- ✅ Access control and privacy
- ✅ Content validation
- ✅ HTTP endpoints
- ✅ Integration workflows
- ✅ UI features (raw view, multiple languages, edit page, my-pastes page)
- ✅ Error handling (empty content, large content, invalid methods, 404s)
- ✅ Session persistence
- ✅ Legacy upload formats (plain text, query params)

All previously manual tests are now automated.

## Running Specific Tests

```bash
# Run only authentication tests
go test -run TestAuthService

# Run only paste service tests
go test -run TestPasteService

# Run only HTTP handler tests
go test -run TestHTTPHandlers

# Run integration tests
go test -run TestUserWorkflow
go test -run TestDeletePasteWorkflow
go test -run TestUIFeatures
go test -run TestErrorHandling
go test -run TestAccessControl
```

## Automated Test Coverage

All functionality is now covered by automated tests. The following areas that were previously tested manually are now automated:

### ✅ User Registration & Login
- Valid and invalid username/password combinations
- Duplicate username rejection
- Session cookie management
- User info in responses

### ✅ Paste Operations
- Create public/private pastes (logged in and anonymous)
- Multiple programming languages
- View pastes (with access control)
- Raw paste view (`?raw=1` and `Accept: text/plain`)
- Edit own pastes
- Delete own pastes
- Access control enforcement

### ✅ UI Pages
- My Pastes page (with metadata and privacy badges)
- Edit page (owner-only access)
- Session persistence across requests

### ✅ Error Handling
- Empty paste content rejection
- Large paste rejection (>10MB)
- Invalid HTTP methods
- Non-existent paste 404s
- Invalid UTF-8 handling

### ✅ Anonymous Users
- Can create public pastes
- Cannot create private pastes
- Cannot edit any pastes
- Cannot delete any pastes

### ✅ Legacy Compatibility
- Plain text uploads (non-JSON)
- Query parameter language selection

## Manual Testing (Optional)

The following can still be tested manually for visual/UX verification:

### Browser Testing
- [ ] Verify syntax highlighting renders correctly for different languages
- [ ] Test "Copy" button functionality  
- [ ] Verify responsive design on mobile devices
- [ ] Test keyboard shortcuts (Ctrl+S to save in edit mode)

### Performance Testing

### Load Testing with `ab` (Apache Bench)
```bash
# Test paste creation
ab -n 1000 -c 10 -p paste.txt -T "text/plain" http://localhost:3001/upload

# Test paste viewing
ab -n 10000 -c 50 http://localhost:3001/p/PASTE_ID
```

### Benchmark Tests
Create `*_bench_test.go` files:
```go
func BenchmarkPasteCreation(b *testing.B) {
    for i := 0; i < b.N; i++ {
        pasteService.CreatePaste("content", "text", false, nil)
    }
}
```

Run benchmarks:
```bash
go test -bench=.
```

## Troubleshooting

### Tests Fail with "database locked"
- SQLite doesn't handle high concurrency well
- Use separate database file for each test
- Or use `:memory:` database (already done)

### Session Tests Fail
- Ensure cookies are properly set in test requests
- Use `req.AddCookie(cookie)` not `req.Header.Set()`

### Template Tests Fail
- Verify `//go:embed templates` is present
- Check template syntax errors
- Ensure all template files exist

## Test Data Cleanup

Tests use in-memory database (`:memory:`) so no cleanup needed.
For file-based tests:
```bash
just clean
```
