package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/joho/godotenv"
)

var bankEndpoints map[string]string
var bankTokens map[string]string

type TransferRequest struct {
	SenderBankID          string  `json:"senderBankId"`
	ReceiverBankID        string  `json:"receiverBankId"`
	SenderAccountNumber   int     `json:"senderAccountNumber"`
	ReceiverAccountNumber int     `json:"receiverAccountNumber"`
	Amount                float64 `json:"amount"`
	Message               string  `json:"message"`
}

type TransferAnswer struct {
	Message string `json:"message"`
	Status  bool   `json:"status"`
}

func transferRequest(c *gin.Context) {
	token := c.Request.Header["Token"]

	if len(token) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No bank token provided."})
		return
	}

	request := TransferRequest{}

	c.Bind(&request)

	if token[0] != bankTokens[request.SenderBankID] {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Not authorized. Provide valid admin key."})
		return
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	postBody, _ := json.Marshal(map[string]string{
		"senderBankId":          request.SenderBankID,
		"senderAccountNumber":   strconv.Itoa(request.SenderAccountNumber),
		"receiverAccountNumber": strconv.Itoa(request.ReceiverAccountNumber),
		"amount":                fmt.Sprintf("%f", request.Amount),
		"message":               request.Message,
	})

	requestBody := bytes.NewBuffer(postBody)

	outRequest, err := http.NewRequest("POST", bankEndpoints[request.ReceiverBankID], requestBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Making request to receiving bank failed.", "status": false})
		return
	}

	outRequest.Header.Set("Token", bankTokens[request.ReceiverBankID])

	response, err := client.Do(outRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Request to receiving bank failed.", "status": false})
		return
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Reading request to receiving bank failed.", "status": false})
		return
	}

	answer := TransferAnswer{}

	err = json.Unmarshal(body, &answer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Parsing request to receiving bank failed.", "status": false})
		return
	}

	if answer.Status {
		c.JSON(http.StatusOK, gin.H{"message": answer.Message, "status": answer.Status})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"message": answer.Message, "status": answer.Status})
}

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	err = json.Unmarshal([]byte(os.Getenv("BANK_ENDPOINTS")), &bankEndpoints)
	if err != nil {
		log.Fatal("Could not parse bank endpoints")
	}

	err = json.Unmarshal([]byte(os.Getenv("BANK_TOKENS")), &bankTokens)
	if err != nil {
		log.Fatal("Could not parse bank tokens")
	}
}

func main() {
	router := gin.Default()

	router.POST("/transfer", transferRequest)

	router.Run(":" + os.Getenv("PORT"))
}
