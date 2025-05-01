package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("supersecretkey")

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var users = map[string]string{
	"admin": "password123",
}

// Generate JWT Token
func generateToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 2).Unix(), // Token expires in 2 hours
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Login Handler (Fixed)
func login(c *gin.Context) {
	var user User

	// Bind JSON & check for errors
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Debugging: Print received data
	println("Received Username:", user.Username)
	println("Received Password:", user.Password)

	// Check if the user exists in the map
	storedPassword, exists := users[user.Username]
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User does not exist"})
		return
	}

	// Debugging: Print stored password
	println("Stored Password:", storedPassword)

	// Compare passwords
	if storedPassword != user.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate and return JWT token
	token, err := generateToken(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

type todo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			c.Abort()
			return
		}

		// Extract token string after "Bearer "
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		fmt.Println("Received Token:", tokenString) // Debugging log

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return jwtSecret, nil
		})

		if err != nil {
			fmt.Println("Token Parsing Error:", err) // Debugging log
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if !token.Valid {
			fmt.Println("Invalid Token Detected") // Debugging log
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

var todos = []todo{
	{ID: "1", Title: "Clean Room", Completed: false},
	{ID: "2", Title: "Read Book", Completed: false},
	{ID: "3", Title: "Record Vid", Completed: false},
}

func addTodo(context *gin.Context) {
	var newTodo todo

	if err := context.BindJSON(&newTodo); err != nil {
		return
	}

	todos = append(todos, newTodo)
	context.IndentedJSON(http.StatusCreated, newTodo)
}

func getTodos(context *gin.Context) {
	context.IndentedJSON(http.StatusOK, todos)
}

func getTodo(context *gin.Context) {
	id := context.Param("id")
	todo, err := getTodoById(id)
	if err != nil {
		context.IndentedJSON(http.StatusNotFound, gin.H{"message": "Todo Not Found"})
		return
	}

	context.IndentedJSON(http.StatusOK, todo)
}

func getTodoById(id string) (*todo, error) {
	for i, t := range todos {
		if t.ID == id {
			return &todos[i], nil
		}
	}

	return nil, errors.New("todo not found")
}

func toggleTodoStatus(context *gin.Context) {
	id := context.Param("id")
	todo, err := getTodoById(id)
	if err != nil {
		context.IndentedJSON(http.StatusNotFound, gin.H{"message": "Todo Not Found"})
		return
	}

	todo.Completed = !todo.Completed
	context.IndentedJSON(http.StatusOK, todo)
}

func deleteTodo(context *gin.Context) {
	id := context.Param("id")

	// Find the index of the todo item
	index := -1
	for i, t := range todos {
		if t.ID == id {
			index = i
			break
		}
	}

	// If not found, return a 404 error
	if index == -1 {
		context.IndentedJSON(http.StatusNotFound, gin.H{"message": "Todo Not Found"})
		return
	}

	// Remove the todo by slicing
	todos = append(todos[:index], todos[index+1:]...)

	context.IndentedJSON(http.StatusOK, gin.H{"message": "Todo deleted successfully"})
}

func main() {
	router := gin.Default()

	// Public routes
	router.POST("/login", login)

	// Protected routes
	protected := router.Group("/")
	protected.Use(authMiddleware())
	protected.GET("/todos", getTodos)
	protected.POST("/todos", addTodo)
	protected.PATCH("/todos/:id", toggleTodoStatus)
	protected.DELETE("/todos/:id", deleteTodo)

	router.Run("localhost:9090")
}
