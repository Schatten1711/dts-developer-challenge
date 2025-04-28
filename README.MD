# Caseworker Tasks Setup and API Docs

## Base URL
```
http://localhost:8080
```

## Using
- Golang
- PostgreSQL

## Project Setip

1. Create a `.env` file in the project root

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_username
DB_PASSWORD=your_password
DB_NAME=your_database
```
2. Initialize Go modules and download dependencies:
```bash
go mod init hmcts-developer-challenge
go mod tidy
```
The above should get all the required packages. 
Use psql to create a database as you've configured in `.env`

3. To Run tests:
```bash
go test -v
```

4. To run the server
```bash
go run main.go
```

## Endpoints

### Web Interface
```http
GET /              # HTML interface for task management
```

### REST API
```http
GET    /tasks              # List all tasks
GET    /tasks/:id          # Get a specific task
POST   /tasks              # Create a new task
GET    /tasks/:id/delete   # Delete a task
GET    /tasks/:id/complete # Mark a task as complete
GET    /search?id=:id      # Search for a task by ID
```


## Example cURL Commands

### List all tasks
```
curl http://localhost:8080/tasks
```

### Get specific task
```
curl http://localhost:8080/tasks/1
```

### Create task
```
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "title=New Task&description=Description&due_date=2025-04-28"
```

### Complete task
```
curl http://localhost:8080/tasks/1/complete
```

### Delete task
```
curl http://localhost:8080/tasks/1/delete
```

