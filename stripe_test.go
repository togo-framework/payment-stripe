package stripe

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/togo-framework/payment"
)

func newTestProvider(h http.HandlerFunc) (*provider, *httptest.Server) {
	srv := httptest.NewServer(h)
	return &provider{key: "sk_test", base: srv.URL, hc: srv.Client()}, srv
}

func TestCreateCharge(t *testing.T) {
	p, srv := newTestProvider(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/charges" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk_test" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
			t.Errorf("content-type = %q", ct)
		}
		_ = r.ParseForm()
		if r.PostForm.Get("amount") != "1500" || r.PostForm.Get("currency") != "usd" || r.PostForm.Get("source") != "tok_visa" {
			t.Errorf("form = %v", r.PostForm)
		}
		fmt.Fprint(w, `{"id":"ch_1","status":"succeeded"}`)
	})
	defer srv.Close()
	ch, err := p.CreateCharge(context.Background(), payment.ChargeRequest{Amount: payment.Money{Amount: 1500, Currency: "USD"}, Token: "tok_visa"})
	if err != nil {
		t.Fatal(err)
	}
	if ch.ID != "ch_1" || ch.Status != "succeeded" {
		t.Errorf("got %+v", ch)
	}
}

func TestRefundAndCustomer(t *testing.T) {
	p, srv := newTestProvider(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/refunds":
			_ = r.ParseForm()
			if r.PostForm.Get("charge") != "ch_1" {
				t.Errorf("refund charge = %q", r.PostForm.Get("charge"))
			}
			fmt.Fprint(w, `{"id":"re_1","status":"succeeded"}`)
		case "/customers":
			fmt.Fprint(w, `{"id":"cus_1"}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	defer srv.Close()
	if err := p.Refund(context.Background(), payment.RefundRequest{ChargeID: "ch_1"}); err != nil {
		t.Fatal(err)
	}
	id, err := p.CreateCustomer(context.Background(), payment.Customer{Email: "a@b.com", Name: "A"})
	if err != nil || id != "cus_1" {
		t.Errorf("customer id=%q err=%v", id, err)
	}
}

func TestCreateCheckoutSession(t *testing.T) {
	p, srv := newTestProvider(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/checkout/sessions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = r.ParseForm()
		if r.PostForm.Get("mode") != "payment" {
			t.Errorf("mode = %q", r.PostForm.Get("mode"))
		}
		if r.PostForm.Get("line_items[0][price_data][unit_amount]") != "2000" {
			t.Errorf("unit_amount = %q", r.PostForm.Get("line_items[0][price_data][unit_amount]"))
		}
		if r.PostForm.Get("line_items[0][price_data][currency]") != "usd" {
			t.Errorf("currency = %q", r.PostForm.Get("line_items[0][price_data][currency]"))
		}
		fmt.Fprint(w, `{"id":"cs_1","url":"https://checkout.stripe/x"}`)
	})
	defer srv.Close()
	cs, err := p.CreateCheckoutSession(context.Background(), payment.CheckoutRequest{Amount: payment.Money{Amount: 2000, Currency: "USD"}, SuccessURL: "https://app/ok", CancelURL: "https://app/no"})
	if err != nil {
		t.Fatal(err)
	}
	if cs.ID != "cs_1" || cs.URL != "https://checkout.stripe/x" {
		t.Errorf("got %+v", cs)
	}
}

func TestErrorStatusReturnsError(t *testing.T) {
	p, srv := newTestProvider(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		fmt.Fprint(w, `{"error":{"message":"card_declined"}}`)
	})
	defer srv.Close()
	if _, err := p.CreateCharge(context.Background(), payment.ChargeRequest{Amount: payment.Money{Amount: 100, Currency: "USD"}, Token: "tok_declined"}); err == nil {
		t.Error("expected an error on HTTP 402")
	}
}

func TestWebhookSignatureVerification(t *testing.T) {
	secret := "whsec_test"
	p := &provider{key: "sk_test", whSecret: secret}
	body := []byte(`{"id":"evt_1","type":"payment_intent.succeeded"}`)
	ts := time.Now().Unix()
	sig := fmt.Sprintf("t=%d,v1=%s", ts, signStripe(secret, ts, body))

	// valid signature → accepted + parsed
	ev, err := p.HandleWebhook(context.Background(), map[string]string{"Stripe-Signature": sig}, body)
	if err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	if ev.ID != "evt_1" || ev.Type != "payment_intent.succeeded" {
		t.Errorf("event parsed wrong: %+v", ev)
	}

	// tampered signature → rejected
	if _, err := p.HandleWebhook(context.Background(), map[string]string{"Stripe-Signature": fmt.Sprintf("t=%d,v1=deadbeef", ts)}, body); err == nil {
		t.Error("tampered signature accepted")
	}

	// valid signature but tampered body → rejected
	if _, err := p.HandleWebhook(context.Background(), map[string]string{"Stripe-Signature": sig}, []byte(`{"id":"evt_x"}`)); err == nil {
		t.Error("tampered body accepted")
	}

	// missing header (secret set) → rejected
	if _, err := p.HandleWebhook(context.Background(), nil, body); err == nil {
		t.Error("missing signature accepted while secret is set")
	}

	// no secret configured → parse-only (dev back-compat)
	p2 := &provider{key: "sk_test"}
	if ev, err := p2.HandleWebhook(context.Background(), nil, body); err != nil || ev.ID != "evt_1" {
		t.Errorf("no-secret path: ev=%+v err=%v", ev, err)
	}
}
