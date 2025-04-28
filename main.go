package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
)

type Task struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DueDate     string `json:"due_date"`
	Completed   bool   `json:"completed"`
}

type Database struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

var connString string
var tasks []Task

func getDotEnv() Database {
	var db Database
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	db.Host = os.Getenv("DB_HOST")
	db.Port = os.Getenv("DB_PORT")
	db.User = os.Getenv("DB_USER")
	db.Password = os.Getenv("DB_PASSWORD")
	db.Name = os.Getenv("DB_NAME")

	return db
}

func initDatabaseConnection(connString string) *pgx.Conn {
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to the database: %v", err)
	}

	return conn
}

func queryAllTasks() []Task {
	tasks = []Task{}
	conn := initDatabaseConnection(connString)

	// Get records from the DB and put them in JSON format
	records, err := conn.Query(context.Background(), `SELECT json_build_object(
		'id', id, 'title', title, 'description', description, 'due_date', due_date, 'completed', completed
		)
		AS json_data FROM Tasks WHERE ID >= 1`)
	if err != nil {
		log.Fatalf("Error reading database records: %v", err)
	}

	defer records.Close()

	for records.Next() {
		var jsonData []byte
		err := records.Scan(&jsonData) // Get JSON data from the result of our query
		if err != nil {
			log.Fatalf("Failed to fetch JSON data from record: %v", err)
		}

		var t Task
		err = json.Unmarshal(jsonData, &t)
		if err != nil {
			log.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		// If we got this far there hasn't been errors.
		// Let's put our data in a slice for easy access
		tasks = append(tasks, t)
	}
	// If we couldn't iterate over records, handle the error
	if err = records.Err(); err != nil {
		log.Fatalf("Error during record iteration: %v", err)
	}
	return tasks
}

func getTasks(c *gin.Context) {
	tasks = queryAllTasks()
	c.IndentedJSON(http.StatusOK, tasks)
}

func getTaskByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Fatalf("Error converting id string to integer: %v", err)
	}

	conn := initDatabaseConnection(connString)
	result, err := conn.Query(context.Background(), `SELECT json_build_object('id', id, 'title', title,'description', description, 'due_date', due_date, 'completed', completed) AS json_data FROM Tasks where id = $1`, id)
	if err != nil {
		log.Print("Error executing query")
	}

	t := parseJSON(result)
	if t.ID == 0 {
		c.IndentedJSON(http.StatusOK, gin.H{"message": "Task Not Found"})
		return
	}
	c.IndentedJSON(http.StatusOK, t)
}

func htmlTasks(c *gin.Context) {
	tasks := queryAllTasks()
	c.HTML(http.StatusOK, "index.html", gin.H{
		"tasks": tasks,
	})
}

func parseJSON(records pgx.Rows) (t Task) {
	for records.Next() {
		var jsonData []byte
		err := records.Scan(&jsonData)
		if err != nil {
			log.Fatalf("Failed to fetch JSON data from record: %v", err)
		}
		err = json.Unmarshal(jsonData, &t)
		return t
	}
	return
}

func deleteTaskByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Fatalf("Error converting string to integer: %v", err)
	}

	conn := initDatabaseConnection(connString)
	defer conn.Close(context.Background())
	query := `DELETE FROM tasks WHERE id = $1`
	_, err = conn.Exec(context.Background(), query, id)

	c.Redirect(http.StatusFound, "/")
}

func completeTaskByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Fatalf("Error converting string to integer: %v", err)
	}
	conn := initDatabaseConnection(connString)
	defer conn.Close(context.Background())
	query := `UPDATE tasks SET completed = true WHERE id = $1`
	conn.Exec(context.Background(), query, id)

	if err != nil {
		log.Print("Unable to complete task ")
	}

	c.Redirect(http.StatusFound, "/")
}

func createTask(c *gin.Context) {
	newTask := Task{
		Title:       c.PostForm("title"),
		Description: c.PostForm("description"),
		DueDate:     c.PostForm("due_date"),
	}

	if newTask.Title == "" || newTask.DueDate == "" {
		c.HTML(http.StatusBadRequest, "index.html", gin.H{
			"error": "Title and due date are required",
			"tasks": queryAllTasks(),
		})
		return
	}

	// Connect and create our task
	conn := initDatabaseConnection(connString)
	defer conn.Close(context.Background())

	query := `
        INSERT INTO tasks (title, description, due_date, completed) 
        VALUES ($1, $2, $3, $4) 
        RETURNING id`

	var taskID int
	err := conn.QueryRow(
		context.Background(),
		query,
		newTask.Title,
		newTask.Description,
		newTask.DueDate,
		false,
	).Scan(&taskID)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.Redirect(http.StatusFound, "/")
}

func searchTask(c *gin.Context) {
	searchID := c.Query("id")
	if searchID == "" {
		c.Redirect(http.StatusFound, "/")
		return
	}

	id, err := strconv.Atoi(searchID)
	if err != nil {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"error": "Invalid ID format",
			"tasks": queryAllTasks(),
		})
		return
	}

	conn := initDatabaseConnection(connString)
	defer conn.Close(context.Background())

	result, err := conn.Query(context.Background(), `
        SELECT json_build_object(
            'id', id,
            'title', title,
            'description', description,
            'due_date', due_date,
            'completed', completed
        ) AS json_data 
        FROM Tasks 
        WHERE id = $1`, id)
	if err != nil {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"error": "Database error",
			"tasks": queryAllTasks(),
		})
		return
	}
	defer result.Close()

	foundTask := parseJSON(result)
	if foundTask.ID == 0 {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"error": "Task not found",
			"tasks": queryAllTasks(),
		})
		return
	}

	// Show only the found task
	tasks := []Task{foundTask}
	c.HTML(http.StatusOK, "index.html", gin.H{
		"tasks": tasks,
	})
}

func main() {
	db := getDotEnv()
	connString = fmt.Sprintf("postgres://%s:%s@%s:%s/%s", db.User, db.Password, db.Host, db.Port, db.Name)
	queryAllTasks()

	// Gin config and routes
	f, _ := os.OpenFile("gin.log", os.O_RDONLY, 0666)
	gin.DefaultWriter = io.MultiWriter(f)
	router := gin.Default()

	// Template handling for our frontend
	router.LoadHTMLGlob("static/*.html")

	// Possible routes that can be taken either via API or HTML
	router.GET("/", htmlTasks)
	router.GET("/tasks", getTasks)
	router.GET("/tasks/:id", getTaskByID)
	router.GET("/tasks/:id/delete", deleteTaskByID)
	router.GET("/tasks/:id/complete", completeTaskByID)
	router.GET("/search", searchTask)
	router.POST("/tasks", createTask)
	router.POST("/", createTask)
	router.Run("localhost:8080")
}
