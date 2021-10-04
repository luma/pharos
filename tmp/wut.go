package main

import "github.com/tidwall/sjson"

// const json = `{"name":{"first":"Janet","last":"Prichard"},"age":47}`
var json = []byte("")

func main() {
	value, _ := sjson.SetBytes(json, "name", "Anderson")
	println(string(value))
}
