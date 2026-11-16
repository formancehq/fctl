package demo

import (
	"maps"
	"strconv"
)

func paymentAcceptanceSteps(c cycleContext) []recipeStep {
	amount := c.amount
	fee := amount / 100
	acquirer := c.account("acquirer")
	client := c.account("client")
	platform := c.account("platform")
	authorization := c.reference("authorization")
	return []recipeStep{
		{template: "CARD_AUTHORIZATION_GROSS_TOPUP", vars: map[string]string{
			"asset": "EUR/2", "amount": strconv.FormatInt(amount, 10), "acquirer_id": acquirer,
			"client_id": client, "platform_name": platform, "authorization_id": authorization,
		}},
		{template: "ACQUIRER_SETTLEMENT", vars: map[string]string{
			"asset": "EUR/2", "net_amount": strconv.FormatInt(amount-fee, 10), "fee_amount": strconv.FormatInt(fee, 10),
			"acquirer_id": acquirer, "bank_id": c.account("bank"), "platform_name": platform, "settlement_ref": c.reference("settlement"),
		}},
		{template: "CARD_REFUND", vars: map[string]string{
			"asset": "EUR/2", "amount": strconv.FormatInt(amount/10, 10), "acquirer_id": acquirer,
			"client_id": client, "refund_id": c.reference("refund"), "original_authorization_id": authorization,
		}},
		{template: "CHARGEBACK", vars: map[string]string{
			"asset": "EUR/2", "amount": strconv.FormatInt(amount/20, 10), "chargeback_fee": "25", "acquirer_id": acquirer,
			"client_id": client, "platform_name": platform, "chargeback_id": c.reference("chargeback"), "original_authorization_id": authorization,
		}},
	}
}

func issuingFinancialHostSteps(c cycleContext) []recipeStep {
	amount := c.amount
	account := c.account("cardholder")
	scheme := c.account("scheme")
	standardAuth := c.reference("authorization")
	tipAuth := c.reference("tip_authorization")
	partialAuth := c.reference("partial_authorization")
	refundAuth := c.reference("refund_authorization")
	chargeback := c.reference("chargeback")
	held := amount / 5
	incremental := amount / 10
	reversed := amount / 20
	captured := amount / 10
	tipHeld := amount / 10
	tip := amount / 100
	partialAvailable := 4*amount - captured - tipHeld - tip
	refund := amount / 20
	dispute := amount / 100
	stip := amount / 100
	base := map[string]string{
		"asset": "EUR/2", "account_id": account, "pii_id": c.reference("synthetic_customer"), "trx_details": "Synthetic demo card transaction",
	}
	vars := func(extra map[string]string) map[string]string {
		result := maps.Clone(base)
		maps.Copy(result, extra)
		return result
	}
	var steps []recipeStep
	add := func(template string, values map[string]string) {
		steps = append(steps, recipeStep{template: template, vars: values})
	}
	// The issuing cookbook has no top-up event. Seed a synthetic opening balance,
	// separately labelled from the official authorization/refund/dispute templates.
	steps = append(steps, recipeStep{template: "DEMO_BOOTSTRAP", script: `#![feature("experimental-account-interpolation")]
vars {
 account $account_id
 monetary $opening
}
send $opening (
 source = @world
 destination = @cardholder:$account_id:main
)
`, vars: map[string]string{"account_id": account, "opening": monetary("EUR/2", 4*amount)}})
	add("CARD_AUTHORIZATION_APPROVED", vars(map[string]string{"amount": strconv.FormatInt(held, 10), "overdraft": "0", "authorization_id": standardAuth}))
	add("CARD_AUTHORIZATION_INCREMENTAL", vars(map[string]string{"amount": strconv.FormatInt(incremental, 10), "overdraft": "0", "authorization_id": standardAuth}))
	add("AUTHORIZATION_REVERSAL", vars(map[string]string{"amount": strconv.FormatInt(reversed, 10), "authorization_id": standardAuth, "reversal_id": c.reference("partial_reversal")}))
	add("PRESENTMENT", vars(map[string]string{"amount": strconv.FormatInt(captured, 10), "authorization_id": standardAuth, "presentment_id": c.reference("presentment"), "scheme_id": scheme}))
	add("HOLD_REVERSAL_WILDCARD", vars(map[string]string{"authorization_id": standardAuth, "reversal_id": c.reference("remaining_reversal")}))
	add("CARD_AUTHORIZATION_APPROVED", vars(map[string]string{"amount": strconv.FormatInt(tipHeld, 10), "overdraft": "0", "authorization_id": tipAuth}))
	add("PRESENTMENT_WITH_TIP", vars(map[string]string{"auth_amount": strconv.FormatInt(tipHeld, 10), "additional_amount": strconv.FormatInt(tip, 10), "authorization_id": tipAuth, "presentment_id": c.reference("tip_presentment"), "scheme_id": scheme}))
	// Request more than the remaining balance: the partial-approval template must
	// approve only that balance. Its complete capture leaves main at zero.
	add("CARD_AUTHORIZATION_PARTIAL", vars(map[string]string{"amount": strconv.FormatInt(partialAvailable+amount, 10), "overdraft": "0", "authorization_id": partialAuth}))
	add("PRESENTMENT", vars(map[string]string{"amount": strconv.FormatInt(partialAvailable, 10), "authorization_id": partialAuth, "presentment_id": c.reference("partial_presentment"), "scheme_id": scheme}))
	add("REFUND_AUTHORIZATION", vars(map[string]string{"amount": strconv.FormatInt(refund, 10), "refund_auth_id": refundAuth, "scheme_id": scheme}))
	refundVars := vars(map[string]string{"amount": strconv.FormatInt(refund, 10), "refund_auth_id": refundAuth, "refund_posting_id": c.reference("refund_posting")})
	delete(refundVars, "pii_id")
	add("REFUND_POSTING", refundVars)
	add("CHARGEBACK_ACCEPTANCE", vars(map[string]string{"amount": strconv.FormatInt(dispute, 10), "chargeback_id": chargeback, "scheme_id": scheme, "original_presentment_id": c.reference("partial_presentment")}))
	confirmation := vars(map[string]string{"amount": strconv.FormatInt(dispute, 10), "chargeback_id": chargeback, "scheme_id": scheme, "settlement_ref": c.reference("chargeback_settlement")})
	delete(confirmation, "account_id")
	delete(confirmation, "pii_id")
	add("CHARGEBACK_CONFIRMATION", confirmation)
	add("SECOND_PRESENTMENT", vars(map[string]string{"amount": strconv.FormatInt(dispute, 10), "chargeback_id": chargeback, "second_presentment_id": c.reference("second_presentment"), "scheme_id": scheme}))
	add("STIP_ADVICE", vars(map[string]string{"amount": strconv.FormatInt(stip, 10), "stip_advice_id": c.reference("stip_advice"), "scheme_id": scheme}))
	// Spend the remaining refund without overdraft, restoring the opening account
	// to zero so subsequent cycles exercise exactly the same partial-approval case.
	add("OFFLINE_PRESENTMENT", vars(map[string]string{"amount": strconv.FormatInt(refund-stip, 10), "presentment_id": c.reference("offline_presentment"), "scheme_id": scheme}))
	return steps
}

func neobankFBOSteps(c cycleContext) []recipeStep {
	amount := c.amount
	customer := c.account("customer")
	recipient := c.account("recipient")
	var steps []recipeStep
	add := func(template string, values map[string]string) {
		steps = append(steps, recipeStep{template: template, vars: values})
	}
	if c.cycle == 0 {
		// The two official operating movements transfer existing platform float but
		// neither creates it. Seed the opening float once per independent stream.
		steps = append(steps, recipeStep{template: "DEMO_BOOTSTRAP", script: `vars {
 monetary $opening
}
send $opening (
 source = @world
 destination = @platform:banks:corporate:operating
)
`, vars: map[string]string{"opening": monetary("USD/2", 1000)}})
	}
	add("ACH_DIRECT_DEPOSIT", map[string]string{"customer_id": customer, "amount": monetary("USD/2", amount), "deposit_id": c.reference("deposit"), "originator": "Synthetic Employer"})
	add("INCOMING_WIRE", map[string]string{"customer_id": customer, "amount": monetary("USD/2", amount/2), "wire_id": c.reference("wire"), "originator": "Synthetic Sender"})
	add("ACH_DIRECT_DEPOSIT", map[string]string{"customer_id": customer, "amount": monetary("USD/2", amount/10), "deposit_id": c.reference("returned_deposit"), "originator": "Synthetic Employer"})
	add("ACH_DEPOSIT_RETURN", map[string]string{"customer_id": customer, "amount": monetary("USD/2", amount/10), "deposit_id": c.reference("returned_deposit")})
	add("CARD_AUTH", map[string]string{"customer_id": customer, "auth_id": c.reference("reversed_auth"), "amount": monetary("USD/2", amount/10)})
	add("CARD_AUTH_REVERSE", map[string]string{"customer_id": customer, "auth_id": c.reference("reversed_auth"), "amount": monetary("USD/2", amount/10)})
	add("CARD_AUTH", map[string]string{"customer_id": customer, "auth_id": c.reference("expired_auth"), "amount": monetary("USD/2", amount/10)})
	add("CARD_AUTH_EXPIRE", map[string]string{"customer_id": customer, "auth_id": c.reference("expired_auth")})
	add("CARD_AUTH", map[string]string{"customer_id": customer, "auth_id": c.reference("captured_auth"), "amount": monetary("USD/2", amount/5)})
	captureIndex := len(steps)
	add("CARD_CAPTURE", map[string]string{"customer_id": customer, "auth_id": c.reference("captured_auth"), "capture_amount": monetary("USD/2", amount/10)})
	add("CARD_REFUND", map[string]string{"customer_id": customer, "amount": monetary("USD/2", amount/10), "refund_id": c.reference("card_refund"), "original_capture_id": c.txReference(captureIndex)})
	add("P2P_TRANSFER", map[string]string{"from_customer_id": customer, "to_customer_id": recipient, "amount": monetary("USD/2", amount/10), "transfer_id": c.reference("p2p_outbound")})
	add("P2P_TRANSFER", map[string]string{"from_customer_id": recipient, "to_customer_id": customer, "amount": monetary("USD/2", amount/10), "transfer_id": c.reference("p2p_return")})
	add("ACH_WITHDRAWAL_RESERVE", map[string]string{"customer_id": customer, "withdrawal_id": c.reference("settled_withdrawal"), "amount": monetary("USD/2", amount/10)})
	add("ACH_WITHDRAWAL_SETTLE", map[string]string{"customer_id": customer, "withdrawal_id": c.reference("settled_withdrawal"), "amount": monetary("USD/2", amount/10)})
	add("ACH_WITHDRAWAL_RESERVE", map[string]string{"customer_id": customer, "withdrawal_id": c.reference("returned_withdrawal"), "amount": monetary("USD/2", amount/20)})
	add("ACH_WITHDRAWAL_RETURN", map[string]string{"customer_id": customer, "withdrawal_id": c.reference("returned_withdrawal"), "amount": monetary("USD/2", amount/20)})
	add("ADVANCE_ORIGINATION", map[string]string{"customer_id": customer, "advance_id": c.reference("settled_advance"), "amount": monetary("USD/2", amount/10)})
	add("ADVANCE_SETTLEMENT", map[string]string{"customer_id": customer, "advance_id": c.reference("settled_advance"), "deposit_amount": monetary("USD/2", amount/5)})
	add("ADVANCE_ORIGINATION", map[string]string{"customer_id": customer, "advance_id": c.reference("written_off_advance"), "amount": monetary("USD/2", amount/20)})
	add("ADVANCE_WRITEOFF", map[string]string{"customer_id": customer, "advance_id": c.reference("written_off_advance")})
	add("FBO_INTEREST_SWEEP", map[string]string{"amount": monetary("USD/2", 50), "period_id": c.reference("interest_period")})
	add("OPERATING_MOVEMENT_TO_FBO", map[string]string{"amount": monetary("USD/2", 1000), "movement_id": c.reference("float_topup")})
	add("OPERATING_MOVEMENT_TO_CORPORATE", map[string]string{"amount": monetary("USD/2", 1000), "movement_id": c.reference("float_sweep")})
	return steps
}
