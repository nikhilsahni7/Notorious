# 🔍 Notorious Search - Complete Implementation

A production-ready search system with authentication, role-based access, search limits, and comprehensive admin dashboard.

## 🎯 Features

### ✅ Authentication & Authorization

- JWT-based authentication (24-hour tokens)
- Role-based access (Admin/User)
- Bcrypt password hashing
- Protected routes with auto-redirect

### ✅ Search System

- OpenSearch integration (500GB+ data)
- Daily search limits per user
- Smart counting (only results > 0)
- Search history tracking
- IST timezone support (auto-reset at 12 AM IST)
- Real-time limit tracking

### ✅ Admin Dashboard

- **Dashboard Tab** - System overview & statistics
- **Users Tab** - Create, update, delete users
- **Requests Tab** - Approve/reject access requests
- **Search History Tab** - Monitor all searches

### ✅ User Management

- Create users with custom limits
- Update user details & limits
- Activate/deactivate accounts
- Track search usage per user
- View individual search history

## 🚀 Quick Start

### Prerequisites

- Go 1.24+
- PostgreSQL
- Node.js 18+
- pnpm
- OpenSearch (already configured)

### 1. Backend Setup

```bash
cd backend

# Setup database
createdb notorious
psql "postgresql://postgres:rajni.surender1@localhost:5432/notorious" \
  -f migrations/001_init_schema.sql

# Create .env
cat > .env << EOF
DATABASE_URL=postgresql://postgres:rajni.surender1@localhost:5432/notorious
JWT_SECRET=your-super-secret-jwt-key-min-32-chars-for-production
EOF

# Run server
go run main.go
# Server on http://localhost:8080
```

### 2. Frontend Setup

```bash
cd frontend

# Install dependencies
pnpm install

# Optional: Create .env.local
echo 'NEXT_PUBLIC_API_URL=http://localhost:8080' > .env.local

# Run development server
pnpm dev
# Frontend on http://localhost:3000
```

### 3. Login

Open `http://localhost:3000`

**Admin Credentials:**

```
Email: admin@notorious.com
Password: admin123
```

**⚠️ Change this immediately in production!**

## 📁 Project Structure

```
notorious/
├── backend/                    # Go backend
│   ├── main.go                 # Entry point (Gin + routes)
│   ├── .env                    # Database & JWT config
│   ├── migrations/             # Database schema
│   └── internal/
│       ├── auth/               # JWT & password utils
│       ├── database/           # PostgreSQL connection
│       ├── middleware/         # Auth middleware
│       ├── models/             # Data models
│       ├── repository/         # Database operations
│       ├── scheduler/          # Daily limit reset
│       ├── handlers/           # API handlers
│       ├── services/           # OpenSearch service
│       └── config/             # Configuration
└── frontend/                   # Next.js frontend
    ├── src/
    │   ├── app/                # Pages (Next.js 14 app router)
    │   │   ├── page.tsx        # Home (auto-redirect)
    │   │   ├── login/          # Login page
    │   │   ├── request-access/ # Request form
    │   │   ├── search/         # Search page
    │   │   └── admin/          # Admin dashboard
    │   ├── components/         # Reusable UI components
    │   │   └── admin/          # Admin dashboard components
    │   ├── contexts/           # React contexts (Auth)
    │   ├── hooks/              # Custom hooks
    │   ├── services/           # API service layer
    │   ├── lib/                # Utilities (API client)
    │   ├── config/             # Configuration (API URLs)
    │   ├── types/              # TypeScript types
    │   └── utils/              # Helper functions
    └── .env.local              # API URL configuration
```

## 🎨 User Interface

### Login & Redirect Logic

1. Visit `/` → Auto-redirects based on auth status:
   - **Not logged in** → `/login`
   - **Admin** → `/admin`
   - **Regular user** → `/search`

### Search Page (`/search`)

- Search form with AND/OR operators
- Real-time limit tracking with progress bar
- User info: Name, email, current usage
- Color-coded warnings (green → yellow → red)
- Results table with pagination
- Client-side filtering
- Copy to clipboard

### Admin Dashboard (`/admin`)

- **Stats Tab**: Overview metrics
- **Users Tab**: Full user management
- **Requests Tab**: Access request workflow
- **Search History Tab**: System-wide search tracking

## 🔐 Security

- JWT tokens with expiry
- Bcrypt password hashing (cost: 12)
- Protected API routes
- Role-based authorization
- Active/inactive account status
- Session management

## 📊 Database Schema

### Users Table

```sql
- id (UUID)
- email (unique)
- password_hash
- name, phone, role
- daily_search_limit
- searches_used_today
- is_active
- last_reset_date (for IST timezone)
```

### User Requests Table

```sql
- id (UUID)
- email, name, phone
- requested_searches_per_day
- status (pending/approved/rejected)
- admin_notes
```

### Search History Table

```sql
- id (UUID)
- user_id (FK)
- query
- total_results
- top_results (JSONB - top 5)
- searched_at
```

## 📡 API Endpoints

### Public

```
POST /auth/login                    # Login
POST /auth/request-access           # Request account
```

### Authenticated (All Users)

```
GET  /search                        # Search with tracking
POST /search                        # Search with tracking
```

### Admin Only

```
GET    /api/admin/users                            # List users
POST   /api/admin/users                            # Create user
GET    /api/admin/users/:id                        # Get user
PUT    /api/admin/users/:id                        # Update user
DELETE /api/admin/users/:id                        # Delete user
GET    /api/admin/user-requests                    # List requests
POST   /api/admin/user-requests/:id/approve        # Approve request
POST   /api/admin/user-requests/:id/reject         # Reject request
GET    /api/admin/search-history                   # All search history
GET    /api/admin/users/:id/search-history         # User search history
```

## 🔧 Configuration

### Change API URL (Single Place!)

```typescript
// frontend/src/config/api.ts
export const API_CONFIG = {
  BASE_URL: process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080",
  // All endpoints defined here
};
```

### Environment Variables

**Backend (.env):**

```
DATABASE_URL=postgresql://user:pass@localhost:5432/notorious
JWT_SECRET=your-secret-key-min-32-chars
```

**Frontend (.env.local):**

```
NEXT_PUBLIC_API_URL=http://localhost:8080
```

## 📚 Documentation

- **ADMIN_CREDENTIALS.md** - Admin access & full feature guide
- **backend/README_AUTH.md** - API authentication details
- **IMPLEMENTATION_COMPLETE.md** - Technical implementation details

## 🧪 Testing

### Test Admin Login

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@notorious.com","password":"admin123"}'
```

### Test Search (with token)

```bash
TOKEN="your-jwt-token"

curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "query": "name:John",
    "fields": ["name", "fname", "mobile"],
    "and_or": "AND",
    "size": 50
  }'
```

## 🛠️ Tech Stack

**Backend:**

- Go 1.24
- Gin (HTTP framework)
- PostgreSQL (database)
- pgx (PostgreSQL driver)
- JWT authentication
- bcrypt password hashing
- OpenSearch (search engine)

**Frontend:**

- Next.js 15.5 (App Router)
- TypeScript
- React 18
- Tailwind CSS
- pnpm (package manager)

## ✨ Code Quality

- ✅ Industry-standard architecture
- ✅ Service layer pattern
- ✅ Custom hooks for reusability
- ✅ Centralized API configuration
- ✅ Full TypeScript coverage
- ✅ Error handling throughout
- ✅ Modular & maintainable
- ✅ Production-ready

## 🐛 Troubleshooting

### Backend won't start

```bash
# Check database connection
psql "postgresql://postgres:rajni.surender1@localhost:5432/notorious" -c "SELECT 1"

# Check .env exists
cat backend/.env

# Rebuild
cd backend && go build -o notorious main.go
```

### Frontend errors

```bash
# Clear and reinstall
cd frontend
rm -rf .next node_modules
pnpm install
pnpm dev
```

### Admin cannot login

```bash
# Verify admin exists
psql "postgresql://postgres:rajni.surender1@localhost:5432/notorious" \
  -c "SELECT * FROM users WHERE email='admin@notorious.com'"

# Re-run migrations if needed
psql "postgresql://postgres:rajni.surender1@localhost:5432/notorious" \
  -f backend/migrations/001_init_schema.sql
```

## 🎉 Features Summary

| Feature          | Status | Description                                 |
| ---------------- | ------ | ------------------------------------------- |
| Authentication   | ✅     | JWT-based with role support                 |
| Search Limits    | ✅     | Per-user daily limits with IST reset        |
| Search Tracking  | ✅     | Complete audit trail with top results       |
| Admin Dashboard  | ✅     | Full user & system management               |
| User Management  | ✅     | Create, update, delete, activate/deactivate |
| Access Requests  | ✅     | Workflow for new user approval              |
| Search History   | ✅     | System-wide and per-user tracking           |
| Auto-redirect    | ✅     | Role-based routing on login                 |
| Real-time UI     | ✅     | Live limit tracking with progress bar       |
| Production Ready | ✅     | Error handling, security, best practices    |

## 📝 License

Private project

## 🤝 Support

For detailed instructions, see **ADMIN_CREDENTIALS.md**

---

**Built with ❤️ using Go, PostgreSQL, Next.js, and TypeScript**
