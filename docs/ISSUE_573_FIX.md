# Fix for Issue #573: Share Recreation After Deletion

## Problem Summary

When a user deletes a share (e.g., "TIMEMACHINE") and then attempts to recreate it with the same name, the system fails with the error:

```
failed to save share 'TIMEMACHINE' to repository: duplicated key not allowed
```

## Root Cause

The issue was in the `CreateShare` method in `backend/src/service/share_service.go` at line 203.

### Original Problematic Code

```go
check, err := gorm.G[dbom.ExportedShare](s.db.Debug()).
    Scopes(dbom.IncludeSoftDeleted).
    Where("name = ? and deleted_at IS NOT NULL", share.Name).
    Update(s.ctx, "deleted_at", nil)
```

This code attempted to use the GORM generic helper's `Update()` method to restore a soft-deleted record by setting `deleted_at` to `nil`. However, this approach failed because:

1. **Incorrect Query Structure**: The generic `Update()` method uses `Model(r)` where `r` is a zero-value struct, which doesn't properly target the specific record
2. **Inadequate Null Handling**: Setting `deleted_at` to `nil` via the generic Update doesn't work the same as using `gorm.Expr("NULL")`
3. **Fallthrough Error**: When the restoration failed, the code proceeded to line 231 which attempted to `Create()` a new record
4. **Primary Key Conflict**: Since the soft-deleted record still existed in the database (with `deleted_at` != NULL), attempting to create a new record with the same primary key (`name`) resulted in a "duplicated key" constraint violation

### Technical Details

GORM uses soft deletes via a `DeletedAt` field. When a record is deleted:
- The `deleted_at` timestamp is set (record is "soft deleted")
- Normal queries automatically exclude records where `deleted_at IS NOT NULL`
- The record still exists in the database with its original primary key

To "undelete" a soft-deleted record:
1. Must use `Unscoped()` to access soft-deleted records
2. Must use `UpdateColumn()` or `Update()` with `gorm.Expr("NULL")` to clear the `deleted_at` field
3. Must ensure associations are handled properly

## Solution

### Fixed Code

```go
// Check if a soft-deleted share with this name exists
var existingShare dbom.ExportedShare
err := s.db.WithContext(s.ctx).Unscoped().
    Where("name = ? AND deleted_at IS NOT NULL", share.Name).
    First(&existingShare).Error

if err == nil {
    // Found a soft-deleted share - restore it by clearing deleted_at
    slog.InfoContext(s.ctx, "Found soft-deleted share, restoring it", "share_name", share.Name)
    
    // Use UpdateColumn to bypass hooks and directly set deleted_at to NULL
    if err := s.db.WithContext(s.ctx).Model(&dbom.ExportedShare{}).Unscoped().
        Where("name = ?", share.Name).
        UpdateColumn("deleted_at", gorm.Expr("NULL")).Error; err != nil {
        slog.ErrorContext(s.ctx, "Failed to restore soft-deleted share", "share_name", share.Name, "error", err)
        return nil, errors.Wrapf(err, "failed to restore soft-deleted share '%s'", share.Name)
    }
    
    // Now update the share with the new data
    return s.UpdateShare(share.Name, share)
} else if !errors.Is(err, gorm.ErrRecordNotFound) {
    // An unexpected error occurred while checking for soft-deleted share
    slog.ErrorContext(s.ctx, "Error checking for existing soft-deleted share", "share_name", share.Name, "error", err)
    return nil, errors.Wrapf(err, "failed to check for existing share '%s'", share.Name)
}

// No soft-deleted share found, proceed with creation
slog.InfoContext(s.ctx, "No soft-deleted share found, creating new share", "share_name", share.Name)
```

### Key Improvements

1. **Explicit Check**: Uses `First()` to explicitly check if a soft-deleted record exists
2. **Proper Restoration**: Uses `UpdateColumn()` with `gorm.Expr("NULL")` to properly clear the `deleted_at` field
3. **Reuse Update Logic**: Calls `UpdateShare()` to apply the new data, ensuring all associations are properly handled
4. **Better Error Handling**: Distinguishes between "record not found" (proceed with create) and other errors
5. **Enhanced Logging**: Added contextual logging at each step for better diagnostics

## Enhanced Logging

The fix includes comprehensive logging to help diagnose issues:

### When Recreating a Deleted Share

```
INFO  Found soft-deleted share, restoring it share_name=TIMEMACHINE
INFO  Updating share share_name=TIMEMACHINE
DEBUG Retrieved share for update share_name=TIMEMACHINE users_count=0 ro_users_count=0
DEBUG Share database record updated successfully share_name=TIMEMACHINE
```

### When Deleting a Share

```
INFO  Deleting share share_name=TIMEMACHINE
DEBUG Clearing share associations before deletion share_name=TIMEMACHINE users_count=2 ro_users_count=1
DEBUG Associations cleared, performing soft delete share_name=TIMEMACHINE
INFO  Share successfully deleted (soft delete) share_name=TIMEMACHINE
```

### When Creating a New Share (No Previous Deletion)

```
INFO  No soft-deleted share found, creating new share share_name=NEWSHARE
```

## Validation Steps

To validate this fix in a deployment environment:

### 1. Create a Share

```bash
# Via the API or UI, create a share named "TEST_SHARE"
curl -X POST http://localhost:8099/api/share \
  -H "Content-Type: application/json" \
  -d '{
    "name": "TEST_SHARE",
    "disabled": false,
    "mountPointData": {
      "path": "/mnt/test",
      "deviceId": "test-device",
      "type": "ADDON"
    }
  }'
```

### 2. Delete the Share

```bash
# Delete the share
curl -X DELETE http://localhost:8099/api/share/TEST_SHARE
```

Check the logs - you should see:
```
INFO  Deleting share share_name=TEST_SHARE
DEBUG Clearing share associations before deletion ...
INFO  Share successfully deleted (soft delete) share_name=TEST_SHARE
```

### 3. Recreate the Share

```bash
# Recreate the share with the same name
curl -X POST http://localhost:8099/api/share \
  -H "Content-Type: application/json" \
  -d '{
    "name": "TEST_SHARE",
    "disabled": false,
    "mountPointData": {
      "path": "/mnt/test",
      "deviceId": "test-device",
      "type": "ADDON"
    }
  }'
```

Check the logs - you should see:
```
INFO  Found soft-deleted share, restoring it share_name=TEST_SHARE
INFO  Updating share share_name=TEST_SHARE
DEBUG Retrieved share for update share_name=TEST_SHARE ...
```

**Expected Result**: The share should be successfully recreated without any "duplicated key" error.

### 4. Verify in Database

If you have direct database access:

```sql
-- Check that the share exists and is NOT soft-deleted
SELECT name, created_at, updated_at, deleted_at 
FROM exported_shares 
WHERE name = 'TEST_SHARE';
```

The `deleted_at` field should be `NULL`.

## Database Schema Reference

### ExportedShare Table

```go
type ExportedShare struct {
    Name               string         `gorm:"primarykey"`     // Primary key
    CreatedAt          time.Time
    UpdatedAt          time.Time
    DeletedAt          gorm.DeletedAt `gorm:"index"`         // Soft delete timestamp
    // ... other fields
    Users              []SambaUser    `gorm:"many2many:user_rw_share"`
    RoUsers            []SambaUser    `gorm:"many2many:user_ro_share"`
}
```

### Join Tables for Associations

- `user_rw_share`: Maps shares to users with read-write access
- `user_ro_share`: Maps shares to users with read-only access

Both join tables have:
- `samba_user_username` (foreign key to `samba_users.username`)
- `exported_share_name` (foreign key to `exported_shares.name`)

## Pattern Reference

This fix follows the pattern established in the legacy `exported_share_repository.go` file (lines 117-123):

```go
if existingShare.DeletedAt.Valid && !share.DeletedAt.Valid {
    if err := tx.Model(&dbom.ExportedShare{}).Unscoped().
        Where("name = ?", share.Name).
        UpdateColumn("deleted_at", gorm.Expr("NULL")).Error; err != nil {
        return errors.WithDetails(err, "share_name", share.Name)
    }
}
```

## Related Files

- `backend/src/service/share_service.go`: Main fix location (CreateShare, UpdateShare, DeleteShare methods)
- `backend/src/dbom/exported_share.go`: ExportedShare model definition
- `backend/src/repository/exported_share_repository.go`: Legacy repository with working pattern

## Testing Considerations

The automated tests in `share_service_test.go` include a test case `TestCreateDeleteAndRecreateShare()` that covers this scenario. However, these tests require system-level permissions to create actual Unix/Samba users due to GORM hooks in the `SambaUser` model.

In a CI/CD environment without these permissions, the tests will fail at the user creation step, not at the share recreation step. This is a pre-existing test infrastructure limitation and not related to this fix.

## Future Improvements

1. **Mock User Creation**: Refactor tests to mock the Unix user creation to enable automated testing
2. **Integration Tests**: Create integration tests that run in a containerized environment with proper permissions
3. **Database Migrations**: Consider adding explicit database constraints/triggers to prevent orphaned join table records
4. **Hard Delete Option**: Consider adding a "hard delete" option that permanently removes records instead of soft-deleting them

## References

- Issue: https://github.com/dianlight/hassio-addons/issues/573
- GORM Soft Delete Documentation: https://gorm.io/docs/delete.html#Soft-Delete
- GORM Update Documentation: https://gorm.io/docs/update.html
