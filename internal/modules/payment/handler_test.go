package payment

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func setupTestRouter() (*gin.Engine, *mockRepository) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	repo := newMockRepository()
	svc := newTestService(repo, zap.NewNop())
	h := NewHandler(svc)

	v1 := r.Group("/api/v1")
	RegisterRoutes(v1, h)

	return r, repo
}

func TestHandler_CreatePayment_Success(t *testing.T) {
	router, repo := setupTestRouter()

	// Seed customer
	_ = repo.CreateCustomer(t.Context(), &Customer{
		ID:    "cust_demo_1",
		Email: "demo@test.com",
	})

	payload := CreatePaymentRequest{
		IdempotencyKey: "ik_api_test_1",
		CustomerID:     "cust_demo_1",
		Amount:         75.50,
		Currency:       "USD",
		Provider:       ProviderStripe,
		Description:    "API Test Payment",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var res PaymentResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.ID == "" {
		t.Errorf("expected non-empty payment ID")
	}
	if res.Amount != 75.50 {
		t.Errorf("expected amount 75.50, got %v", res.Amount)
	}
}

func TestHandler_CreatePayment_CustomerNotFound(t *testing.T) {
	router, _ := setupTestRouter()

	payload := CreatePaymentRequest{
		IdempotencyKey: "ik_missing_cust",
		CustomerID:     "cust_does_not_exist",
		Amount:         50.00,
		Currency:       "USD",
		Provider:       ProviderStripe,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestHandler_GetPaymentByID(t *testing.T) {
	router, repo := setupTestRouter()

	_ = repo.CreateCustomer(t.Context(), &Customer{
		ID:    "cust_demo_2",
		Email: "demo2@test.com",
	})

	// Create a payment first
	svc := newTestService(repo, zap.NewNop())
	payRes, _ := svc.CreatePayment(t.Context(), CreatePaymentRequest{
		IdempotencyKey: "ik_get_api",
		CustomerID:     "cust_demo_2",
		Amount:         100.00,
		Currency:       "USD",
		Provider:       ProviderPayPal,
	})

	// 1. Valid Get
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/payments/"+payRes.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var detail PaymentDetailResponse
	_ = json.Unmarshal(w.Body.Bytes(), &detail)
	if detail.ID != payRes.ID {
		t.Errorf("expected payment ID %s, got %s", payRes.ID, detail.ID)
	}

	// 2. Not found Get
	reqNotFound, _ := http.NewRequest(http.MethodGet, "/api/v1/payments/non_existent_pay_id", nil)
	wNotFound := httptest.NewRecorder()
	router.ServeHTTP(wNotFound, reqNotFound)

	if wNotFound.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for missing payment, got %d", wNotFound.Code)
	}
}

func TestHandler_ListPayments(t *testing.T) {
	router, repo := setupTestRouter()

	_ = repo.CreateCustomer(t.Context(), &Customer{ID: "cust_list", Email: "l@test.com"})
	svc := newTestService(repo, zap.NewNop())
	_, _ = svc.CreatePayment(t.Context(), CreatePaymentRequest{
		IdempotencyKey: "ik_l1",
		CustomerID:     "cust_list",
		Amount:         30.00,
		Currency:       "USD",
		Provider:       ProviderBKash,
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/payments?customer_id=cust_list&page=1&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res PaymentListResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Total < 1 {
		t.Errorf("expected at least 1 payment, got %d", res.Total)
	}
}

func TestHandler_Refunds(t *testing.T) {
	router, repo := setupTestRouter()

	_ = repo.CreateCustomer(t.Context(), &Customer{ID: "cust_ref_h", Email: "refh@test.com"})
	svc := newTestService(repo, zap.NewNop())
	payRes, _ := svc.CreatePayment(t.Context(), CreatePaymentRequest{
		IdempotencyKey: "ik_ref_h",
		CustomerID:     "cust_ref_h",
		Amount:         150.00,
		Currency:       "USD",
		Provider:       ProviderStripe,
	})

	// 1. Create refund via API
	refundPayload := CreateRefundRequest{
		PaymentID: payRes.ID,
		Amount:    50.00,
		Reason:    "Customer returned damaged goods",
	}
	body, _ := json.Marshal(refundPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payments/refunds", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for refund, got %d: %s", w.Code, w.Body.String())
	}

	var refRes RefundResponse
	_ = json.Unmarshal(w.Body.Bytes(), &refRes)
	if refRes.ID == "" {
		t.Errorf("expected non-empty refund ID")
	}

	// 2. Get refund by ID
	reqGetRef, _ := http.NewRequest(http.MethodGet, "/api/v1/payments/refunds/"+refRes.ID, nil)
	wGetRef := httptest.NewRecorder()
	router.ServeHTTP(wGetRef, reqGetRef)

	if wGetRef.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for get refund, got %d", wGetRef.Code)
	}

	// 3. List refunds for payment
	reqListRef, _ := http.NewRequest(http.MethodGet, "/api/v1/payments/"+payRes.ID+"/refunds", nil)
	wListRef := httptest.NewRecorder()
	router.ServeHTTP(wListRef, reqListRef)

	if wListRef.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for list payment refunds, got %d", wListRef.Code)
	}
}

func TestHandler_CustomerEndpoints(t *testing.T) {
	router, _ := setupTestRouter()

	// 1. POST /api/v1/payments/customers
	custPayload := CreateCustomerRequest{
		Email: "sarah.connor@cyberdyne.io",
		Name:  "Sarah Connor",
		Phone: "+18005550199",
	}
	body, _ := json.Marshal(custPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payments/customers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for customer, got %d: %s", w.Code, w.Body.String())
	}

	var custRes CustomerResponse
	_ = json.Unmarshal(w.Body.Bytes(), &custRes)
	if custRes.ID == "" {
		t.Errorf("expected customer ID")
	}

	// 2. GET /api/v1/payments/customers/:id
	reqGet, _ := http.NewRequest(http.MethodGet, "/api/v1/payments/customers/"+custRes.ID, nil)
	wGet := httptest.NewRecorder()
	router.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for get customer, got %d", wGet.Code)
	}

	// 3. GET /api/v1/payments/customers
	reqList, _ := http.NewRequest(http.MethodGet, "/api/v1/payments/customers", nil)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for list customers, got %d", wList.Code)
	}
}
