package gemini

import (
	"crypto/hmac"
	"crypto/sha512"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	ApiUrl = "https://api.gemini.com/"
)

// API structure stores Bitfinex API credentials
type API struct {
	APIKey    string
	APISecret string
	client    *http.Client
}

// ErrorMessage ...
type ErrorMessage struct {
	Message string `json:"message"` // Returned only on error
}

// Ticker ...
type Ticker struct {
	Mid       float64 `json:"mid,string"`        // mid (price): (bid + ask) / 2
	Bid       float64 `json:"bid,string"`        // bid (price): Innermost bid.
	Ask       float64 `json:"ask,string"`        // ask (price): Innermost ask.
	LastPrice float64 `json:"last_price,string"` // last_price (price) The price at which the last order executed.
	Low       float64 `json:"low,string"`        // low (price): Lowest trade price of the last 24 hours
	High      float64 `json:"high,string"`       // high (price): Highest trade price of the last 24 hours
	Volume    float64 `json:"volume,string"`     // volume (price): Trading volume of the last 24 hours
	Timestamp float64 `json:"timestamp,string"`  // timestamp (time) The timestamp at which this information was valid.
}

type Orderbook struct {
	Bids []OrderbookOffer // bids (array of bid offers)
	Asks []OrderbookOffer // asks (array of ask offers)
}

type OrderbookOffer struct {
	Price  float64 `json:"price,string"`  // price
	Amount float64 `json:"amount,string"` // amount
}

type OrderStatus struct {
	Symbol            string  `json:"symbol, string"`             // symbol: The symbol name the order belongs to.
	Exchange          string  `json:"exchange, string"`           // exchange: "gemini".
	Price             float64 `json:"price,string"`               // price: The price the order was issued at
	AvgExecutionPrice float64 `json:"avg_execution_price,string"` // avg_execution_price: The average price at which this order as been executed so far. 0 if the order has not been executed at all.
	Side              string  `json:"side, string"`               // side: Either “buy” or “sell”.
	Type              string  `json:"type, string"`               // type: Will always be “exchange limit”
	Timestamp         float64 `json:"timestampms"`                // timestampms: The timestamp the order was submitted in milliseconds.
	Live              bool    `json:"is_live,bool"`               // is_live: true if the order is active on the book (has remaining quantity and has not been canceled)
	Canceled          bool    `json:"is_cancelled,bool"`          // is_cancelled: true if the order has been canceled. Note the spelling, “cancelled” instead of “canceled”. This is for compatibility reasons.
	Forced            bool    `json:"was_forced,bool"`            // was_forced: Will always be false.
	ExecutedAmount    float64 `json:"executed_amount,string"`     // executed_amount: The amount of the order that has been filled.
	RemainingAmount   float64 `json:"remaining_amount,string"`    // remaining_amount: The amount of the order that has not been filled.
	OriginalAmount    float64 `json:"original_amount,string"`     // original_amount: The originally submitted amount of the order.
	OrderID           int     `json:"order_id,string"`            // id: The order ID
}

type Orders []OrderStatus
type Trades []Trade
type Trade struct {
	TID       int     `json:"tid"`
	Timestamp int64   `json:"timestamp"`
	Price     float64 `json:"price,string"`
	Amount    float64 `json:"amount,string"`
	Exchange  string  `json:"exchange"`
	Type      string  `json:"type"`
}

type WalletBalance struct {
	Currency  string  `json:"currency"`         // Currency
	Amount    float64 `json:"amount,string"`    // How much balance of this currency in this wallet
	Available float64 `json:"available,string"` // How much X there is in this wallet that is available to trade.
}
type WalletBalances map[string]WalletBalance

// New returns a new Bitfinex API instance
func New(key, secret, url string) (api *API) {
	var tr *http.Transport
	dialContext := (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 5 * time.Minute,
	}).DialContext
	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		DialContext:     dialContext,
	}
	client := &http.Client{
		Transport: tr,
	}
	api = &API{
		APIKey:    key,
		APISecret: secret,
		client:    client,
	}
	if url != "" {
		ApiUrl = url
	}
	return api
}

func (api *API) Orderbook(symbol string, limitBids, limitAsks int) (orderbook Orderbook, err error) {
	symbol = strings.ToLower(symbol)

	url := "/v1/book/" + symbol + "?"
	if limitBids != -1 {
		url += "limit_bids=" + strconv.Itoa(limitBids)
	}
	if limitAsks != -1 {
		url += "limit_asks=" + strconv.Itoa(limitAsks)
	}

	body, err := api.get(url)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &orderbook)
	if err != nil {
		return
	}

	return
}

// WalletBalances return your balances.
func (api *API) WalletBalances() (wallet WalletBalances, err error) {
	request := map[string]interface{}{
		"request": "/v1/balances",
	}

	body, err := api.post(request)
	if err != nil {
		return
	}

	tmpBalances := []WalletBalance{}
	err = json.Unmarshal(body, &tmpBalances)
	if err != nil { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return nil, errors.New(errorMessage.Message)
	}

	wallet = make(WalletBalances)
	for _, w := range tmpBalances {
		wallet[w.Currency] = w
	}

	return
}

// Trades returns a list of the most recent trades for the given symbol.
// ... Request ...
// 	timestamp (time): Optional. Only show trades at or after this timestamp.
//	limit_trades (int): Optional. Limit the number of trades returned. Must be >= 1. Default is 50.
func (api *API) Trades(symbol string, since int64, limitTrades int, includeBreaks bool) (trades Trades, err error) {
	symbol = strings.ToLower(symbol)

	url := "/v1/trades/" + symbol + "?"

	if since > -1 {
		url += "since=" + strconv.FormatInt(since, 10)
	}
	if limitTrades > -1 {
		url += "limitTrades=" + strconv.Itoa(limitTrades)
	}
	if includeBreaks {
		url += "include_breaks=true"
	}

	body, err := api.get(url)
	if err != nil {
		return nil, errors.New("body: " + string(body) + " err: " + err.Error())
	}

	err = json.Unmarshal(body, &trades)
	if err != nil { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return nil, errors.New(errorMessage.Message)
	}
	return
}

// ActiveOrders returns an array of your active orders.
func (api *API) ActiveOrders() (orders Orders, err error) {
	request := map[string]interface{}{
		"request": "/v1/orders",
	}

	body, err := api.post(request)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &orders)
	if err != nil { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return orders, errors.New(errorMessage.Message)
	}

	return
}

// OrderStatus returns the status of an order given its id.
func (api *API) OrderStatus(id int) (order OrderStatus, err error) {
	request := map[string]interface{}{
		"request":  "/v1/order/status",
		"order_id": id,
	}

	body, err := api.post(request)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &order)
	if err != nil || order.OrderID != id { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return order, errors.New(errorMessage.Message)
	}

	return
}

// CancelOrder cancel an offer give its id.
func (api *API) CancelOrder(id int) (err error) {
	request := map[string]interface{}{
		"request":  "/v1/order/cancel",
		"order_id": id,
	}

	body, err := api.post(request)
	if err != nil {
		return
	}

	tmpOrder := struct {
		ID        int  `json:"order_id,string"`
		Cancelled bool `json:"is_cancelled,bool"`
	}{}

	err = json.Unmarshal(body, &tmpOrder)
	if err != nil || tmpOrder.ID != id { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return errors.New(errorMessage.Message)
	}

	return
}

func (api *API) NewOrder(currency string, amount, price float64, isBuy bool) (order OrderStatus, err error) {
	request := map[string]interface{}{
		"request": "/v1/order/new",
		"symbol":  currency,
		"amount":  strconv.FormatFloat(amount, 'f', -1, 64),
		"price":   strconv.FormatFloat(price, 'f', -1, 64),
		"type":    "exchange limit",
	}

	if isBuy {
		request["side"] = "buy"
	} else {
		request["side"] = "sell"
	}

	body, err := api.post(request)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &order)
	if err != nil || order.OrderID == 0 { // Failed to unmarshal expected message
		fmt.Printf("%+v, %+v, %s\n", err, order, string(body))
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return order, errors.New(errorMessage.Message)
	}

	return
}

///////////////////////////////////////
// API helper methods
///////////////////////////////////////

func (api *API) get(url string) (body []byte, err error) {
	resp, err := http.Get(ApiUrl + url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	return
}

func (api *API) post(payload map[string]interface{}) (body []byte, err error) {
	payload["nonce"] = strconv.FormatInt(time.Now().UnixNano(), 10)
	// X-GEMINI-PAYLOAD
	// parameters-dictionary -> JSON encode -> base64
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return
	}
	payloadBase64 := base64.StdEncoding.EncodeToString(payloadJSON)

	// X-GEMINI-SIGNATURE
	// hex(HMAC_SHA384(base64(payload), key=api_secret))
	h := hmac.New(sha512.New384, []byte(api.APISecret))
	h.Write([]byte(payloadBase64))
	signature := hex.EncodeToString(h.Sum(nil))

	// POST
	req, err := http.NewRequest("POST", ApiUrl+payload["request"].(string), nil)
	if err != nil {
		return
	}

	req.Header.Add("X-GEMINI-APIKEY", api.APIKey)
	req.Header.Add("X-GEMINI-PAYLOAD", payloadBase64)
	req.Header.Add("X-GEMINI-SIGNATURE", signature)

	resp, err := api.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	return
}
