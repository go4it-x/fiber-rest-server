package server

import (
	"home/pkg/conf"
	"home/pkg/e"
	"home/pkg/resp"
	"home/pkg/utils/jwt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	jwtWare "github.com/gofiber/jwt/v3"
)

type Config struct {
	Addr       string
	EnableCors bool
	EnableCsrf bool
	Logger     bool
	BodyLimit  int
}

type Static struct {
	Prefix string
	Root   string
	Config struct {
		Switch bool
		Static fiber.Static
	}
}

// Init server
func Init(prefix string, static []Static, noAuth func(*fiber.App), auth func(fiber.Router)) (err error) {
	var config Config
	if err = conf.Load("server.http", &config); err != nil {
		return
	}

	if config.BodyLimit == 0 {
		config.BodyLimit = 4 * 1024 * 1024 // 4MB
	}

	app := fiber.New(fiber.Config{
		BodyLimit:    config.BodyLimit,
		ErrorHandler: resp.ErrorHandler,
	})

	app.Use(recover.New())
	if config.Logger {
		app.Use(logger.New())
	}

	if config.EnableCsrf {
		app.Use(csrf.New())
	}

	if config.EnableCors {
		app.Use(cors.New(cors.Config{
			AllowCredentials: true,
		}))
	}

	for _, v := range static {
		if v.Config.Switch {
			app.Static(v.Prefix, v.Root, v.Config.Static)
		} else {
			app.Static(v.Prefix, v.Root)
		}
	}

	// custom return results
	app.Get("/", func(ctx *fiber.Ctx) error {
		return resp.New(ctx).JSON("hello world")
	})

	if noAuth != nil {
		noAuth(app)
	}

	if auth != nil {
		router := app.Group(prefix, jwtWare.New(jwtWare.Config{
			SigningKey: []byte(jwt.Signing),
			ErrorHandler: func(ctx *fiber.Ctx, err error) error {
				if err != nil {
					err = e.NewError(fiber.StatusUnauthorized, err.Error())
				}

				return err
			},
		}))

		auth(router)
	}

	// Handle not founds
	app.Use(func(ctx *fiber.Ctx) error {
		return fiber.ErrNotFound
	})

	err = app.Listen(config.Addr)
	return
}
