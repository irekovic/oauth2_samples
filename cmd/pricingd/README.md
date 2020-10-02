# Pricing Daemon

This component is a stub for pricing engine api. 

# OAuth2

Application is registrated in AzureAD.
Application exposes two routes:
- GET /
- POST /

Both routes are protected with JWT access tokens.

We expect that there is a role claim with 'Price.Read' value for GET request, and we expect 'Price.Write' for POST requests.

Application is totaly stateless with regard to security, no session is created for user. All data needed for authorization decision is contained in authroization header.