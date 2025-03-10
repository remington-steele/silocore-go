# Implementation Steps for Multi-Tenant SAAS Application

## Database
1. **Set up PostgreSQL Database**
   - [x] Install and configure PostgreSQL.
   - [x] Configure go-migrate to run the migration scripts contained in sql/migrations.
   - [x] Develop migration scripts using go-migrate to manage schema changes and versioning.
   - [x] Create the initial database schema with a `usr` table including a `tenant_id` column using go-migrate migrations.
   - [x] Create a role table to define system roles, including ADMIN, INTERNAL, and TENANT_SUPER.
   - [x] Create a user_role table to link users to roles at the platform level.
   - [x] Create a tenant table to hold the tenants.
   - [x] Create a tenant_member table to map users to tenants.
   - [x] Create a tenant_role table to map users to tenants and roles.
   - [x] Implement Row Level Security (RLS) policies to enforce tenant isolation.

2. **JWT Token Configuration**
   - [x] Define the structure of JWT tokens to include `user_id` and optionally `tenant_id` as claims.
   - [x] Implement token generation and validation logic.

## HTTP Routing
3. **Set up Chi for HTTP Requests**
   - [x] Configure Chi to handle incoming HTTP requests.
   - [x] Define test/stub routes for tenant-specific and admin-specific endpoints.

4. **Implement Chi Middleware**
   - [x] Develop middleware for authentication, CORS, and security.
   - [x] Create custom middleware to extract `tenant_id` from JWT tokens and set it in the request context.
   - [x] Create custom middleware to extract `user_id` from JWT tokens and query the service layer for the user's roles.
   - [x] Set the "AuthContext" or other data structure into the request context for security checks.

5. **Route Security**
   - [x] Validate that the currently logged-in user, and their AuthContext, linked to their chosen tenant context, has access to the routes they are requesting.
   - [x] Ensure ADMIN users have access to all routes and tenant contexts.
   - [x] Ensure TENANT_SUPER users have access to certain tenant administrative routes if they hold the TENANT_SUPER tenant_role for the tenant context they are currently using.
   - [x] Ensure regular users with no other roles have access to tenant-specific routes if they hold a tenant_member mapping, but are not restricted from accessing certain global routes.

## Services
6. **Develop Business Logic Services**
   - [x] Define interfaces for each service to ensure modularity.
   - [x] Implement AuthService for context switching and security checks.
   - [x] Ensure services accept `tenant_id` parameters and pass them to the data access layer.

## Data Access
7. **Use idiomatic Go for Database Interactions**
   - [x] Create an order service, ensuring proper tenant isolation on all queries in the service.
   - [x] Implement data access layers with tenant isolation by including `tenant_id` in queries.
   - [x] Develop admin-specific data access layers that bypass tenant filters.

## Views
8. **Set up templ for Server-Side Rendering**
   - [x] Configure templ to render HTMLX templates.
   - [x] Develop initial page templates and partials for dynamic updates.
   - [x] Create login views and handle login requests.
   - [x] Create user order history list page.

9. **Implement HTMX for Dynamic Interactions**
   - [x] Integrate HTMX for handling dynamic interactions and partial page updates.
   - [x] Ensure Chi endpoints return updated HTML fragments for HTMX to process.

## CSS
10. **Configure Tailwind CSS**
   - [x] Set up Tailwind CSS using Tailwind CLI-based tooling for compilation and optimization.
   - [x] Develop a consistent styling strategy across the application.

## Build and Deployment
11. **Containerize Application with Docker**
   - [ ] Create Dockerfiles for consistent application deployment.
   - [ ] Set up Docker Compose for local development and testing.

12. **Implement CI/CD Pipelines**
   - [ ] Configure CI/CD pipelines to automate testing, building, and deployment.
   - [ ] Use environment variables for configuration management across environments.

---

This implementation plan will guide the development process, ensuring that each component is built according to the architecture's specifications. As features are completed, they can be marked off to track progress. 