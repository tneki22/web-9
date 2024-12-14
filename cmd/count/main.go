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

type Counter struct {
	ID    int `json:"id"`
	Value int `json:"value"`
}

func (dp *DatabaseProvider) GetCounter() (*Counter, error) {
	query := "SELECT id, value FROM counter LIMIT 1"
	row := dp.db.QueryRow(query)

	var counter Counter
	err := row.Scan(&counter.ID, &counter.Value)
	if err == sql.ErrNoRows {
		return nil, nil // Счетчик не найден
	} else if err != nil {
		return nil, err
	}

	return &counter, nil
}

func (dp *DatabaseProvider) IncreaseCounter(value int) error {
	query := "UPDATE counter SET value = value + $1 WHERE id = 1"
	_, err := dp.db.Exec(query, value)
	return err
}

func (dp *DatabaseProvider) initializeCounter() error {
	var count Counter

	query := "SELECT id FROM counter LIMIT 1"
	err := dp.db.QueryRow(query).Scan(&count.ID)
	if err == sql.ErrNoRows {
		// Счетчик не найден, добавляем начальное значение
		insertQuery := "INSERT INTO counter (value) VALUES ($1)"
		_, err := dp.db.Exec(insertQuery, 0)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// Подключение к PostgreSQL
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Successfully connected to the database!")

	// Инициализация провайдера БД
	dbProvider := &DatabaseProvider{db: db}

	// Инициализация счетчика, если он отсутствует
	if err := dbProvider.initializeCounter(); err != nil {
		log.Fatal(err)
	}

	// Инициализация Echo
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Роуты
	e.GET("/count", func(c echo.Context) error {
		counter, err := dbProvider.GetCounter()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		if counter == nil {
			return echo.NewHTTPError(http.StatusNotFound, "Counter not found")
		}

		return c.JSON(http.StatusOK, counter)
	})

	e.POST("/count", func(c echo.Context) error {
		var requestBody struct {
			Count int `json:"count"`
		}

		if err := c.Bind(&requestBody); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON format")
		}

		if requestBody.Count == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Parameter 'count' is required")
		}

		if err := dbProvider.IncreaseCounter(requestBody.Count); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.String(http.StatusOK, fmt.Sprintf("Counter increased by %d", requestBody.Count))
	})

	// Запуск сервера
	e.Logger.Fatal(e.Start(":3333"))
}
