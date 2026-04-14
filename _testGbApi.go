package main

import (
	"fmt"
	"xxmimm/internal/gamebanana"
)

// test with go run _testGbApi.go

func main() {
	url := "https://gamebanana.com/mods/650540"
	
	data, errStr := gamebanana.FetchModInfo(url)
	
	if errStr != "" {
		fmt.Println("Error:", errStr)
		return
	}

	mod := data.(gamebanana.ModData)
	
	fmt.Printf("Mod Name: %s\n", mod.Name)
	fmt.Printf("Description: %s\n", mod.Description)
	fmt.Printf("Image: %s\n", mod.ImageURL)
	fmt.Printf("1st file id: %d\n", mod.Files[1].ID)
	fmt.Printf("1st file name: %s\n", mod.Files[1].Name)
	fmt.Printf("1st file desc: %s\n", mod.Files[1].Description)
	fmt.Printf("1st file url: %s\n", mod.Files[1].DirectURL)
	fmt.Printf("1st file size: %d\n", mod.Files[1].Size)
	fmt.Printf("1st file hash: %s\n", mod.Files[1].MD5)
	fmt.Printf("everything files: %s\n", mod.Files)
	
}