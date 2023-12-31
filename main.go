package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"github.com/PeernetOfficial/core"
	"github.com/PeernetOfficial/core/blockchain"
	"github.com/PeernetOfficial/core/btcec"
	"github.com/PeernetOfficial/core/merkle"
	"github.com/PeernetOfficial/core/protocol"
	"github.com/PeernetOfficial/core/warehouse"
	"github.com/PeernetOfficial/core/webapi"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	limiter "github.com/julianshen/gin-limiter"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// Variables for the flags to get the address
var (
	// BackEndApiAddress Refers to the Peernet address ex: <address>:<port no>
	//BackEndApiAddress *string
	// Port Refers to the address for upload webgate server
	Port *string
	// SSL To ensure SSL is required checks if SSL is required and subsequently
	// checks if the certificate is provided
	SSL *bool
	// Certificate SSL Certificate file
	Certificate *string
	// Key SSL Key file
	Key *string
	// BackendAddressWithHTTP ex: http://<address>:<port no>
	//BackendAddressWithHTTP string
	// Production mode
	Production *bool
)

//  -------------------------------------------------------------------------------------------------------------
//  ---------------------------------------- Initialize flags and run Peernet  ---------------------------------
//  -------------------------------------------------------------------------------------------------------------

// init reading flags before any part of the code is executed
func init() {
	//BackEndApiAddress = flag.String("BackEndApiAddress", "localhost:8088", "current environment")
	Port = flag.String("Port", "8098", "current environment")
	SSL = flag.Bool("SSL", false, "Flag to check if the SSL certificate is enabled or not")
	Certificate = flag.String("Certificate", "server.crt", "SSL Certificate file")
	Key = flag.String("Key", "server.key", "SSL Key file")
	Production = flag.Bool("Production", false, "Flag to check if required to run on production mode")
}

// InitPeernet Initializes Peernet backend
func InitPeernet() *core.Backend {
	backend, status, err := core.Init("Peernet Upload Application/1.0", "Config.yaml", nil, nil)
	if status != core.ExitSuccess {
		fmt.Printf("Error %d initializing backend: %s\n", status, err.Error())
		return nil
	}

	return backend
}

// RunPeernet Starts the WebAPI and peernet
func RunPeernet(backend *core.Backend) *webapi.WebapiInstance {
	api := webapi.Start(backend, []string{}, false, "", "", 10*time.Second, 10*time.Second, uuid.Nil)
	backend.Connect()

	return api
}

//  -------------------------------------------------------------------------------------------------------------
//  ---------------------- Custom Structs required parse information from the Peernet backend API ---------------
//  -------------------------------------------------------------------------------------------------------------
//  1. Storing file in the warehouse
//  2. Storing file metadata in the blockchain

// -----------------------------------------------------------------------------------------
// ---------------------------------- Warehouse related structs ---------------------------

type WarehouseResult struct {
	Status int    `json:"status"`
	Hash   []byte `json:"hash"`
}

// -----------------------------------------------------------------------------------------
// -------------------------------- Blockchain related structs -----------------------------

// BlockchainRequest blockchain backend API request struct
type BlockchainRequest struct {
	Files []File `json:"files"`
}

type File struct {
	ID          uuid.UUID         `json:"id"`          // Unique ID.
	Hash        []byte            `json:"hash"`        // Blake3 hash of the file data
	Type        uint8             `json:"type"`        // File Type. For example audio or document. See TypeX.
	Format      uint16            `json:"format"`      // File Format. This is more granular, for example PDF or Word file. See FormatX.
	Size        uint64            `json:"size"`        // Size of the file
	Folder      string            `json:"folder"`      // Folder, optional
	Name        string            `json:"name"`        // Name of the file
	Description string            `json:"description"` // Description. This is expected to be multiline and contain hashtags!
	Date        time.Time         `json:"date"`        // Date shared
	NodeID      []byte            `json:"nodeid"`      // Node ID, owner of the file. Read only.
	Metadata    []apiFileMetadata `json:"metadata"`    // Additional metadata.
}

type apiFileMetadata struct {
	Type uint16 `json:"type"` // See core.TagX constants.
	Name string `json:"name"` // User friendly name of the metadata type. Use the Type fields to identify the metadata as this name may change.
	// Depending on the exact type, one of the below fields is used for proper encoding:
	Text   string    `json:"text"`   // Text value. UTF-8 encoding.
	Blob   []byte    `json:"blob"`   // Binary data
	Date   time.Time `json:"date"`   // Date
	Number uint64    `json:"number"` // Number
}

// BlockchainResponse blockchain backend API response struct
type BlockchainResponse struct {
	Status  int `json:"status"`
	Height  int `json:"height"`
	Version int `json:"version"`
}

// -----------------------------------------------------------------------------------------
// -----------------------------------------------------------------------------------------

//  -------------------------------------------------------------------------------------------------------------
//  -------------------------------------- Functions to call Peernet Apis ---------------------------------------
//  -------------------------------------------------------------------------------------------------------------
//  1. Storing file in the warehouse (/warehouse/create)
//  2. Storing file metadata in the blockchain (/blockchain/file/add)

// -----------------------------------------------------------------------------------------
// ---------------------------------- Warehouse related ------------------------------------

// AddFileWarehouse API call for (Storing file in the warehouse)
func AddFileWarehouse(file io.Reader, backend *core.Backend) (*WarehouseResult, error) {

	var warehouse WarehouseResult
	buf := new(bytes.Buffer)
	buf.ReadFrom(file)

	hash, status, err := backend.UserWarehouse.CreateFile(file, uint64(buf.Len()))
	if err != nil {
		return nil, err
	}
	warehouse.Hash = hash
	warehouse.Status = status

	return &warehouse, nil

	//url := BackendAddressWithHTTP + "/warehouse/create"
	//
	//req, err := http.NewRequest("POST", url, file)
	////req.Header.Set("X-Custom-Header", "myvalue")
	////req.Header.Set("Content-Type", "application/json")
	//
	//client := &http.Client{}
	//resp, err := client.Do(req)
	//if err != nil {
	//    panic(err)
	//}
	//defer resp.Body.Close()
	//
	//body, err := ioutil.ReadAll(resp.Body)
	//if err != nil {
	//    fmt.Println(err)
	//}
	//var result WarehouseResult
	//err = json.Unmarshal(body, &result)
	//if err != nil {
	//    fmt.Println(err)
	//}
	//
	//return &result
}

// UploadFile Simple abstracted function to add files to peernet core
func UploadFile(backend *core.Backend, file *multipart.File, header *multipart.FileHeader) (*btcec.PublicKey, *WarehouseResult, error) {
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, *file); err != nil {
		return nil, nil, errors.New("io.Copy not successful")
	}

	// adds file to warehouse
	warehouseResult, err := AddFileWarehouse(buf, backend)
	if err != nil {
		return nil, nil, err
	}
	// current using default port for Peernet api which is 8080
	// First add file to warehouse

	// Adds the file to a blockchain
	Blockchainfo := AddFileToBlockchain(warehouseResult.Hash, header.Filename, backend)
	if Blockchainfo == nil {
		return nil, nil, errors.New("add file to blockchain not successful")
	}

	_, publicKey := backend.ExportPrivateKey()

	return publicKey, warehouseResult, nil
}

// -----------------------------------------------------------------------------------------
// ----------------------------------- Blockchain related  ---------------------------------

func blockRecordFileFromAPI(input File) (output blockchain.BlockRecordFile) {
	output = blockchain.BlockRecordFile{ID: input.ID, Hash: input.Hash, Type: input.Type, Format: input.Format, Size: input.Size}

	if input.Name != "" {
		output.Tags = append(output.Tags, blockchain.TagFromText(blockchain.TagName, input.Name))
	}
	if input.Folder != "" {
		output.Tags = append(output.Tags, blockchain.TagFromText(blockchain.TagFolder, input.Folder))
	}
	if input.Description != "" {
		output.Tags = append(output.Tags, blockchain.TagFromText(blockchain.TagDescription, input.Description))
	}

	for _, meta := range input.Metadata {
		if blockchain.IsTagVirtual(meta.Type) { // Virtual tags are not mapped back. They are read-only.
			continue
		}

		switch meta.Type {
		case blockchain.TagName, blockchain.TagFolder, blockchain.TagDescription: // auto mapped tags

		case blockchain.TagDateCreated:
			output.Tags = append(output.Tags, blockchain.TagFromDate(meta.Type, meta.Date))

		default:
			output.Tags = append(output.Tags, blockchain.BlockRecordFileTag{Type: meta.Type, Data: meta.Blob})
		}
	}

	return output
}

// setFileMerkleInfo sets the merkle fields in the BlockRecordFile
func setFileMerkleInfo(backend *core.Backend, file *blockchain.BlockRecordFile) (valid bool) {
	if file.Size <= merkle.MinimumFragmentSize {
		// If smaller or equal than the minimum fragment size, the merkle tree is not used.
		file.MerkleRootHash = file.Hash
		file.FragmentSize = merkle.MinimumFragmentSize
	} else {
		// Get the information from the Warehouse .merkle companion file.
		tree, status, _ := backend.UserWarehouse.ReadMerkleTree(file.Hash, true)
		if status != warehouse.StatusOK {
			return false
		}

		file.MerkleRootHash = tree.RootHash
		file.FragmentSize = tree.FragmentSize
	}

	return true
}

// AddFileToBlockchain The following function adds the filename and hash to the blockchain
func AddFileToBlockchain(hash []byte, filename string, backend *core.Backend) *BlockchainResponse {

	// Get file type
	detectType, _, err := webapi.FileDetectType(filename)
	if err != nil {
		panic(err)
	}

	// Create file object for post
	var blockchainRequest BlockchainRequest
	var files File
	files.Name = filename
	files.Hash = hash
	files.Type = uint8(detectType)
	blockchainRequest.Files = append(blockchainRequest.Files, files)

	var filesAdd []blockchain.BlockRecordFile

	for _, file := range blockchainRequest.Files {
		if len(file.Hash) != protocol.HashSize {
			//http.Error(w, "", http.StatusBadRequest)
			//return
		}
		if file.ID == uuid.Nil { // if the ID is not provided by the caller, set it
			file.ID = uuid.New()
		}

		// Verify that the file exists in the warehouse. Folders are exempt from this check as they are only virtual.
		//if !file.IsVirtualFolder() {
		//    if _, err := warehouse.ValidateHash(file.Hash); err != nil {
		//        //http.Error(w, "", http.StatusBadRequest)
		//        //return
		//    } else if _, fileSize, status, _ := backend.UserWarehouse.FileExists(file.Hash); status != warehouse.StatusOK {
		//        //EncodeJSON(api.backend, w, r, apiBlockchainBlockStatus{Status: blockchain.StatusNotInWarehouse})
		//        //return
		//    } else {
		//        file.Size = fileSize
		//    }
		//} else {
		//    file.Hash = protocol.HashData(nil)
		//    file.Size = 0
		//}

		blockRecord := blockRecordFileFromAPI(file)

		// Set the merkle tree info as appropriate.
		setFileMerkleInfo(backend, &blockRecord)

		filesAdd = append(filesAdd, blockRecord)
	}

	newHeight, newVersion, status := backend.UserBlockchain.AddFiles(filesAdd)

	//Byte, err := json.Marshal(blockchainRequest)
	//
	//// convert bytes
	//req, err := http.NewRequest("POST", url, bytes.NewBuffer(Byte))
	////req.Header.Set("X-Custom-Header", "myvalue")
	//req.Header.Set("Content-Type", "application/json")
	//
	//client := &http.Client{}
	//resp, err := client.Do(req)
	//if err != nil {
	//	panic(err)
	//}
	//defer resp.Body.Close()
	//
	//body, _ := ioutil.ReadAll(resp.Body)
	//
	//var result BlockchainResponse
	//err = json.Unmarshal(body, &result)
	//if err != nil {
	//	return nil
	//}

	var result BlockchainResponse
	result.Status = status
	fmt.Println(status)
	result.Version = int(newVersion)
	result.Height = int(newHeight)

	return &result
}

// -----------------------------------------------------------------------------------------
// -----------------------------------------------------------------------------------------

//  -------------------------------------------------------------------------------------------------------------
//  --------------------------------------------- Main function -------------------------------------------------
//  -------------------------------------------------------------------------------------------------------------

func main() {
	// Parsing flags
	flag.Parse()

	// Start peernet
	backend := InitPeernet()
	RunPeernet(backend)

	var r *gin.Engine
	if *Production {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
	} else {
		r = gin.Default()
	}

	// Set of trusted proxies which can be IPV4 or IPV6
	r.SetTrustedProxies([]string{""})

	r.LoadHTMLGlob("templates/*.html")
	r.Static("/templates", "./templates")

	// --------------------------------- Middleware rate limiter -----------------------------------
	lm := limiter.NewRateLimiter(time.Minute, 10, func(ctx *gin.Context) (string, error) {
		return "", nil
	})
	// ---------------------------------------------------------------------------------------------

	// ---------------------------------------- Routes ---------------------------------------------
	// GET /upload to open upload page from webgateway
	r.GET("/upload", lm.Middleware(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "upload.html", nil)
	})

	// POST /uploadFile Uploads file to peernet from Webgateway
	r.POST("/upload", lm.Middleware(), func(c *gin.Context) {
		file, header, err := c.Request.FormFile("file")
		defer file.Close()

		if err != nil {
			c.HTML(http.StatusBadRequest, "upload.html", gin.H{
				"error": "File not added during upload",
			})
			return
		}

		publicKey, warehouseResult, err := UploadFile(backend, &file, header)
		if err != nil {
			c.HTML(http.StatusBadRequest, "upload.html", gin.H{
				"error": "File not added during upload",
			})
			return
			//fmt.Println(err)
		}

		c.HTML(http.StatusOK, "upload.html", gin.H{
			"hash":     hex.EncodeToString(warehouseResult.Hash),
			"filename": header.Filename,
			"size":     header.Size,
			"link":     "https://peer.ae/" + hex.EncodeToString(publicKey.SerializeCompressed()) + "/" + hex.EncodeToString(warehouseResult.Hash),
			"address":  *Port,
		})

	})

	// Implement CURL script to ensure linux users can upload directly
	// the Cli like https://bashupload.com
	// Ex: curl http://localhost:8080/uploadCurl -F add=@<file name>
	r.POST("/uploadCurl", lm.Middleware(), func(c *gin.Context) {
		file, header, err := c.Request.FormFile("add")
		defer file.Close()

		if err != nil {
			fmt.Println(err)
		}

		publicKey, warehouseResult, err := UploadFile(backend, &file, header)
		if err != nil {
			fmt.Println(err)
		}

		link := "https://peer.ae/" + hex.EncodeToString(publicKey.SerializeCompressed()) + "/" + hex.EncodeToString(warehouseResult.Hash)
		c.Data(http.StatusOK, "plain/text", []byte(link))
	})

	// ---------------------------------------------------------------------------------------------

	// ---------------------------------- Start Gin server -----------------------------------------
	// check if SSL is used or not
	if *SSL {
		//BackendAddressWithHTTP = "https://" + *BackEndApiAddress
		r.RunTLS(":"+*Port, *Certificate, *Key)
		//*Port = "https://" + *Port
	} else {
		//BackendAddressWithHTTP = "http://" + *BackEndApiAddress
		//r.Run(*Port)
		r.Run(":" + *Port)
		//*Port = "http://" + *Port
	}
	// ---------------------------------------------------------------------------------------------
}
