# RateEd

A community-driven platform for rating and reviewing educational institutions. Students can share their experiences, discover schools, and contribute to a transparent education ecosystem.

## Features

### Core Functionality
- **Institution Ratings & Reviews** — Rate schools on various metrics and write detailed reviews
- **User Profiles** — Build your profile, track education history, earn contribution points
- **Photo Uploads** — Share institution photos and verification photos for authenticity
- **Discussion Forums** — Comment and engage in conversations about institutions
- **Leaderboards** — View top-rated schools and active community contributors
- **Email Verification** — Verify institutional affiliations with email addresses

### User Management
- **Authentication** — Secure registration and login with hashed passwords
- **Session Management** — Cookie-based session tokens
- **Points System** — Earn contribution points for activity
- **User Moderation** — Ban system for managing problematic users
- **Admin Dashboard** — Manage users, verify submissions, monitor activity

## Tech Stack

**Backend**
- Go 1.24.0
- Gorilla Mux (HTTP routing)
- SQLite3 (database)
- golang.org/x/crypto (password hashing)

**Frontend**
- HTML5 Templates
- Vanilla JavaScript (ES6)
- CSS3

**Deployment**
- Docker containerization
- Railway deployment (configured)
- SQLite with persistent data directory

## Project Structure

```
RateEd/
├── project/
│   ├── main.go              # Entry point, router setup
│   ├── handlers.go          # HTTP request handlers
│   ├── auth.go              # Authentication logic
│   ├── db.go                # Database initialization & queries
│   ├── config.go            # Configuration management
│   ├── go.mod / go.sum      # Dependencies
│   ├── Dockerfile           # Container config
│   ├── static/
│   │   ├── app.js          # Frontend logic
│   │   └── style.css       # Styling
│   ├── templates/           # HTML pages
│   │   ├── landing.html    # Homepage
│   │   ├── login.html      # Auth page
│   │   ├── index.html      # Main app
│   │   ├── institution.html # School detail view
│   │   ├── profile.html    # User profiles
│   │   └── admin.html      # Admin dashboard
│   └── uploads/             # User-generated files (profiles, photos)
```

## Getting Started

### Prerequisites
- Go 1.24.0+
- SQLite3

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/VedadKrajina/RateEd.git
   cd RateEd/project
   ```

2. **Install dependencies**
   ```bash
   go mod download
   go mod tidy
   ```

3. **Run the application**
   ```bash
   go run .
   ```
   The server starts on `http://localhost:3141`

### Using Docker

```bash
cd project
docker build -t rateed .
docker run -p 3141:3141 -e PORT=3141 -v $(pwd):/app rateed
```

## API Endpoints

### Authentication
- `POST /api/register` — Create new account
- `POST /api/login` — Login user
- `POST /api/logout` — Logout user
- `GET /api/me` — Get current user info

### Institutions
- `GET /api/institutions` — List all institutions
- `POST /api/institutions` — Create institution (auth required)
- `GET /api/institutions/{id}` — Get institution details
- `POST /api/institutions/{id}/rate` — Rate an institution
- `POST /api/institutions/{id}/verify` — Verify email for institution
- `POST /api/institutions/{id}/photos` — Upload institution photos
- `GET /api/institutions/{id}/leaderboard` — View top contributors
- `GET /api/institutions/{id}/discussion` — Get comments
- `POST /api/institutions/{id}/discussion` — Post comment

### Profiles & Rankings
- `GET /api/users/{username}` — Get user profile
- `POST /api/profile/picture` — Upload profile picture
- `POST /api/profile/education` — Add education history
- `GET /api/schools/rankings` — Get top-rated schools
- `GET /api/stats` — Platform statistics

### Admin (requires admin role)
- `GET /api/admin/users` — List all users
- `PUT /api/admin/users/{id}/points` — Set user points
- `POST /api/admin/users/{id}/ban` — Ban user
- `GET /api/admin/verification-requests` — Review pending verifications

## Database Schema

RateEd uses SQLite with the following main tables:
- **users** — User accounts, credentials, points
- **institutions** — Schools/educational institutions
- **ratings** — User ratings and reviews
- **photos** — Institution and profile photos
- **comments** — Discussion threads
- **sessions** — Active user sessions
- **verifications** — Email verification requests

## Configuration

Environment variables:
- `PORT` — Server port (default: 3141)
- `DATA_DIR` — Directory for database and uploads (default: current directory)

## Development Roadmap

**Current Features**
- ✅ User authentication and profiles
- ✅ Institution ratings and reviews
- ✅ Photo uploads
- ✅ Discussion system
- ✅ Leaderboards
- ✅ Admin dashboard

**Potential Enhancements**
- [ ] API rate limiting
- [ ] Advanced search and filtering
- [ ] Category-specific ratings (teaching quality, facilities, etc.)
- [ ] Mobile app (React Native/Flutter)
- [ ] Email notifications
- [ ] Integration with external education datasets
- [ ] Machine learning for review moderation
- [ ] Export reports for institutions

## Contributing

For improvements or bug reports, open an issue or submit a pull request.

## License

Bosnian Science Project Olympiad 2026 (BOSEPO)

---

**Built by:** Vedad Krajina  
**Project:** RateEd (Educational Institution Rating Platform)  
**For:** BOSEPO 2026
