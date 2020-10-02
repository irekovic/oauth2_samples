# Service acting on it's own behalf

In this scenario we have a simple service, that uses it's own credentials (not on behalf of a user) and executes read/write requests against
pricing engine daemon.

Note how application requests access token for use in pricingd (scope params).