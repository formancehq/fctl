package demo

func marketplacePayoutsSteps(c cycleContext) []recipeStep {
	gross := c.amount
	commission := int64(500)
	fee := int64(200)
	net := gross - commission - fee
	psp, bank := c.account("psp"), c.account("bank")
	seller := c.account("seller_payout")
	order := c.reference("payout_order")
	returnedPayout := c.reference("returned_payout")
	paidPayout := c.reference("paid_payout")
	fxPayout := c.reference("fx_payout")
	payoutAmount := net / 4
	fxSell := net / 5
	fxGross := fxSell * 9 / 10
	fxSpread := int64(10)
	usd := func(amount int64) string { return monetary("USD/2", amount) }
	eur := func(amount int64) string { return monetary("EUR/2", amount) }
	steps := []recipeStep{
		{template: "ORDER_PAYMENT_CAPTURED", vars: map[string]string{"psp_id": psp, "order_id": order, "seller_id": seller, "gross": usd(gross), "commission": usd(commission), "psp_fee": usd(fee), "order_ref": order}},
		{template: "PSP_SETTLEMENT", vars: map[string]string{"psp_id": psp, "bank_id": bank, "gross_settled": usd(gross), "fees_retained": usd(fee), "settlement_ref": c.reference("payout_order_settlement")}},
		{template: "SELLER_FUNDS_RELEASE", vars: map[string]string{"order_id": order, "seller_id": seller, "order_ref": order}},
		{template: "SELLER_PAYOUT_INITIATE", vars: map[string]string{"seller_id": seller, "amount": usd(payoutAmount), "payout_ref": returnedPayout}},
		{template: "SELLER_PAYOUT_RETURN", vars: map[string]string{"seller_id": seller, "amount": usd(payoutAmount), "payout_ref": returnedPayout}},
		{template: "SELLER_PAYOUT_INITIATE", vars: map[string]string{"seller_id": seller, "amount": usd(payoutAmount), "payout_ref": paidPayout}},
		{template: "SELLER_PAYOUT_SETTLE", vars: map[string]string{"seller_id": seller, "bank_id": bank, "amount": usd(payoutAmount), "payout_ref": paidPayout}},
		{template: "MULTI_CURRENCY_PAYOUT", vars: map[string]string{"seller_id": seller, "conversion_id": c.reference("conversion"), "sell_amount": usd(fxSell), "buy_gross": eur(fxGross), "spread": eur(fxSpread), "execution_rate": "0.90", "payout_ref": fxPayout}},
		{template: "SELLER_PAYOUT_SETTLE", vars: map[string]string{"seller_id": seller, "bank_id": bank, "amount": eur(fxGross - fxSpread), "payout_ref": fxPayout}},
	}
	// Separate sellers keep the refund and chargeback branches independent of
	// the successful payout path, including the multi-currency reservation.
	refundSeller, refundOrder := c.account("seller_refund"), c.reference("refund_order")
	refundCapture := len(steps)
	steps = append(steps,
		recipeStep{template: "ORDER_PAYMENT_CAPTURED", vars: map[string]string{"psp_id": psp, "order_id": refundOrder, "seller_id": refundSeller, "gross": usd(gross), "commission": usd(commission), "psp_fee": usd(fee), "order_ref": refundOrder}},
		recipeStep{template: "PSP_SETTLEMENT", vars: map[string]string{"psp_id": psp, "bank_id": bank, "gross_settled": usd(gross), "fees_retained": usd(fee), "settlement_ref": c.reference("refund_order_settlement")}},
		recipeStep{template: "SELLER_FUNDS_RELEASE", vars: map[string]string{"order_id": refundOrder, "seller_id": refundSeller, "order_ref": refundOrder}},
		recipeStep{template: "REFUND", vars: map[string]string{"psp_id": psp, "seller_id": refundSeller, "gross": usd(gross), "seller_net": usd(net), "commission": usd(commission), "order_ref": refundOrder, "original_posting_id": c.txReference(refundCapture)}},
	)
	disputeSeller, disputeOrder := c.account("seller_chargeback"), c.reference("disputed_order")
	disputeCapture := len(steps)
	steps = append(steps,
		recipeStep{template: "ORDER_PAYMENT_CAPTURED", vars: map[string]string{"psp_id": psp, "order_id": disputeOrder, "seller_id": disputeSeller, "gross": usd(gross), "commission": usd(commission), "psp_fee": usd(fee), "order_ref": disputeOrder}},
		recipeStep{template: "PSP_SETTLEMENT", vars: map[string]string{"psp_id": psp, "bank_id": bank, "gross_settled": usd(gross), "fees_retained": usd(fee), "settlement_ref": c.reference("disputed_order_settlement")}},
		recipeStep{template: "SELLER_FUNDS_RELEASE", vars: map[string]string{"order_id": disputeOrder, "seller_id": disputeSeller, "order_ref": disputeOrder}},
		recipeStep{template: "CHARGEBACK", vars: map[string]string{"psp_id": psp, "seller_id": disputeSeller, "gross": usd(gross), "order_ref": disputeOrder, "dispute_ref": c.txReference(disputeCapture)}},
		recipeStep{template: "RECON_ADJUSTMENT", vars: map[string]string{"psp_id": psp, "amount": usd(35), "recon_ref": c.reference("reconciliation")}},
	)
	return steps
}

func bnplLendingSteps(c cycleContext) []recipeStep {
	borrower, merchant := c.account("borrower"), c.account("merchant")
	psp, bank := c.account("psp"), c.account("bank")
	plan := c.reference("plan")
	principal := c.amount / 4
	discount := int64(300)
	interest, lateFee := int64(100), int64(50)
	collection := principal + interest + lateFee
	recovery := principal / 2
	settlement := principal*4 - discount
	usd := func(amount int64) string { return monetary("USD/2", amount) }
	return []recipeStep{
		{template: "MERCHANT_PURCHASE_FINANCED", vars: map[string]string{"borrower_id": borrower, "plan_id": plan, "merchant_id": merchant, "principal_1": usd(principal), "principal_2": usd(principal), "principal_3": usd(principal), "principal_4": usd(principal), "merchant_discount_fee": usd(discount)}},
		{template: "INTEREST_ACCRUAL", vars: map[string]string{"borrower_id": borrower, "plan_id": plan, "seq": "1", "interest_amount": usd(interest)}},
		{template: "LATE_FEE_ASSESSED", vars: map[string]string{"borrower_id": borrower, "plan_id": plan, "seq": "1", "fee_amount": usd(lateFee)}},
		{template: "INSTALLMENT_COLLECTION", vars: map[string]string{"borrower_id": borrower, "plan_id": plan, "seq": "1", "psp_id": psp, "payment": usd(collection)}},
		{template: "INSTALLMENT_COLLECTION_RETURN", vars: map[string]string{"borrower_id": borrower, "plan_id": plan, "seq": "1", "psp_id": psp, "fees_collected": usd(lateFee), "interest_collected": usd(interest), "principal_collected": usd(principal), "original_posting_id": c.txReference(3)}},
		{template: "INSTALLMENT_COLLECTION", vars: map[string]string{"borrower_id": borrower, "plan_id": plan, "seq": "1", "psp_id": psp, "payment": usd(collection)}},
		{template: "INTEREST_ACCRUAL", vars: map[string]string{"borrower_id": borrower, "plan_id": plan, "seq": "2", "interest_amount": usd(80)}},
		{template: "LATE_FEE_ASSESSED", vars: map[string]string{"borrower_id": borrower, "plan_id": plan, "seq": "2", "fee_amount": usd(40)}},
		{template: "PLAN_WRITE_OFF", vars: map[string]string{"borrower_id": borrower, "plan_id": plan, "seq": "2"}},
		{template: "RECOVERY", vars: map[string]string{"borrower_id": borrower, "plan_id": plan, "seq": "2", "psp_id": psp, "amount": usd(recovery)}},
		{template: "PSP_REMITTANCE", vars: map[string]string{"psp_id": psp, "bank_id": bank, "remittance_id": c.reference("remittance"), "amount": usd(collection + recovery)}},
		{template: "MERCHANT_WEEKLY_SETTLEMENT", vars: map[string]string{"merchant_id": merchant, "bank_id": bank, "settlement_id": c.reference("merchant_settlement")}},
		{template: "MERCHANT_SETTLEMENT_RETURN", vars: map[string]string{"merchant_id": merchant, "bank_id": bank, "settlement_id": c.reference("merchant_settlement"), "amount": usd(settlement), "original_posting_id": c.txReference(11)}},
		{template: "MERCHANT_WEEKLY_SETTLEMENT", vars: map[string]string{"merchant_id": merchant, "bank_id": bank, "settlement_id": c.reference("merchant_settlement_retry")}},
	}
}
