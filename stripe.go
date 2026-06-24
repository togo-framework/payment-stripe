// Package stripe is a Stripe driver for togo payment. Blank-import it and set
// PAYMENT_DRIVER=stripe, STRIPE_SECRET_KEY. (For production, verify webhook
// signatures with STRIPE_WEBHOOK_SECRET.)
package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/togo-framework/payment"
	"github.com/togo-framework/togo"
)

const api = "https://api.stripe.com/v1"

func init() {
	payment.RegisterDriver("stripe", func(k *togo.Kernel) (payment.PaymentProvider, error) {
		key := os.Getenv("STRIPE_SECRET_KEY")
		if key == "" {
			return nil, errors.New("payment-stripe: STRIPE_SECRET_KEY not set")
		}
		return &provider{key: key, hc: &http.Client{Timeout: 20 * time.Second}}, nil
	})
}

type provider struct {
	key string
	hc  *http.Client
}

func (p *provider) post(ctx context.Context, path string, form url.Values) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.key)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := p.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if resp.StatusCode >= 300 {
		return m, fmt.Errorf("payment-stripe: %s %d: %s", path, resp.StatusCode, string(b))
	}
	return m, nil
}

func (p *provider) CreateCharge(ctx context.Context, r payment.ChargeRequest) (*payment.Charge, error) {
	form := url.Values{}
	form.Set("amount", strconv.FormatInt(r.Amount.Amount, 10))
	form.Set("currency", strings.ToLower(r.Amount.Currency))
	form.Set("source", r.Token)
	if r.Description != "" {
		form.Set("description", r.Description)
	}
	m, err := p.post(ctx, "/charges", form)
	if err != nil {
		return nil, err
	}
	id, _ := m["id"].(string)
	status, _ := m["status"].(string)
	return &payment.Charge{ID: id, Status: status, Amount: r.Amount, Provider: "stripe", Raw: m}, nil
}

func (p *provider) Refund(ctx context.Context, r payment.RefundRequest) error {
	form := url.Values{}
	form.Set("charge", r.ChargeID)
	if r.Amount != nil {
		form.Set("amount", strconv.FormatInt(r.Amount.Amount, 10))
	}
	_, err := p.post(ctx, "/refunds", form)
	return err
}

func (p *provider) CreateCheckoutSession(ctx context.Context, r payment.CheckoutRequest) (*payment.CheckoutSession, error) {
	items := r.Items
	if len(items) == 0 {
		items = []payment.LineItem{{Name: "Payment", Amount: r.Amount, Quantity: 1}}
	}
	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("success_url", r.SuccessURL)
	form.Set("cancel_url", r.CancelURL)
	for i, it := range items {
		pfx := fmt.Sprintf("line_items[%d]", i)
		form.Set(pfx+"[price_data][currency]", strings.ToLower(it.Amount.Currency))
		form.Set(pfx+"[price_data][unit_amount]", strconv.FormatInt(it.Amount.Amount, 10))
		form.Set(pfx+"[price_data][product_data][name]", it.Name)
		q := it.Quantity
		if q == 0 {
			q = 1
		}
		form.Set(pfx+"[quantity]", strconv.FormatInt(q, 10))
	}
	m, err := p.post(ctx, "/checkout/sessions", form)
	if err != nil {
		return nil, err
	}
	id, _ := m["id"].(string)
	u, _ := m["url"].(string)
	return &payment.CheckoutSession{ID: id, URL: u}, nil
}

func (p *provider) CreateCustomer(ctx context.Context, c payment.Customer) (string, error) {
	form := url.Values{}
	if c.Email != "" {
		form.Set("email", c.Email)
	}
	if c.Name != "" {
		form.Set("name", c.Name)
	}
	m, err := p.post(ctx, "/customers", form)
	if err != nil {
		return "", err
	}
	id, _ := m["id"].(string)
	return id, nil
}

func (p *provider) CreateSubscription(ctx context.Context, r payment.SubscriptionRequest) (*payment.Subscription, error) {
	cust := r.Customer.ID
	if cust == "" {
		var err error
		if cust, err = p.CreateCustomer(ctx, r.Customer); err != nil {
			return nil, err
		}
	}
	form := url.Values{}
	form.Set("customer", cust)
	form.Set("items[0][price]", r.PlanID)
	m, err := p.post(ctx, "/subscriptions", form)
	if err != nil {
		return nil, err
	}
	id, _ := m["id"].(string)
	status, _ := m["status"].(string)
	return &payment.Subscription{ID: id, Status: status, PlanID: r.PlanID, Provider: "stripe"}, nil
}

func (p *provider) HandleWebhook(_ context.Context, _ map[string]string, body []byte) (*payment.WebhookEvent, error) {
	var ev map[string]any
	if err := json.Unmarshal(body, &ev); err != nil {
		return nil, err
	}
	typ, _ := ev["type"].(string)
	id, _ := ev["id"].(string)
	return &payment.WebhookEvent{Type: typ, ID: id, Provider: "stripe", Raw: ev}, nil
}
