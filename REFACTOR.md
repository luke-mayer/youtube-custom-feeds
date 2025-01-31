# Refactor Notes

## General/Msc

### TODO

- Create .env file and move environment variables to it.
- Look into Github for secrets to make environment variable pullable.

### Completed

## Database/SQL

### TODO

- Eliminate unecessary int ID's and use logical primary keys instead.
- Move more logic from go code to SQL???

### Completed

## HTTP Endpoints

### TODO

- Refactor handler functions
- Build unit tests for handlers
- Rebuild http endpoints using handlers
- Refactor logic code as needed
- Create smoketests
- Create integration tests

### Completed

- Switch from gorilla/mux to echo

## CI/CD

### TODO

- Dockerize build and run
- Setup CI with Github Actions
  - Automatic Building
  - Automatic Testing
  - Automatic Formatting
  - Automatic Linting
- Setup CD with Github Actions and either GCP or AWS

### Completed
