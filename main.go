package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kyeongbin-oms/reid-monitoring/util"
)

func main() {
	checkDirectories()
	appCtx, cancelAppCtx := context.WithCancel(context.Background())
	defer cancelAppCtx()

	go util.StartDataCleanup(appCtx, "data")

	// Start monitoring server
	srv := util.WebApp()
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("listen: %v\n", err)
		}
	}()

	log.Printf("Monitoring Server Start Awaiting Signal, Port: %s\n", srv.Addr)

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)

	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")
	cancelAppCtx()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}

	// catching ctx.Done(). timeout of 5 seconds.
	select {
	case <-ctx.Done():
		log.Println("timeout of 5 seconds.")
	}
	log.Println("Server exiting")
}

func checkDirectories() {
	_, err := os.Stat("data")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			mkdirErr := os.Mkdir("data", 0755)
			if mkdirErr != nil {
				panic("not create data directory")
			}
		}
	}
}
