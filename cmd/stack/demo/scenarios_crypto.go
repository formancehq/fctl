package demo

import "strconv"

// cryptoCustodySteps exercises deposits, both trade directions, settled and
// returned withdrawals, hot-wallet refills, and a stranded trade compensation.
// Every spend follows the acknowledgement of its funding transaction.
func cryptoCustodySteps(c cycleContext) []recipeStep {
	customer, bank := c.account("customer"), c.account("bank")
	custodian, network := c.account("custodian"), c.account("network")
	fiatDeposit, cryptoDeposit := c.reference("fiat_deposit"), c.reference("crypto_deposit")
	buy, sell, compensated := c.reference("buy"), c.reference("sell"), c.reference("compensated")
	fiatSettled, fiatReturned := c.reference("fiat_settled"), c.reference("fiat_returned")
	cryptoSettled, cryptoCancelled := c.reference("crypto_settled"), c.reference("crypto_cancelled")
	usd := func(amount int64) string { return monetary("USD/2", amount) }
	crypto := func(amount int64) string { return monetary("USDC/6", amount) }
	return []recipeStep{
		{template: "FIAT_DEPOSIT_INITIATE", vars: map[string]string{"customer_id": customer, "bank_id": bank, "amount": usd(c.amount * 10), "deposit_id": fiatDeposit}},
		{template: "FIAT_DEPOSIT_SETTLE", vars: map[string]string{"customer_id": customer, "bank_id": bank, "amount": usd(c.amount * 10), "deposit_id": fiatDeposit}},
		{template: "CRYPTO_DEPOSIT_DETECTED", vars: map[string]string{"customer_id": customer, "custodian": custodian, "amount": crypto(c.amount * 10000), "deposit_id": cryptoDeposit}},
		{template: "CRYPTO_DEPOSIT_CONFIRMED", vars: map[string]string{"customer_id": customer, "amount": crypto(c.amount * 10000), "deposit_id": cryptoDeposit}},
		{template: "BUY_TRADE_INITIATE", vars: map[string]string{"customer_id": customer, "conversion_id": buy, "usd_gross": usd(c.amount)}},
		{template: "BUY_TRADE_SETTLE", vars: map[string]string{"customer_id": customer, "conversion_id": buy, "custodian": custodian, "usd_gross": usd(c.amount), "spread": usd(100), "crypto_amount": crypto((c.amount - 100) * 10000)}},
		{template: "SELL_TRADE_INITIATE", vars: map[string]string{"customer_id": customer, "conversion_id": sell, "crypto_amount": crypto(50000000)}},
		{template: "SELL_TRADE_SETTLE", vars: map[string]string{"customer_id": customer, "conversion_id": sell, "custodian": custodian, "crypto_amount": crypto(50000000), "usd_gross": usd(5000), "spread": usd(50)}},
		{template: "FIAT_WITHDRAWAL_INITIATE", vars: map[string]string{"customer_id": customer, "bank_id": bank, "amount": usd(1000), "withdrawal_id": fiatSettled}},
		{template: "FIAT_WITHDRAWAL_SETTLE", vars: map[string]string{"bank_id": bank, "amount": usd(1000), "withdrawal_id": fiatSettled}},
		{template: "FIAT_WITHDRAWAL_INITIATE", vars: map[string]string{"customer_id": customer, "bank_id": bank, "amount": usd(1000), "withdrawal_id": fiatReturned}},
		{template: "FIAT_WITHDRAWAL_RETURN", vars: map[string]string{"customer_id": customer, "bank_id": bank, "amount": usd(1000), "withdrawal_id": fiatReturned, "original_posting_id": c.txReference(10)}},
		{template: "CRYPTO_WITHDRAWAL_INITIATE", vars: map[string]string{"customer_id": customer, "withdrawal_id": cryptoSettled, "amount": crypto(10000000)}},
		{template: "CRYPTO_WITHDRAWAL_SETTLE", vars: map[string]string{"customer_id": customer, "withdrawal_id": cryptoSettled, "network": network, "amount": crypto(10000000), "network_fee": crypto(10000)}},
		{template: "CRYPTO_WITHDRAWAL_INITIATE", vars: map[string]string{"customer_id": customer, "withdrawal_id": cryptoCancelled, "amount": crypto(10000000)}},
		{template: "CRYPTO_WITHDRAWAL_CANCEL", vars: map[string]string{"customer_id": customer, "withdrawal_id": cryptoCancelled, "amount": crypto(10000000), "original_posting_id": c.txReference(14)}},
		{template: "CUSTODIAN_REFILL", vars: map[string]string{"custodian": custodian, "network": network, "amount": crypto(10000000), "refill_id": c.reference("refill")}},
		{template: "BUY_TRADE_INITIATE", vars: map[string]string{"customer_id": customer, "conversion_id": compensated, "usd_gross": usd(1000)}},
		{template: "CONVERSION_COMPENSATE", vars: map[string]string{"conversion_id": compensated, "return_account": "customers:" + customer + ":cash:available", "amount": usd(1000), "original_posting_id": c.txReference(17)}},
	}
}

// stablecoinIssuerSteps covers mint/return, transfer/reversal, redemption and a
// returned payout, reserve rebalancing in both outcomes, and segregated yield.
func stablecoinIssuerSteps(c cycleContext) []recipeStep {
	bank, secondBank := c.account("bank"), c.account("second_bank")
	holder, recipient, network := c.account("holder"), c.account("recipient"), c.account("network")
	mint, returnedMint := c.reference("mint"), c.reference("returned_mint")
	redemption := c.reference("redemption")
	rebalance, returnedRebalance := c.reference("rebalance"), c.reference("returned_rebalance")
	fiat := func(amount int64) string { return monetary("USD/2", amount) }
	token := func(amount int64) string { return monetary("USDC/6", amount) }
	return []recipeStep{
		{template: "MINT_INITIATE", vars: map[string]string{"mint_id": mint, "fiat_amount": fiat(c.amount * 10), "mint_ref": mint}},
		{template: "MINT_SETTLE", vars: map[string]string{"bank_id": bank, "mint_id": mint, "holder_id": holder, "network_id": network, "fiat_amount": fiat(c.amount * 10), "token_amount": token(c.amount * 100000), "mint_ref": mint}},
		{template: "MINT_INITIATE", vars: map[string]string{"mint_id": returnedMint, "fiat_amount": fiat(2000), "mint_ref": returnedMint}},
		{template: "MINT_RETURN", vars: map[string]string{"mint_id": returnedMint, "fiat_amount": fiat(2000), "original_posting_id": c.txReference(2)}},
		{template: "TRANSFER", vars: map[string]string{"from_holder_id": holder, "to_holder_id": recipient, "token_amount": token(10000000), "transfer_ref": c.reference("transfer")}},
		{template: "TRANSFER_REVERSE", vars: map[string]string{"from_holder_id": holder, "to_holder_id": recipient, "token_amount": token(10000000), "original_posting_id": c.txReference(4)}},
		{template: "REDEEM_REQUEST", vars: map[string]string{"holder_id": holder, "network_id": network, "redemption_id": redemption, "token_amount": token(100000000), "gross_fiat": fiat(10000), "fee": fiat(10), "redemption_ref": redemption}},
		{template: "REDEEM_SETTLE", vars: map[string]string{"bank_id": bank, "redemption_id": redemption, "gross_fiat": fiat(10000), "net_fiat": fiat(9990), "redemption_ref": redemption}},
		{template: "REDEEM_RETURN", vars: map[string]string{"bank_id": bank, "holder_id": holder, "network_id": network, "redemption_id": redemption, "token_amount": token(100000000), "net_fiat": fiat(9990), "fee": fiat(10), "original_posting_id": c.txReference(7)}},
		{template: "REBALANCE_INITIATE", vars: map[string]string{"from_bank_id": bank, "rebalance_id": rebalance, "fiat_amount": fiat(10000), "rebalance_ref": rebalance}},
		{template: "REBALANCE_SETTLE", vars: map[string]string{"to_bank_id": secondBank, "rebalance_id": rebalance, "fiat_amount": fiat(10000), "rebalance_ref": rebalance}},
		{template: "REBALANCE_INITIATE", vars: map[string]string{"from_bank_id": bank, "rebalance_id": returnedRebalance, "fiat_amount": fiat(5000), "rebalance_ref": returnedRebalance}},
		{template: "REBALANCE_RETURN", vars: map[string]string{"from_bank_id": bank, "rebalance_id": returnedRebalance, "fiat_amount": fiat(5000), "original_posting_id": c.txReference(11)}},
		{template: "YIELD_ACCRUE", vars: map[string]string{"bank_id": bank, "yield_amount": fiat(1000), "accrual_period": c.reference("accrual")}},
		{template: "YIELD_SWEEP", vars: map[string]string{"bank_id": bank, "yield_amount": fiat(1000), "sweep_period": c.reference("sweep")}},
	}
}

// stablecoinOperationsSteps preserves a one-to-one USD/USDC conversion across
// scales: one fiat minor unit corresponds to 10000 token minor units. The shared
// fiat reserve is only debited after this stream's own acknowledged credit.
func stablecoinOperationsSteps(c cycleContext) []recipeStep {
	psp, customer := c.account("psp"), c.account("customer")
	bank, network := c.account("bank"), c.account("network")
	authorization, mint, burn := c.reference("authorization"), c.reference("mint"), c.reference("burn")
	fiatAmount := strconv.FormatInt(c.amount, 10)
	stableAmount := strconv.FormatInt(c.amount*10000, 10)
	return []recipeStep{
		{template: "ONRAMP_STEP1_PAYMENT_AUTH_CREDIT", vars: map[string]string{"fiat_asset": "USD/2", "stable_asset": "USDC/6", "fiat_amount": fiatAmount, "stable_amount": stableAmount, "psp_id": psp, "client_id": customer, "authorization_id": authorization}},
		{template: "ONRAMP_STEP2_MINT_INSTRUCTION", vars: map[string]string{"stable_asset": "USDC/6", "stable_amount": stableAmount, "network": network, "mint_tx_hash": mint, "authorization_id": authorization}},
		{template: "ONRAMP_STEP3_MINT_CONFIRMATION", vars: map[string]string{"stable_asset": "USDC/6", "stable_amount": stableAmount, "network": network, "mint_tx_hash": mint, "block_number": "123456"}},
		{template: "ONRAMP_STEP4_PSP_SETTLEMENT", vars: map[string]string{"fiat_asset": "USD/2", "net_amount": strconv.FormatInt(c.amount-100, 10), "fee_amount": "100", "psp_id": psp, "bank_id": bank, "settlement_ref": c.reference("settlement")}},
		{template: "OFFRAMP_STEP1_BURN_INSTRUCTION", vars: map[string]string{"stable_asset": "USDC/6", "stable_amount": stableAmount, "network": network, "client_id": customer, "burn_tx_hash": burn}},
		{template: "OFFRAMP_STEP2_BURN_CONFIRMATION", vars: map[string]string{"fiat_asset": "USD/2", "stable_asset": "USDC/6", "fiat_amount": fiatAmount, "stable_amount": stableAmount, "network": network, "client_id": customer, "burn_tx_hash": burn, "block_number": "123457"}},
		{template: "OFFRAMP_STEP3_FIAT_WITHDRAWAL", vars: map[string]string{"fiat_asset": "USD/2", "fiat_amount": fiatAmount, "bank_id": bank, "client_id": customer, "transfer_ref": c.reference("fiat_withdrawal")}},
	}
}
