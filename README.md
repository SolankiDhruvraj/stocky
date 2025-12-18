# Stocky - Stock Rewards API

Stocky is a Go-based backend application that allows users to earn shares of Indian stocks (e.g., Reliance, TCS, Infosys) as incentives for actions like onboarding, referrals, or trading milestones.

## Features

- **Stock Rewards**: Grant fractional shares of stocks to users.
- **Internal Ledger**: Automatically tracks company cash-out, stock inventory, and internal fees (brokerage, taxes) for every reward.
- **Portfolio Management**: Real-time valuation of user holdings based on the latest market prices.
- **Historical Valuation**: Daily snapshots of user portfolio value in INR.
- **Reward Reversal**: Ability to revert rewards, which automatically adjusts user holdings and updates the internal status.
- **Idempotency**: Built-in protection against duplicate reward processing using unique idempotency keys.
- **Stale Price Check**: Ensures valuations use fresh data by ignoring prices older than 15 minutes.
- **Mock Price Service**: A background service that simulates a live market by updating stock prices every hour.

## 🛠 Tech Stack

- **Language**: Go (Golang)
- **Framework**: Gin Web Framework
- **Database**: PostgreSQL
- **Libraries**: `sqlx` (database), `decimal` (financial precision), `logrus` (logging), `godotenv` (config).
- **Deployment**: Docker & Railway

## API Endpoints

### System
- `GET /health`: Check server status.

### Rewards
- `POST /reward`: Grant a reward.
- `POST /reward/:id/revert`: Reverse a reward.

### User Data
- `GET /portfolio/:userId`: Get current holdings and total value.
- `GET /stats/:userId`: Get summary statistics.
- `GET /today-stocks/:userId`: List rewards granted today.
- `GET /historical-inr/:userId`: Get daily historical valuation.

### Local Setup
1. Clone the repository.
2. Create a `.env` file:
   ```env
   POSTGRES_URL=postgres://user:pass@localhost:5432/stocky?sslmode=disable
   ```
3. Run migrations:
   ```bash
   psql "$POSTGRES_URL" -f migrations/0001_init.up.sql
   psql "$POSTGRES_URL" -f migrations/0002_seed.up.sql
   psql "$POSTGRES_URL" -f migrations/0003_change_uuid_to_text.up.sql
   psql "$POSTGRES_URL" -f migrations/0004_add_reward_status.up.sql
   ```
4. Run the application:
   ```bash
   make run
   ```

### Docker Setup
```bash
docker build -t stocky .
docker run -p 8080:8080 --env-file .env stocky
```

## Deployment (Railway)

1. Connect the GitHub repository to Railway.
2. Add a PostgreSQL database service.
3. Set the `POSTGRES_URL` environment variable in your Go service.
4. **Custom Domain**: 
   - Add `api.dhruvrajsolanki.in` in Railway Networking settings.
   - Configure a `CNAME` record in Hostinger pointing to your Railway app URL.
5. Run migrations against the Railway database URL.

**Live API URL**: [https://api.dhruvrajsolanki.in](https://api.dhruvrajsolanki.in)


## Postman Collection

You can test all API endpoints using the Postman collection below:

👉 **Postman Collection:**  
[https://www.postman.com/campaa/workspace/new-workspace/collection/23889350-11a3ec58-c3b4-4374-9df9-6b256c959482?action=share&creator=23889350](https://www.postman.com/campaa/workspace/new-workspace/collection/23889350-11a3ec58-c3b4-4374-9df9-6b256c959482?action=share&creator=23889350)

