# Web Gateway upload 
The following repository consists of the Peernet Web gateway upload implementation.

## Features: 
- Upload file from Webpage
- Upload file to Peernet from a single Curl request 

## Build
```go
go build . 
```

## Run 
Run on default parameters (With Debug output)
```
./WebGatewayUpload
```
Run on Production mode
```
./WebGatewayUpload -Production
```
Custom Flags 
```
./WebGatewayUpload -h 

Usage of ./WebGatewayUpload:
  -BackEndApiAddress string
    	current environment (default "localhost:8088")
  -Certificate string
    	SSL Certificate file (default "server.crt")
  -Key string
    	SSL Key file (default "server.key")
  -Production
    	Flag to check if required to run on production mode
  -SSL
    	Flag to check if the SSL certificate is enabled or not
  -WebpageAddress string
    	current environment (default "localhost:8098")
```

## Routes 
- (GET) `/upload` (Opens upload page in the webgateway)
- (POST) `/upload` (Uploads file to peernet from Webpage)
- (POST) `/uploadCurl` (Uploads file from CURL)

   Ex:
   ```
   curl http://localhost:8088/uploadCurl -F add=@test.txt
   ```


