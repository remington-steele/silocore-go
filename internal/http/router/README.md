# Router Organization

This directory contains the HTTP router implementation for the SiloCore application. The router is built using the Chi router package and follows a modular organization pattern.

## File Structure

- `router.go`: Contains the base router setup with global middleware and configuration options.
- `routes.go`: Registers all application routes and organizes them into logical groups (public, admin, tenant).
- `auth.go`: Handles authentication-related routes (login, register, logout).
- `admin.go`: Handles admin-related routes (tenant management, user management).
- `tenant.go`: Handles tenant-related routes (dashboard, profile, members).
- `order/`: Contains order-specific routes and handlers.
  - `router.go`: Registers order-specific routes.
  - `handlers.go`: Implements handlers for order-related endpoints.

## Router Organization Pattern

The router organization follows these principles:

1. **Base Router Setup**: The `router.go` file provides the foundation for the router, including global middleware and configuration options.

2. **Route Registration**: The `routes.go` file registers all routes and organizes them into logical groups (public, admin, tenant).

3. **Feature-Specific Routers**: Each feature has its own router struct that only takes the dependencies it needs. This promotes modularity and makes the code more maintainable.

4. **Dependency Injection**: Each router only receives the dependencies it needs, rather than all services. This makes the code more modular and easier to test.

## Security

All routes that require authentication have the appropriate middleware applied:

- `AuthMiddleware`: Verifies the JWT token and sets the user in the request context.
- `RoleMiddleware`: Fetches and sets the user's roles in the request context.
- `RequireTenantContext`: Ensures that a tenant context is present in the request.
- `RequireAdmin`: Ensures that the user has the ADMIN role.
- `RequireTenantMember`: Ensures that the user is a member of the current tenant.
- `RequireTenantSuper`: Ensures that the user has the TENANT_SUPER role for the current tenant.

## Adding New Routes

When adding new routes:

1. For simple endpoints, add them to the appropriate section in `routes.go`.
2. For complex features, create a new router struct in a dedicated file.
3. Register the new router in `routes.go`.

## Middleware

Middleware is applied in a hierarchical manner:

1. Global middleware is applied to all routes in `router.go`.
2. Authentication middleware is applied to protected routes in `routes.go`.
3. Feature-specific middleware is applied to feature routes in their respective router files.

This ensures that all routes have the appropriate security measures applied, even if the parent router changes. 