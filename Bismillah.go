package main

import "fmt"

var a = 10

//Stardard Function
func add(a, b int) {
	c := a + b
	fmt.Println(c)
}
func main() {

	// Anonumous function
	// Immidiately Invoker(call) Fucntion Expression/IIFE
	func(a, b int) {
		c := a + b
		fmt.Println(c)
	}(5, 7)
}

//Init Function
func init() {
	fmt.Println("I will call First then Main")
}
