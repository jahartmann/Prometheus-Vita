ALTER TABLE resource_recommendations ADD COLUMN IF NOT EXISTS vm_context TEXT DEFAULT '';
ALTER TABLE resource_recommendations ADD COLUMN IF NOT EXISTS context_reason TEXT DEFAULT '';
