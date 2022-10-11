package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"
)

var (
	client = http.Client{}
)

type HashLookup struct {
	Zone            string `json:"Zone"`
	FileGeneralInfo struct {
		FileStatus string    `json:"FileStatus"`
		Sha1       string    `json:"Sha1"`
		Md5        string    `json:"Md5"`
		Sha256     string    `json:"Sha256"`
		FirstSeen  time.Time `json:"FirstSeen"`
		LastSeen   time.Time `json:"LastSeen"`
		Signer     string    `json:"Signer"`
		Size       int       `json:"Size"`
		Type       string    `json:"Type"`
		HitsCount  int       `json:"HitsCount"`
	} `json:"FileGeneralInfo"`
}

// These are the possible output strings of this function (Malware, Adware and other, Clean, No threats detected, or Not categorized)
// Example calls
//getHashStatus("7a2278a9a74f49852a5d75c745ae56b80d5b4c16f3f6a7fdfd48cb4e2431c688") // Bad
//getHashStatus("408f31d86c6bf4a8aff4ea682ad002278f8cb39dc5f37b53d343e63a61f3cc4f") // Uncategorized
//getHashStatus("27dfb7631807c7bd185f57cd6de0628c6e9c47ed9b390a9b8544fdf12a323e04") // No results
//getHashStatus("98E07EDE313BAB4D2B659F4AF09804DB554287308EC1882D3D4036BEAE0D126E") // Clean

func getHashStatus(hash string, hashType string) string {
	var isValidHash bool

	switch {
	case hashType == "md5":
		isValidHash = verifyMD5Hash(hash)
	case hashType == "sha1":
		isValidHash = verifySHA1Hash(hash)
	case hashType == "sha256":
		isValidHash = verifySHA256Hash(hash)
	default:
		log.Fatalln("Supported hash types are: md5, sha1, sha256")
	}

	if !isValidHash {
		log.Fatalln("Hash verification failed!")
	}

	url := "https://opentip.kaspersky.com/api/v1/search/hash?request=" + hash

	request, _ := http.NewRequest("GET", url, nil)
	request.Header.Add("x-api-key", getKaperskyKey())
	response, err := client.Do(request)
	if err != nil {
		log.Fatalln(err)
	}

	switch response.Status != "200 OK" {
	case "400 Bad Request" == response.Status:
		log.Print("No results")
	case "401 Unauthorized" == response.Status:
		log.Fatalln("Your API key is not working!")
	case "403 Forbidden" == response.Status:
		log.Fatalln("You may have been blocked or your quote with kapersky reached")
	case "404 Not Found" == response.Status:
		log.Fatalln("Resource not found, check your endpoint address")
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var result HashLookup

	sb := string(body)
	log.Printf(sb)

	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}

	return result.FileGeneralInfo.FileStatus
}

func verifySHA256Hash(hash string) bool {
	match, _ := regexp.MatchString("[A-Fa-f0-9]{64}", hash)
	return match
}

func verifySHA1Hash(hash string) bool {
	match, _ := regexp.MatchString("[a-fA-F0-9]{40}$", hash)
	return match
}

func verifyMD5Hash(hash string) bool {
	match, _ := regexp.MatchString("/^[a-f0-9]{32}$/i", hash)
	return match
}
