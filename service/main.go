package main

import (
	"log"

	"firefly-atlas/database"
	"firefly-atlas/global"
	"firefly-atlas/router"
)

func main() {
	global.Init()
	if err := database.Init(); err != nil {
		log.Fatal("database init failed: ", err)
	}
	r := router.Init()
	log.Printf("Firefly Atlas %s listening on :%s (data: %s)", global.Version, global.Port, global.DataDir)
	if err := r.Run(":" + global.Port); err != nil {
		log.Fatal(err)
	}
}
