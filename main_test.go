package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	db := getDotEnv()
	connString = fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		db.User, db.Password, db.Host, db.Port, db.Name)
}

// Setup test router with actual routes from main
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.LoadHTMLGlob("static/*.html")

	router.GET("/", htmlTasks)
	router.GET("/tasks", getTasks)
	router.GET("/tasks/:id", getTaskByID)
	router.POST("/tasks", createTask)

	return router
}

// Test getting all tasks
func TestGetTasks(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/tasks", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var tasks []Task
	err := json.Unmarshal(w.Body.Bytes(), &tasks)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
}

// Test creating a new task
func TestCreateTask(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()

	form := url.Values{}
	form.Add("title", "Test Task")
	form.Add("description", "Test Description")
	form.Add("due_date", "2025-04-28")

	req, _ := http.NewRequest("POST", "/tasks",
		strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	router.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
	}

	// Clean up test data
	conn := initDatabaseConnection(connString)
	defer conn.Close(context.Background())
	_, err := conn.Exec(context.Background(),
		"DELETE FROM tasks WHERE title = 'Test Task'")
	if err != nil {
		t.Logf("Failed to clean up test task: %v", err)
	}
}

// Test getting a specific task
func TestGetTaskByID(t *testing.T) {
	// First create a task to get
	conn := initDatabaseConnection(connString)
	defer conn.Close(context.Background())

	var taskID int
	err := conn.QueryRow(context.Background(),
		`INSERT INTO tasks (title, description, due_date, completed) 
         VALUES ($1, $2, $3, $4) RETURNING id`,
		"Test Task", "Test Description", "2025-04-28", false).Scan(&taskID)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/tasks/"+strconv.Itoa(taskID), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Clean up
	_, err = conn.Exec(context.Background(),
		"DELETE FROM tasks WHERE id = $1", taskID)
	if err != nil {
		t.Logf("Failed to clean up test task: %v", err)
	}
}
