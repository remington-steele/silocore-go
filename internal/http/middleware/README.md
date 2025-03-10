# Route Security Implementation

This document outlines the route security implementation for the multi-tenant SAAS application.

## Middleware Components

### Authentication Middleware

- `AuthMiddleware`: Validates JWT tokens and sets user and tenant information in the request context.
  - Extracts the JWT token from the Authorization header
  - Validates the token using the JWTService
  - Sets user ID, username, and tenant ID (if present) in the request context

### Role Middleware

- `RoleMiddleware`: Fetches and sets user roles in the request context.
  - Fetches system-wide roles for the user
  - If a tenant context is present:
    - Checks if the user is a member of the tenant
    - For admin users, allows access to any tenant context
    - For non-admin users, requires tenant membership
    - Fetches tenant-specific roles for tenant members
    - Merges system-wide and tenant-specific roles

### Access Control Middleware

- `RequireAdmin`: Ensures the user has the ADMIN role.
  - Checks if the user has the ADMIN role in the request context
  - Returns 403 Forbidden if the user is not an admin

- `RequireTenantContext`: Ensures a tenant context is present.
  - Checks if a tenant ID is present in the request context
  - Returns 403 Forbidden if no tenant context is found

- `RequireTenantMember`: Ensures the user is a member of the current tenant.
  - Checks if a tenant ID is present in the request context
  - For admin users, allows access to any tenant
  - For non-admin users, checks if the user is a member of the tenant
  - Returns 403 Forbidden if the user is not a member of the tenant

- `RequireTenantSuper`: Ensures the user has the TENANT_SUPER role for the current tenant.
  - Checks if a tenant ID is present in the request context
  - For admin users, allows access to any tenant admin functionality
  - For non-admin users, checks if the user has the TENANT_SUPER role
  - Returns 403 Forbidden if the user does not have the TENANT_SUPER role

### Utility Middleware

- `TenantIDFromURL`: Extracts the tenant ID from the URL parameter and adds it to the context.
  - Extracts the tenant ID from the URL parameter
  - Converts the tenant ID to int64
  - Sets the tenant ID in the request context

## Route Security Rules

1. **Admin Users**:
   - Have access to all routes and all tenant contexts
   - Can perform administrative functions across the entire platform
   - Can switch between tenant contexts

2. **Tenant Super Users**:
   - Have access to administrative functions within their tenant
   - Must have the TENANT_SUPER role for the specific tenant
   - Cannot access administrative functions for other tenants

3. **Regular Users**:
   - Must be members of a tenant to access tenant-specific routes
   - Can access global routes (e.g., user profile) regardless of tenant membership
   - Cannot access administrative functions

## Implementation in Router

The router uses these middleware components to secure routes:

1. **Public Routes**: No authentication required
   - Login, registration, etc.

2. **Protected Routes**: Require authentication
   - Apply `AuthMiddleware` and `RoleMiddleware` to all protected routes

3. **Admin Routes**: Require ADMIN role
   - Apply `RequireAdmin` middleware to admin routes

4. **Tenant Routes**: Require tenant context and membership
   - Apply `RequireTenantContext` and `RequireTenantMember` middleware to tenant routes

5. **Tenant Admin Routes**: Require TENANT_SUPER role
   - Apply `RequireTenantSuper` middleware to tenant admin routes 