package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/PeernetOfficial/core"
	"github.com/PeernetOfficial/core/webapi"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// StartPeernet start peernet
func StartPeernet() {
	backend, status, err := core.Init("Your application/1.0", "Config.yaml", nil, nil)
	if status != core.ExitSuccess {
		fmt.Printf("Error %d initializing backend: %s\n", status, err.Error())
		return
	}

	webapi.Start(backend, []string{"0.0.0.0:8081"}, false, "", "", 10*time.Second, 10*time.Second, uuid.Nil)
	backend.Connect()

	for {

	}

}

type WarehouseResult struct {
	Status int    `json:"status"`
	Hash   string `json:"hash"`
}

type BlockchainRequest struct {
	Files []File `json:"files"`
}

type File struct {
	Hash string `json:"hash"`
	Type int    `json:"type"`
	Name string `json:"name"`
}

func AddFileWarehouse(file io.Reader) *WarehouseResult {
	url := "http://0.0.0.0:8081/warehouse/create"

	req, err := http.NewRequest("POST", url, file)
	//req.Header.Set("X-Custom-Header", "myvalue")
	//req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
	var result WarehouseResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println(err)
	}

	return &result
}

type BlockchainResponse struct {
	Status  int `json:"status"`
	Height  int `json:"height"`
	Version int `json:"version"`
}

func AddFileToBlockchain(hash string, filename string) *BlockchainResponse {
	url := "http://0.0.0.0:8081/blockchain/file/add"

	// Create file object for post
	var blockchainRequest BlockchainRequest
	var files File
	files.Name = filename
	files.Hash = hash
	files.Type = 0
	blockchainRequest.Files = append(blockchainRequest.Files, files)

	Byte, err := json.Marshal(blockchainRequest)

	// convert bytes
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(Byte))
	//req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
	var result BlockchainResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil
	}

	return &result

	//return &result
}

// Add files
func main() {
	// Start peernet
	go StartPeernet()
	r := gin.Default()
	r.LoadHTMLGlob("templates/*.html")
	r.GET("/upload", func(c *gin.Context) {
		fmt.Println("here")
		c.HTML(http.StatusOK, "upload.html", nil)
	})

	r.POST("/uploadFile", func(c *gin.Context) {
		fmt.Println("here")
		file, header, err := c.Request.FormFile("file")
		defer file.Close()

		if err != nil {
			fmt.Println(err)
		}

		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, file); err != nil {
			fmt.Println(err)
		}

		// adds file to warehouse
		warehouseResult := AddFileWarehouse(buf)
		fmt.Println(warehouseResult.Hash)
		// current using default port for Peernet api which is 8080
		// First add file to warehouse

		// Adds the file to a blockchain
		Blockchainfo := AddFileToBlockchain(warehouseResult.Hash, header.Filename)
		if Blockchainfo == nil {
			return
		}

		fmt.Println("here")
		c.HTML(http.StatusOK, "upload.html", gin.H{
			"hash": warehouseResult.Hash,
		})

	})

	r.Run()
}
