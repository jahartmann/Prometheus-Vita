-- Node deletion was blocked because several tables referenced nodes(id) with no
-- ON DELETE action. Any migration/snapshot-policy/scheduled-action/dependency/
-- reflex-rule row for a node made DELETE FROM nodes fail with a FK violation.
-- Add ON DELETE CASCADE (drop the dependent rows with the node), and SET NULL
-- for the nullable reflex_rules.node_id (a rule may be cluster-wide).
-- Constraint names follow Postgres' default <table>_<column>_fkey convention.

ALTER TABLE vm_migrations DROP CONSTRAINT IF EXISTS vm_migrations_source_node_id_fkey;
ALTER TABLE vm_migrations ADD CONSTRAINT vm_migrations_source_node_id_fkey
  FOREIGN KEY (source_node_id) REFERENCES nodes(id) ON DELETE CASCADE;
ALTER TABLE vm_migrations DROP CONSTRAINT IF EXISTS vm_migrations_target_node_id_fkey;
ALTER TABLE vm_migrations ADD CONSTRAINT vm_migrations_target_node_id_fkey
  FOREIGN KEY (target_node_id) REFERENCES nodes(id) ON DELETE CASCADE;

ALTER TABLE snapshot_policies DROP CONSTRAINT IF EXISTS snapshot_policies_node_id_fkey;
ALTER TABLE snapshot_policies ADD CONSTRAINT snapshot_policies_node_id_fkey
  FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE;

ALTER TABLE scheduled_actions DROP CONSTRAINT IF EXISTS scheduled_actions_node_id_fkey;
ALTER TABLE scheduled_actions ADD CONSTRAINT scheduled_actions_node_id_fkey
  FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE;

ALTER TABLE vm_dependencies DROP CONSTRAINT IF EXISTS vm_dependencies_source_node_id_fkey;
ALTER TABLE vm_dependencies ADD CONSTRAINT vm_dependencies_source_node_id_fkey
  FOREIGN KEY (source_node_id) REFERENCES nodes(id) ON DELETE CASCADE;
ALTER TABLE vm_dependencies DROP CONSTRAINT IF EXISTS vm_dependencies_target_node_id_fkey;
ALTER TABLE vm_dependencies ADD CONSTRAINT vm_dependencies_target_node_id_fkey
  FOREIGN KEY (target_node_id) REFERENCES nodes(id) ON DELETE CASCADE;

ALTER TABLE reflex_rules DROP CONSTRAINT IF EXISTS reflex_rules_node_id_fkey;
ALTER TABLE reflex_rules ADD CONSTRAINT reflex_rules_node_id_fkey
  FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE SET NULL;
