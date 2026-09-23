DROP TRIGGER IF EXISTS update_clusters_updated_at on clusters;
DROP TRIGGER IF EXISTS update_organizations_updated_at on organizations;
DROP TRIGGER IF EXISTS update_houses_updated_at on houses;
DROP TRIGGER IF EXISTS update_users_updated_at on users;
DROP FUNCTION IF EXISTS update_updated_at();