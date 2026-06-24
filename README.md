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
