# Prompt Market

> Marketplace for buying and selling AI prompts

## 🛠 Tech Stack
- **Backend:** Go 1.21+, Fiber v2
- **Database:** PostgreSQL 15
- **Cache:** Redis 7
- **Auth:** JWT
- **ORM:** GORM
- **Deploy:** Docker

## 📁 Structure
promt-market/
├──cmd/api/ # Entry point
├── internal/
│ ├── config/ # Config (Viper)
│ ├── domain/ # Models
│ ├── repository/ # Database
│ ├── service/ # Business logic
│ ├── handler/ # HTTP handlers
│ └── middleware/ # Fiber middleware
├── pkg/ # Reusable packages
├── migrations/ # DB migrations
└── docs/ # Documentation

text

## 🗄 Main Tables
- **users** - Auth, profiles, wallet
- **prompts** - Title, content, price, images, SEO
- **orders** - Purchases, payments
- **reviews** - Ratings & feedback

## 🔐 Auth
- JWT with Access (15m) + Refresh (7d)
- bcrypt for passwords
- Rate limiting: 100 req/min

## 📡 Key Endpoints
Public:
GET /api/v1/prompts # List + filters
GET /api/v1/prompts/:slug # Details
POST /api/v1/auth/register # Signup
POST /api/v1/auth/login # Login

Protected:
POST /api/v1/prompts # Create
PUT /api/v1/prompts/:id # Update
POST /api/v1/orders # Purchase
POST /api/v1/reviews # Rate

Admin:
GET /api/v1/admin/users
PUT /api/v1/admin/prompts/:id/approve
GET /api/v1/admin/stats

text

## 🚀 Quick Start
```bash
cp .env.example .env
make docker    # or make run
make migrate-up
🔧 ENV Variables
env
APP_PORT=8080
DB_HOST=localhost
DB_NAME=promt_market
JWT_SECRET=your-secret
REDIS_HOST=localhost
📦 Commands
bash
make run          # Run locally
make build        # Build binary
make test         # Run tests
make docker       # Docker up
make migrate-up   # Run migrations
🔒 Security
bcrypt password hashing

JWT with refresh rotation

Rate limiting

SQL injection protection (GORM)

XSS sanitization

📊 Monitoring
Logs: Logrus (JSON format)

Metrics: Prometheus (optional)

Errors: Sentry (optional)

Version: 1.0.0
Status: 🚧 Development

text

---


```bash
# Create docs directory
mkdir -p docs

# Create and save the file
cat > docs/README.md << 'EOF'
[محتوای بالا رو کامل کپی کن اینجا]
EOF

# Zip it
zip -r docs.zip docs/

# Done!