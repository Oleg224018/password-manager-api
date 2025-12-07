package main

import (
	"fmt"
	"net/http"
	"os"
	"password-manager-api/data"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

var appData data.AppData
var masterPassword string

func main() {

	if err := loadAppData(); err != nil {
		fmt.Printf("Ошибка загрузки данных: %v\n", err)
		return
	}

	app := fiber.New(fiber.Config{
		AppName: "Password Manager API",
	})

	app.Use(logger.New())
	app.Use(cors.New())

	setupRoutes(app)

	port := ":3000"
	fmt.Printf("Сервер запущен на http://localhost%s\n", port)
	if err := app.Listen(port); err != nil {
		fmt.Printf("Ошибка запуска сервера: %v\n", err)
	}
}

func loadAppData() error {
	var err error

	masterPassword = "default_master_password"

	appData, err = data.LoadEncrypted(masterPassword)
	if err != nil {

		if _, ok := err.(*os.PathError); ok {

			appData = data.AppData{
				User:       data.User{Name: "Пользователь"},
				Categories: []data.Category{},
				Entries:    []data.PasswordEntry{},
			}

			if saveErr := data.SaveEncrypted(appData, masterPassword); saveErr != nil {
				return fmt.Errorf("не удалось создать новый файл: %w", saveErr)
			}
		} else {
			return fmt.Errorf("не удалось загрузить данные: %w", err)
		}
	}
	return nil
}

func saveAppData() error {
	return data.SaveEncrypted(appData, masterPassword)
}

func setupRoutes(app *fiber.App) {

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	app.Get("/passwords", getPasswords)
	app.Get("/passwords/:id", getPasswordByID)
	app.Post("/passwords", createPassword)
	app.Put("/passwords/:id", updatePassword)
	app.Delete("/passwords/:id", deletePassword)

	app.Get("/categories", getCategories)
	app.Put("/master-password", changeMasterPassword)
}

func getPasswords(c *fiber.Ctx) error {
	return c.JSON(appData.Entries)
}

func getPasswordByID(c *fiber.Ctx) error {
	id := c.Params("id")
	for _, entry := range appData.Entries {
		if entry.ID == id {
			return c.JSON(entry)
		}
	}
	return c.Status(http.StatusNotFound).JSON(fiber.Map{
		"error": "Пароль не найден",
	})
}

func createPassword(c *fiber.Ctx) error {
	var req struct {
		Service  string `json:"service"`
		Category string `json:"category,omitempty"`
		Length   int    `json:"length,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	if req.Service == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Название сервиса обязательно",
		})
	}

	length := req.Length
	if length < 10 {
		length = 10
	}

	category := req.Category
	if category == "" {
		category = "другое"
	}

	catID := appData.GetCategoryID(category)
	password := data.GeneratePassword(length)

	entry := data.PasswordEntry{
		ID:       data.NewID(),
		Service:  req.Service,
		Password: password,
		Category: catID,
		Created:  time.Now().Format("2006-01-02 15:04:05"),
	}

	appData.Entries = append(appData.Entries, entry)

	if err := saveAppData(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось сохранить данные",
		})
	}

	return c.Status(http.StatusCreated).JSON(entry)
}

func updatePassword(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		Service  string `json:"service,omitempty"`
		Category string `json:"category,omitempty"`
		Password string `json:"password,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	idx := appData.FindEntryIndex(id)
	if idx == -1 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Пароль не найден",
		})
	}

	current := appData.Entries[idx]

	service := req.Service
	if service == "" {
		service = current.Service
	}

	category := current.Category
	if req.Category != "" {
		category = appData.GetCategoryID(req.Category)
	}

	password := req.Password
	if password == "" {
		password = data.GeneratePassword(10)
	}

	updatedEntry := data.PasswordEntry{
		ID:       id,
		Service:  service,
		Password: password,
		Category: category,
		Created:  time.Now().Format("2006-01-02 15:04:05"),
	}

	appData.Entries[idx] = updatedEntry

	if err := saveAppData(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось сохранить данные",
		})
	}

	return c.JSON(updatedEntry)
}

func deletePassword(c *fiber.Ctx) error {
	id := c.Params("id")

	idx := appData.FindEntryIndex(id)
	if idx == -1 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Пароль не найден",
		})
	}

	appData.Entries = append(appData.Entries[:idx], appData.Entries[idx+1:]...)

	if err := saveAppData(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось сохранить данные",
		})
	}

	return c.SendStatus(http.StatusNoContent)
}

func getCategories(c *fiber.Ctx) error {
	return c.JSON(appData.Categories)
}

func changeMasterPassword(c *fiber.Ctx) error {
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
		ConfirmPassword string `json:"confirmPassword"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат запроса",
		})
	}

	if req.CurrentPassword != masterPassword {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Неверный текущий мастер-пароль",
		})
	}

	if req.NewPassword == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Новый пароль не может быть пустым",
		})
	}

	if req.NewPassword != req.ConfirmPassword {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Пароли не совпадают",
		})
	}

	oldMaster := masterPassword
	masterPassword = req.NewPassword

	if err := saveAppData(); err != nil {

		masterPassword = oldMaster
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Не удалось изменить мастер-пароль",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Мастер-пароль успешно изменён",
	})
}
