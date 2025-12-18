CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY,
  name TEXT
);

CREATE TABLE IF NOT EXISTS stocks (
  symbol TEXT PRIMARY KEY,
  name TEXT
);

CREATE TABLE IF NOT EXISTS rewards (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  symbol TEXT NOT NULL REFERENCES stocks(symbol),
  quantity NUMERIC(18,6) NOT NULL,
  timestamp TIMESTAMPTZ NOT NULL,
  idempotency_key TEXT,
  source TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE (idempotency_key)
);

CREATE TABLE IF NOT EXISTS ledger_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reward_id UUID REFERENCES rewards(id),
  entry_time TIMESTAMPTZ DEFAULT now(),
  account_debit TEXT NOT NULL,
  account_credit TEXT NOT NULL,
  amount_inr NUMERIC(18,4) NOT NULL,
  stock_symbol TEXT,
  stock_quantity NUMERIC(18,6),
  description TEXT
);

CREATE TABLE IF NOT EXISTS holdings (
  user_id UUID NOT NULL REFERENCES users(id),
  symbol TEXT NOT NULL REFERENCES stocks(symbol),
  quantity NUMERIC(18,6) NOT NULL DEFAULT 0,
  last_updated TIMESTAMPTZ DEFAULT now(),
  PRIMARY KEY (user_id, symbol)
);

CREATE TABLE IF NOT EXISTS price_history (
  id BIGSERIAL PRIMARY KEY,
  symbol TEXT NOT NULL REFERENCES stocks(symbol),
  price_inr NUMERIC(18,4) NOT NULL,
  timestamp TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS daily_valuations (
  user_id UUID,
  date DATE,
  total_inr NUMERIC(18,4),
  PRIMARY KEY (user_id, date)
);
