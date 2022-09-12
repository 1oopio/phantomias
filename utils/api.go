package utils

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
)

var (
	ErrPoolNotFound = errors.New("pool not found")
	ErrInvalidRange = errors.New("invalid range")
)

type APIError struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

func SendAPIError(c *fiber.Ctx, code int, err error) error {
	return c.Status(code).JSON(APIError{
		Error: err.Error(),
		Code:  code,
	},
	)
}

func HandleMCError(c *fiber.Ctx, err error) error {
	log.Println("error querying miningcore:", err)
	return SendAPIError(c, 500, fiber.ErrInternalServerError)
}
