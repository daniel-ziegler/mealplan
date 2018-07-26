package main

import (
	"fmt"

	. "github.com/pikans/mealplan"
)

var dataFile = "signups.dat"

func main() {
	data, err := ReadData(dataFile)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", data)
	for _, ass := range data.Assignments {
		for ix := range ass {
			if ass[ix] == "dganelin" {
				ass[ix] = ""
			}
		}
	}
	err = WriteData(dataFile, data)
	if err != nil {
		panic(err)
	}
}
