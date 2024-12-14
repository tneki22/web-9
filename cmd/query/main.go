package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "sandbox"
)

type DatabaseProvider struct {
	db *sql.DB
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Методы работы с базой данных
func (dp *DatabaseProvider) GetUser(name string) (*User, error) {
	query := "SELECT id, name FROM users WHERE name = $1"
	row := dp.db.QueryRow(query, name)

	var user User
	err := row.Scan(&user.ID, &user.Name)
	if err == sql.ErrNoRows {
		return nil, nil // Пользователь не найден
	} else if err != nil {
		return nil, err
	}

	return &user, nil
}

func (dp *DatabaseProvider) AddUser(name string) error {
	query := "INSERT INTO users (name) VALUES ($1)"
	_, err := dp.db.Exec(query, name)
	return err
}

func main() {
	// Формирование строки подключения для PostgreSQL
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Подключение к PostgreSQL
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Проверка соединения
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Successfully connected to the database!")

	// Инициализация провайдера БД
	dbProvider := &DatabaseProvider{db: db}

	// Инициализация Echo
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Роуты
	e.GET("/api/user", func(c echo.Context) error {
		name := c.QueryParam("name")
		if name == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Parameter 'name' is required")
		}

		user, err := dbProvider.GetUser(name)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if user == nil {
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}

		return c.JSON(http.StatusOK, user)
	})

	e.POST("/api/user", func(c echo.Context) error {
		var user User
		if err := c.Bind(&user); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON format")
		}

		if err := dbProvider.AddUser(user.Name); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.String(http.StatusCreated, fmt.Sprintf("User %s added successfully", user.Name))
	})

	// Запуск сервера
	e.Logger.Fatal(e.Start(":9000"))
}
