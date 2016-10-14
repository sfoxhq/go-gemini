package gemini

import (
	"os"
	"strconv"
	"testing"
	"fmt"
)

var APIKey = os.Getenv("GEMINI_API_KEY")
var APISecret = os.Getenv("GEMINI_API_SECRET")

var url = os.Getenv("GEMINI_URL")
var apiPublic = New("", "", url)
var apiPrivate = New(APIKey, APISecret, url)

func checkEnv(t *testing.T) {
	if APIKey == "" || APISecret == "" {
		t.Skip("Skipping test because because APIKey and/or APISecret env variables are not set")
	}
}

func TestOrderbook(t *testing.T) {
	// Test normal request
	orderbook, err := apiPublic.Orderbook("btcusd", -1, -1)
	if err != nil || len(orderbook.Asks) != 50 || len(orderbook.Bids) != 50 {
		t.Error("Failed")
		return
	}
}

func TestTrades(t *testing.T) {
	// Test normal request
	trades, err := apiPublic.Trades("btcusd", 0, -1, false)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	if len(trades) != 50 {
		t.Error(fmt.Errorf("returned %d != 50", len(trades)))
		return
	}
}

func TestNewOrder(t *testing.T) {
	checkEnv(t)

	order, err := apiPrivate.NewOrder("btcusd", 1, 1, true)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}

	if order.OrderID == 0 {
		t.Error("failed to create order")
		return
	}
}

func TestActiveOrders(t *testing.T) {
	checkEnv(t)

	orders, err := apiPrivate.ActiveOrders()
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}

	if len(orders) == 0 {
		t.Log("No active offers detected, please inspect")
		return
	}

	t.Log("Detected active orders, please inspect:")
	for _, o := range orders {
		t.Log("\tid:" + strconv.Itoa(o.OrderID) + " " + o.Symbol + ":" + o.Side + ":" + o.Type + " : " + strconv.FormatFloat(o.RemainingAmount, 'f', -1, 64) + " at " + strconv.FormatFloat(o.Price, 'f', -1, 64))
	}
}

func TestOrderStatus(t *testing.T) {
	checkEnv(t)

	// Assuming TestActiveOrders has PASSED
	orders, err := apiPrivate.ActiveOrders()
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}

	if len(orders) == 0 {
		t.Log("No active orders, nothing to get the status of, please inspect")
		return
	}

	t.Log("Order status # " + strconv.Itoa(orders[0].OrderID))
	o, err := apiPrivate.OrderStatus(orders[0].OrderID)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	t.Log("\tid:" + strconv.Itoa(o.OrderID) + " " + o.Symbol + ":" + o.Side + ":" + o.Type + " : " + strconv.FormatFloat(o.RemainingAmount, 'f', -1, 64) + " at " + strconv.FormatFloat(o.Price, 'f', -1, 64))

	return
}

func TestCancelOrder(t *testing.T) {
	checkEnv(t)

	// Assuming TestActiveOrders has PASSED
	orders, err := apiPrivate.ActiveOrders()
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}

	t.Logf("Active orders = %d", len(orders))
	if len(orders) == 0 {
		t.Log("No active orders, nothing to cancel, please inspect")
		return
	}

	for _,order := range orders {
		t.Log("Cancelling order # " + strconv.Itoa(order.OrderID))
		err = apiPrivate.CancelOrder(order.OrderID)
		err = apiPrivate.CancelOrder(order.OrderID)
		if err != nil {
			t.Error("Failed: " + err.Error())
			return
		}
	}
}

func TestCancelUnknownOrder(t *testing.T) {
	checkEnv(t)

	t.Log("Cancelling order #666")
	err = apiPrivate.CancelOrder(666)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
	if len(orders) == 0 {
		t.Log("No active orders, nothing to cancel, please inspect")
		return
	}
}

func TestWalletBalances(t *testing.T) {
	checkEnv(t)

	balances, err := apiPrivate.WalletBalances()
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}

	if len(balances) == 0 {
		t.Log("No wallet balances detected, please inspect")
		return
	}

	t.Log("Detected wallet balances, please inspect:")
	for k, v := range balances {
		t.Log("\t" + k + ": " +
			" (available: " + strconv.FormatFloat(v.Available, 'f', -1, 64) + ") " + k)

	}
}

