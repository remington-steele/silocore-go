-- Set the role for executing the migration
SET ROLE silocore_admin_user;

-- Set the owner of the migration table
ALTER TABLE _migration OWNER TO silocore_admin;