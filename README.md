<!-- togo-header -->
<div align="center">
  <img src=".github/assets/togo-mark.svg" alt="togo" height="64" />
  <h1>togo-framework/payment-stripe</h1>
  <p>
    <a href="https://to-go.dev/marketplace"><img src="https://img.shields.io/badge/marketplace-to--go.dev-1FC7DC" alt="marketplace" /></a>
    <a href="https://pkg.go.dev/github.com/togo-framework/payment-stripe"><img src="https://pkg.go.dev/badge/github.com/togo-framework/payment-stripe.svg" alt="pkg.go.dev" /></a>
    <img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT" />
  </p>
  <p><strong>Part of the <a href="https://to-go.dev">togo</a> framework.</strong></p>
</div>

## Install

```bash
togo install togo-framework/payment-stripe
```

<!-- /togo-header -->

# payment-stripe

[Stripe](https://stripe.com) driver for togo **payment**. Blank-import + select it:

```bash
togo install togo-framework/payment        # base
togo install togo-framework/payment-stripe  # this driver
```
```env
PAYMENT_DRIVER=stripe
STRIPE_SECRET_KEY=sk_live_...
# optional: STRIPE_WEBHOOK_SECRET for production webhook verification
```

Implements charges, refunds, hosted Checkout, customers, subscriptions, and
webhooks on the togo `payment.PaymentProvider` interface via the Stripe REST API.

MIT

<!-- togo-sponsors -->
---

<div align="center">
  <h3>Premium sponsors</h3>
  <p>
    <a href="https://id8media.com"><strong>ID8 Media</strong></a> &nbsp;·&nbsp;
    <a href="https://one-studio.co"><strong>One Studio</strong></a>
  </p>
  <p><sub>Support togo — <a href="https://github.com/sponsors/fadymondy">become a sponsor</a>.</sub></p>
</div>
<!-- /togo-sponsors -->
