package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
	"tipjar/internal/models"
	"tipjar/internal/repository"
)

type MpesaService struct {
	txRepo  *repository.TransactionRepository
	jarRepo *repository.JarRepository
}

func NewMpesaService(txRepo *repository.TransactionRepository, jarRepo *repository.JarRepository) *MpesaService {
	return &MpesaService{
		txRepo:  txRepo,
		jarRepo: jarRepo,
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

func (s *MpesaService) getToken() (string, error) {
	consumerKey := os.Getenv("DARAJA_CONSUMER_KEY")
	consumerSecret := os.Getenv("DARAJA_CONSUMER_SECRET")

	if consumerKey == "" || consumerSecret == "" {
		return "", errors.New("Daraja credentials not set")
	}

	credentials := base64.StdEncoding.EncodeToString(
		[]byte(consumerKey + ":" + consumerSecret),
	)

	req, err := http.NewRequest("GET",
		"https://sandbox.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials",
		nil,
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Basic "+credentials)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp tokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return "", err
	}

	if tokenResp.AccessToken == "" {
		return "", errors.New("empty token received from Daraja")
	}

	return tokenResp.AccessToken, nil
}

type stkPushRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            int    `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

type stkPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

func (s *MpesaService) InitiateSTKPush(jarID, phone string, amount int, name, message string) (models.Transaction, error) {
	shortcode := os.Getenv("DARAJA_SHORTCODE")
	passkey := os.Getenv("DARAJA_PASSKEY")
	callbackURL := os.Getenv("DARAJA_CALLBACK_URL")

	timestamp := time.Now().Format("20060102150405")

	rawPassword := shortcode + passkey + timestamp
	password := base64.StdEncoding.EncodeToString([]byte(rawPassword))

	token, err := s.getToken()
	if err != nil {
		return models.Transaction{}, fmt.Errorf("failed to get token: %w", err)
	}

	payload := stkPushRequest{
		BusinessShortCode: shortcode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            amount,
		PartyA:            phone,
		PartyB:            shortcode,
		PhoneNumber:       phone,
		CallBackURL:       callbackURL,
		AccountReference:  "TipJar",
		TransactionDesc:   "TipJar Payment",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return models.Transaction{}, err
	}

	req, err := http.NewRequest("POST",
		"https://sandbox.safaricom.co.ke/mpesa/stkpush/v1/processrequest",
		bytes.NewBuffer(payloadBytes),
	)
	if err != nil {
		return models.Transaction{}, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.Transaction{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Transaction{}, err
	}

	var stkResp stkPushResponse
	err = json.Unmarshal(body, &stkResp)
	if err != nil {
		return models.Transaction{}, err
	}

	if stkResp.ResponseCode != "0" {
		return models.Transaction{}, fmt.Errorf("STK push failed: %s", stkResp.ResponseDescription)
	}

	log.Printf("STK push initiated: %s", stkResp.CheckoutRequestID)

	tx := models.Transaction{
		JarID:             jarID,
		Phone:             phone,
		Name:              name,
		Amount:            amount,
		Message:           message,
		Status:            "pending",
		CheckoutRequestID: stkResp.CheckoutRequestID,
	}

	return s.txRepo.Create(tx)
}

type mpesaCallback struct {
	Body struct {
		StkCallback struct {
			MerchantRequestID string `json:"MerchantRequestID"`
			CheckoutRequestID string `json:"CheckoutRequestID"`
			ResultCode        int    `json:"ResultCode"`
			ResultDesc        string `json:"ResultDesc"`
			CallbackMetadata  struct {
				Item []struct {
					Name  string      `json:"Name"`
					Value interface{} `json:"Value"`
				} `json:"Item"`
			} `json:"CallbackMetadata"`
		} `json:"stkCallback"`
	} `json:"Body"`
}

func (s *MpesaService) HandleCallback(body []byte) error {
	var callback mpesaCallback
	err := json.Unmarshal(body, &callback)
	if err != nil {
		return fmt.Errorf("failed to parse callback: %w", err)
	}

	stk := callback.Body.StkCallback
	checkoutID := stk.CheckoutRequestID
	resultCode := stk.ResultCode

	log.Printf("Callback received: CheckoutID=%s ResultCode=%d", checkoutID, resultCode)

	if resultCode != 0 {
		log.Printf("Payment failed: %s", stk.ResultDesc)
		return s.txRepo.UpdateStatus(checkoutID, "failed", "")
	}

	var receipt string
	var amount float64

	for _, item := range stk.CallbackMetadata.Item {
		switch item.Name {
		case "MpesaReceiptNumber":
			receipt = fmt.Sprintf("%v", item.Value)
		case "Amount":
			amount = item.Value.(float64)
		}
	}

	log.Printf("Payment success: receipt=%s amount=%.0f", receipt, amount)

	err = s.txRepo.UpdateStatus(checkoutID, "complete", receipt)
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	tx, err := s.txRepo.GetByCheckoutID(checkoutID)
	if err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	err = s.jarRepo.UpdateTotal(tx.JarID, int(amount))
	if err != nil {
		return fmt.Errorf("failed to update jar total: %w", err)
	}

	return nil
}

func (s *MpesaService) GetTransactionStatus(checkoutID string) (models.Transaction, error) {
	if checkoutID == "" {
		return models.Transaction{}, errors.New("checkout ID is required")
	}
	return s.txRepo.GetByCheckoutID(checkoutID)
}
