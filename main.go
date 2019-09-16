package main

import (
	"github.com/andrepxx/location-visualizer/controller"
)

/*
 * The entry point of our program.
 */
func main() {
	cn := controller.CreateController()
	cn.Operate()
}
