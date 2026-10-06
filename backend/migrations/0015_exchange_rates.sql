-- 0015_exchange_rates: display/settlement rates from the base currency.

CREATE TABLE IF NOT EXISTS exchange_rates (
    currency   VARCHAR(8) PRIMARY KEY,
    rate_micro BIGINT NOT NULL CHECK (rate_micro > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
