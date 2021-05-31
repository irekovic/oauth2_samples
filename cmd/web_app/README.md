# Fee Manager WebApplication

This is mimicing classic webapplication approach. This application exposes also several routes, but difference with backend_service engine witch is "API" component, this one provides also user interface for it's users (sort of... mind my design skills).

Endpoints exposed by this application are:

- GET / -> provides us with home page with some access tokens displayed
- GET /login -> initiates login flow with AzureAD
- POST /callback -> receives authorization code from AzureAD, and exhange it for AccessToken, next user is redirected to homepage ('/')
- GET|POST /logout -> logs out user from application and from AzureAD.