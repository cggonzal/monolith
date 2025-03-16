# Monolith
A skeleton app for Go web projects. Has foundations for auth, oauth, User model, ORM (gorm), sessions, etc.

## Development
If you have `air` installed, then you can start a development server with hot reloading by running the following in the root of the repo:
```
air
```

Otherwise, just run the app with:
```
go run main.go
```

## Tests
Run the following from the root of the repo:

`go test -v ./...`

## Deployment
Run the following from the root of the repo:

`./server_management/deploy.sh`