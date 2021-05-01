package main

import (
	"flag"
	"github.com/gorilla/handlers"
	_ "github.com/joho/godotenv/autoload"
	"gitlab.com/pbobby001/postit-scheduler/app/middlewares"
	"gitlab.com/pbobby001/postit-scheduler/app/router"
	"gitlab.com/pbobby001/postit-scheduler/db"
	"gitlab.com/pbobby001/postit-scheduler/pkg/logs"
	"golang.org/x/net/context"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {

	defer logs.Logger.Flush()
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	// Is this better?
	db.Connect()

	r := router.InitRoutes()

	origins := handlers.AllowedOrigins([]string{ /*"https://postit-ui.herokuapp.com"*/ "*"})
	headers := handlers.AllowedHeaders([]string{
		"Content-Type",
		"Content-Length",
		"Content-Event-Type",
		"X-Requested-With",
		"Accept-Encoding",
		"Accept",
		"Authorization",
		"Access-Control-Allow-Origin",
		"User-Agent",
		"tenant-namespace",
		"trace-id",
	})
	methods := handlers.AllowedMethods([]string{
		http.MethodPost,
		http.MethodGet,
		http.MethodPut,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodPut,
	})

	var port string
	port = os.Getenv("PORT")
	if port == "" {
		_ = logs.Logger.Warn("Defaulting to port 7894")
		port = "7894"
	}

	address := ":" + port

	server := &http.Server{
		Addr: address,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Hour * 12,
		Handler:      handlers.CORS(origins, headers, methods)(r), // Pass our instance of gorilla/mux in.
	}

	r.Use(middlewares.JSONMiddleware)
	//r.Use(middlewares.JWTMiddleware)

	go func() {
		for {
			ticker := time.NewTicker(30 * time.Second)
			for t := range ticker.C {
				resp, err := http.Get(os.Getenv("PING_URL"))
				if err != nil {
					_ = logs.Logger.Error("unable to ping")
					continue
				}
				body, _ := ioutil.ReadAll(resp.Body)
				logs.Logger.Info(string(body))
				logs.Logger.Info("Pinging " + os.Getenv("PING_URL")+ t.String())
			}
		}
	}()

	defer db.Disconnect()
	// Run our server in a goroutine so that it doesn't block.
	go func() {
		logs.Logger.Info("Server running on port", address)
		if err := server.ListenAndServe(); err != nil {
			a := logs.Logger.Warn(err)
			if a != nil {
				_ = logs.Logger.Error(a)
			}
		}
	}()

	channel := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.

	signal.Notify(channel, os.Interrupt)
	// Block until we receive our signal.
	<-channel

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	err := server.Shutdown(ctx)
	if err != nil {
		_ = logs.Logger.Error(err)
		os.Exit(0)
	}

	// Optionally, you could run server.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	_ = logs.Logger.Warn("shutting down")
	os.Exit(0)
}
