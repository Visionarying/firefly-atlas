package main

import (
	"log"

	"firefly-panel/database"
	"firefly-panel/global"
	"firefly-panel/router"
)

func main() {
	global.Init()
	if err := database.Init(); err != nil {
		log.Fatal("database init failed: ", err)
	}
	r := router.Init()
	log.Printf("Firefly-Panel %s listening on :%s (data: %s)", global.Version, global.Port, global.DataDir)
	if err := r.Run(":" + global.Port); err != nil {
		log.Fatal(err)
	}
}
