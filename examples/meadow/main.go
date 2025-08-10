//go run main.go one.go two.go three.go

package main

import (
    "fmt"
    )

func main(){
    fmt.Println("From function main()")
    One()
    Two()
    Three()
}
