ALTER TABLE vm_permissions
    ALTER COLUMN target_type TYPE VARCHAR(20);

ALTER TABLE vm_permissions
    DROP CONSTRAINT IF EXISTS vm_permissions_target_type_check;

ALTER TABLE vm_permissions
    ADD CONSTRAINT vm_permissions_target_type_check
    CHECK (target_type IN ('vm', 'group', 'node', 'environment'));
