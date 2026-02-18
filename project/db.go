package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB(dataDir string) {
	var err error
	db, err = sql.Open("sqlite3", dataDir+"/rateit.db?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		log.Fatal(err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS topics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT UNIQUE NOT NULL,
		institution_type TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		cover_image TEXT NOT NULL DEFAULT '',
		created_by INTEGER NOT NULL REFERENCES users(id),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS ratings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		topic_id INTEGER NOT NULL REFERENCES topics(id),
		user_id INTEGER NOT NULL REFERENCES users(id),
		score INTEGER NOT NULL CHECK(score >= 1 AND score <= 5),
		comment TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(topic_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS user_profiles (
		user_id INTEGER PRIMARY KEY REFERENCES users(id),
		profile_picture TEXT NOT NULL DEFAULT '',
		contribution_points INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS discussion_comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		institution_id INTEGER NOT NULL REFERENCES topics(id),
		user_id INTEGER NOT NULL REFERENCES users(id),
		parent_id INTEGER REFERENCES discussion_comments(id),
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS contribution_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id),
		event_type TEXT NOT NULL,
		points INTEGER NOT NULL,
		reference_id INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		log.Fatal(err)
	}

	// Idempotent migrations
	migrations := []string{
		"ALTER TABLE ratings ADD COLUMN score_academic INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE ratings ADD COLUMN score_infrastructure INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE ratings ADD COLUMN score_student_life INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE ratings ADD COLUMN score_career INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE ratings ADD COLUMN score_guidance INTEGER NOT NULL DEFAULT 0",
	}
	for _, m := range migrations {
		db.Exec(m) // ignore errors (column already exists)
	}

	// email_domain on topics
	db.Exec("ALTER TABLE topics ADD COLUMN email_domain TEXT NOT NULL DEFAULT ''")

	// User verifications table
	db.Exec(`CREATE TABLE IF NOT EXISTS user_verifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id),
		institution_id INTEGER NOT NULL REFERENCES topics(id),
		verified_email TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, institution_id)
	)`)

	// Education history table
	db.Exec(`CREATE TABLE IF NOT EXISTS education_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id),
		institution_name TEXT NOT NULL DEFAULT '',
		institution_id INTEGER REFERENCES topics(id),
		start_date TEXT NOT NULL DEFAULT '',
		end_date TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// New institution metadata columns
	db.Exec("ALTER TABLE topics ADD COLUMN city TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE topics ADD COLUMN ownership TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE topics ADD COLUMN tuition_min INTEGER NOT NULL DEFAULT 0")
	db.Exec("ALTER TABLE topics ADD COLUMN tuition_max INTEGER NOT NULL DEFAULT 0")
	db.Exec("ALTER TABLE topics ADD COLUMN programs TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE topics ADD COLUMN pros TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE topics ADD COLUMN cons TEXT NOT NULL DEFAULT ''")

	// Institution photos table
	db.Exec(`CREATE TABLE IF NOT EXISTS institution_photos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		institution_id INTEGER NOT NULL REFERENCES topics(id),
		path TEXT NOT NULL,
		uploaded_by INTEGER NOT NULL REFERENCES users(id),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// User mutes table
	db.Exec(`CREATE TABLE IF NOT EXISTS user_mutes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		institution_id INTEGER NOT NULL REFERENCES topics(id),
		user_id INTEGER NOT NULL REFERENCES users(id),
		muted_until DATETIME NOT NULL,
		muted_by INTEGER NOT NULL REFERENCES users(id),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// Verification requests table
	db.Exec(`CREATE TABLE IF NOT EXISTS verification_requests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		institution_id INTEGER NOT NULL REFERENCES topics(id),
		user_id INTEGER NOT NULL REFERENCES users(id),
		image_path TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		reviewed_by INTEGER REFERENCES users(id),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		reviewed_at DATETIME
	)`)

	// User bans table
	db.Exec(`CREATE TABLE IF NOT EXISTS user_bans (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL UNIQUE REFERENCES users(id),
		banned_by INTEGER NOT NULL REFERENCES users(id),
		reason TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
}

func createUser(username, passwordHash string) (int64, error) {
	res, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, passwordHash)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func getUserByUsername(username string) (int64, string, error) {
	var id int64
	var hash string
	err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", username).Scan(&id, &hash)
	return id, hash, err
}

func getUserByID(id int64) (string, error) {
	var username string
	err := db.QueryRow("SELECT username FROM users WHERE id = ?", id).Scan(&username)
	return username, err
}

// ==================== User Profiles ====================

type UserProfile struct {
	UserID             int64  `json:"user_id"`
	Username           string `json:"username"`
	ProfilePicture     string `json:"profile_picture"`
	ContributionPoints int    `json:"contribution_points"`
	RatingCount        int    `json:"rating_count"`
}

func ensureUserProfile(userID int64) error {
	_, err := db.Exec(`INSERT OR IGNORE INTO user_profiles (user_id) VALUES (?)`, userID)
	return err
}

func getUserProfile(username string) (*UserProfile, error) {
	p := &UserProfile{}
	err := db.QueryRow(`
		SELECT u.id, u.username,
			COALESCE(p.profile_picture, ''),
			COALESCE(p.contribution_points, 0),
			(SELECT COUNT(*) FROM ratings WHERE user_id = u.id)
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.username = ?
	`, username).Scan(
		&p.UserID, &p.Username,
		&p.ProfilePicture,
		&p.ContributionPoints, &p.RatingCount,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func updateProfilePicture(userID int64, path string) error {
	if err := ensureUserProfile(userID); err != nil {
		return err
	}
	_, err := db.Exec(`UPDATE user_profiles SET profile_picture = ? WHERE user_id = ?`, path, userID)
	return err
}

func getProfilePicture(userID int64) (string, error) {
	var pic string
	err := db.QueryRow(`SELECT COALESCE(profile_picture, '') FROM user_profiles WHERE user_id = ?`, userID).Scan(&pic)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return pic, err
}

func getContributionPoints(userID int64) (int, error) {
	var pts int
	err := db.QueryRow(`SELECT COALESCE(contribution_points, 0) FROM user_profiles WHERE user_id = ?`, userID).Scan(&pts)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return pts, err
}

// ==================== Institutions (topics table) ====================

type InstitutionSummary struct {
	ID              int64   `json:"id"`
	Title           string  `json:"title"`
	InstitutionType string  `json:"institution_type"`
	Description     string  `json:"description"`
	CreatedBy       string  `json:"created_by"`
	AvgRating       float64 `json:"avg_rating"`
	NumRating       int     `json:"num_ratings"`
	CreatedAt       string  `json:"created_at"`
	CoverImage      string  `json:"cover_image"`
	EmailDomain     string  `json:"email_domain"`
	City            string  `json:"city"`
	Ownership       string  `json:"ownership"`
	TuitionMin      int     `json:"tuition_min"`
	TuitionMax      int     `json:"tuition_max"`
	Programs        string  `json:"programs"`
}

type InstitutionPhoto struct {
	ID        int64  `json:"id"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
}

type InstitutionDetail struct {
	ID              int64              `json:"id"`
	Title           string             `json:"title"`
	InstitutionType string             `json:"institution_type"`
	Description     string             `json:"description"`
	CreatedBy       string             `json:"created_by"`
	CreatedByID     int64              `json:"created_by_id"`
	AvgRating       float64            `json:"avg_rating"`
	NumRating       int                `json:"num_ratings"`
	CreatedAt       string             `json:"created_at"`
	CoverImage      string             `json:"cover_image"`
	EmailDomain     string             `json:"email_domain"`
	City            string             `json:"city"`
	Ownership       string             `json:"ownership"`
	TuitionMin      int                `json:"tuition_min"`
	TuitionMax      int                `json:"tuition_max"`
	Programs        string             `json:"programs"`
	Pros            string             `json:"pros"`
	Cons            string             `json:"cons"`
	Photos          []InstitutionPhoto `json:"photos"`
	Ratings         []RatingDetail     `json:"ratings"`
}

type RatingDetail struct {
	ID                   int64  `json:"id"`
	UserID               int64  `json:"user_id"`
	Username             string `json:"username"`
	Score                int    `json:"score"`
	ScoreAcademic        int    `json:"score_academic"`
	ScoreInfrastructure  int    `json:"score_infrastructure"`
	ScoreStudentLife     int    `json:"score_student_life"`
	ScoreCareer          int    `json:"score_career"`
	ScoreGuidance        int    `json:"score_guidance"`
	Comment              string `json:"comment"`
	CreatedAt            string `json:"created_at"`
	IsVerified           bool   `json:"is_verified"`
}

type InstitutionFilter struct {
	Query      string
	City       string
	Ownership  string
	Programs   string
	MinRating  float64
	MinTuition int
	MaxTuition int
}

func searchInstitutions(f InstitutionFilter) ([]InstitutionSummary, error) {
	q := `
		SELECT t.id, t.title, COALESCE(t.institution_type, ''), COALESCE(t.description, ''),
			u.username,
			COALESCE(AVG(r.score), 0), COUNT(r.id), t.created_at,
			COALESCE(t.cover_image, ''), COALESCE(t.email_domain, ''),
			COALESCE(t.city, ''), COALESCE(t.ownership, ''),
			COALESCE(t.tuition_min, 0), COALESCE(t.tuition_max, 0),
			COALESCE(t.programs, '')
		FROM topics t
		JOIN users u ON t.created_by = u.id
		LEFT JOIN ratings r ON r.topic_id = t.id
	`
	var conditions []string
	var args []interface{}

	if f.Query != "" {
		conditions = append(conditions, "t.title LIKE ?")
		args = append(args, "%"+f.Query+"%")
	}
	if f.City != "" {
		conditions = append(conditions, "LOWER(t.city) LIKE ?")
		args = append(args, "%"+strings.ToLower(f.City)+"%")
	}
	if f.Ownership != "" {
		conditions = append(conditions, "t.ownership = ?")
		args = append(args, f.Ownership)
	}
	if f.Programs != "" {
		conditions = append(conditions, "LOWER(t.programs) LIKE ?")
		args = append(args, "%"+strings.ToLower(f.Programs)+"%")
	}
	if f.MinTuition > 0 {
		conditions = append(conditions, "t.tuition_min >= ?")
		args = append(args, f.MinTuition)
	}
	if f.MaxTuition > 0 {
		conditions = append(conditions, "(t.tuition_max <= ? OR t.tuition_max = 0)")
		args = append(args, f.MaxTuition)
	}

	if len(conditions) > 0 {
		q += " WHERE " + strings.Join(conditions, " AND ")
	}
	q += " GROUP BY t.id"

	if f.MinRating > 0 {
		q += fmt.Sprintf(" HAVING AVG(r.score) >= %g", f.MinRating)
	}
	q += " ORDER BY t.created_at DESC"

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []InstitutionSummary
	for rows.Next() {
		var t InstitutionSummary
		if err := rows.Scan(&t.ID, &t.Title, &t.InstitutionType, &t.Description, &t.CreatedBy,
			&t.AvgRating, &t.NumRating, &t.CreatedAt, &t.CoverImage, &t.EmailDomain,
			&t.City, &t.Ownership, &t.TuitionMin, &t.TuitionMax, &t.Programs); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, nil
}

func createInstitution(title, instType, description, emailDomain, city, ownership string, tuitionMin, tuitionMax int, createdBy int64) (int64, error) {
	res, err := db.Exec("INSERT INTO topics (title, institution_type, description, email_domain, city, ownership, tuition_min, tuition_max, created_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		title, instType, description, emailDomain, city, ownership, tuitionMin, tuitionMax, createdBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateInstitutionMeta(instID int64, title, emailDomain, city, ownership, programs, pros, cons string, tuitionMin, tuitionMax int) error {
	_, err := db.Exec(`UPDATE topics SET title=?, email_domain=?, city=?, ownership=?, programs=?, pros=?, cons=?, tuition_min=?, tuition_max=? WHERE id=?`,
		title, emailDomain, city, ownership, programs, pros, cons, tuitionMin, tuitionMax, instID)
	return err
}

func getInstitutionDetail(instID int64) (*InstitutionDetail, error) {
	t := &InstitutionDetail{ID: instID}
	err := db.QueryRow(`
		SELECT t.title, COALESCE(t.institution_type, ''), COALESCE(t.description, ''),
			u.username, t.created_by, COALESCE(AVG(r.score), 0), COUNT(r.id), t.created_at,
			COALESCE(t.cover_image, ''), COALESCE(t.email_domain, ''),
			COALESCE(t.city, ''), COALESCE(t.ownership, ''),
			COALESCE(t.tuition_min, 0), COALESCE(t.tuition_max, 0),
			COALESCE(t.programs, ''), COALESCE(t.pros, ''), COALESCE(t.cons, '')
		FROM topics t
		JOIN users u ON t.created_by = u.id
		LEFT JOIN ratings r ON r.topic_id = t.id
		WHERE t.id = ?
		GROUP BY t.id
	`, instID).Scan(&t.Title, &t.InstitutionType, &t.Description, &t.CreatedBy, &t.CreatedByID,
		&t.AvgRating, &t.NumRating, &t.CreatedAt, &t.CoverImage, &t.EmailDomain,
		&t.City, &t.Ownership, &t.TuitionMin, &t.TuitionMax, &t.Programs, &t.Pros, &t.Cons)
	if err != nil {
		return nil, err
	}

	// Load photos
	t.Photos, _ = getInstitutionPhotos(instID)
	if t.Photos == nil {
		t.Photos = []InstitutionPhoto{}
	}

	rows, err := db.Query(`
		SELECT r.id, r.user_id, u.username, r.score,
			r.score_academic, r.score_infrastructure, r.score_student_life, r.score_career, r.score_guidance,
			r.comment, r.created_at,
			CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END AS is_verified
		FROM ratings r
		JOIN users u ON r.user_id = u.id
		LEFT JOIN user_verifications v ON v.user_id = r.user_id AND v.institution_id = r.topic_id
		WHERE r.topic_id = ?
		ORDER BY is_verified DESC, r.created_at DESC
	`, instID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r RatingDetail
		if err := rows.Scan(&r.ID, &r.UserID, &r.Username, &r.Score,
			&r.ScoreAcademic, &r.ScoreInfrastructure, &r.ScoreStudentLife, &r.ScoreCareer, &r.ScoreGuidance,
			&r.Comment, &r.CreatedAt, &r.IsVerified); err != nil {
			return nil, err
		}
		t.Ratings = append(t.Ratings, r)
	}

	return t, nil
}

func upsertRating(topicID, userID int64, academic, infrastructure, studentLife, career, guidance int, comment string) (bool, error) {
	// Check if rating already exists
	var existing int
	err := db.QueryRow("SELECT COUNT(*) FROM ratings WHERE topic_id = ? AND user_id = ?", topicID, userID).Scan(&existing)
	if err != nil {
		return false, err
	}
	isNew := existing == 0

	avg := math.Round(float64(academic+infrastructure+studentLife+career+guidance) / 5.0)
	score := int(avg)

	_, err = db.Exec(`
		INSERT INTO ratings (topic_id, user_id, score, score_academic, score_infrastructure, score_student_life, score_career, score_guidance, comment)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(topic_id, user_id) DO UPDATE SET
			score=excluded.score,
			score_academic=excluded.score_academic,
			score_infrastructure=excluded.score_infrastructure,
			score_student_life=excluded.score_student_life,
			score_career=excluded.score_career,
			score_guidance=excluded.score_guidance,
			comment=excluded.comment,
			created_at=CURRENT_TIMESTAMP
	`, topicID, userID, score, academic, infrastructure, studentLife, career, guidance, comment)
	return isNew, err
}

func updateTopicCover(topicID int64, path string) error {
	_, err := db.Exec(`UPDATE topics SET cover_image = ? WHERE id = ?`, path, topicID)
	return err
}

func getInstitutionPhotos(instID int64) ([]InstitutionPhoto, error) {
	rows, err := db.Query(`SELECT id, path, created_at FROM institution_photos WHERE institution_id = ? ORDER BY created_at ASC`, instID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var photos []InstitutionPhoto
	for rows.Next() {
		var p InstitutionPhoto
		if err := rows.Scan(&p.ID, &p.Path, &p.CreatedAt); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, nil
}

func addInstitutionPhoto(instID, userID int64, path string) (int64, error) {
	res, err := db.Exec(`INSERT INTO institution_photos (institution_id, uploaded_by, path) VALUES (?, ?, ?)`, instID, userID, path)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func deleteInstitutionPhoto(photoID int64) error {
	var path string
	db.QueryRow("SELECT path FROM institution_photos WHERE id = ?", photoID).Scan(&path)
	_, err := db.Exec("DELETE FROM institution_photos WHERE id = ?", photoID)
	if err == nil && path != "" {
		os.Remove(path)
	}
	return err
}

func getInstitutionCreator(instID int64) (int64, error) {
	var creatorID int64
	err := db.QueryRow("SELECT created_by FROM topics WHERE id = ?", instID).Scan(&creatorID)
	return creatorID, err
}

// ==================== Discussion Comments ====================

type DiscussionComment struct {
	ID                 int64  `json:"id"`
	UserID             int64  `json:"user_id"`
	Username           string `json:"username"`
	ParentID           *int64 `json:"parent_id"`
	Content            string `json:"content"`
	CreatedAt          string `json:"created_at"`
	ContributionPoints int    `json:"contribution_points"`
}

func getInstitutionDiscussion(instID int64) ([]DiscussionComment, error) {
	rows, err := db.Query(`
		SELECT dc.id, dc.user_id, u.username, dc.parent_id, dc.content, dc.created_at,
			COALESCE(p.contribution_points, 0) AS contribution_points
		FROM discussion_comments dc
		JOIN users u ON dc.user_id = u.id
		LEFT JOIN user_profiles p ON p.user_id = dc.user_id
		WHERE dc.institution_id = ?
		ORDER BY contribution_points DESC, dc.created_at ASC
	`, instID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []DiscussionComment
	for rows.Next() {
		var c DiscussionComment
		if err := rows.Scan(&c.ID, &c.UserID, &c.Username, &c.ParentID, &c.Content, &c.CreatedAt, &c.ContributionPoints); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	if comments == nil {
		comments = []DiscussionComment{}
	}
	return comments, nil
}

func createDiscussionComment(instID, userID int64, parentID *int64, content string) (int64, error) {
	res, err := db.Exec(`
		INSERT INTO discussion_comments (institution_id, user_id, parent_id, content)
		VALUES (?, ?, ?, ?)
	`, instID, userID, parentID, content)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func getCommentOwner(commentID int64) (int64, error) {
	var userID int64
	err := db.QueryRow("SELECT user_id FROM discussion_comments WHERE id = ?", commentID).Scan(&userID)
	return userID, err
}

// ==================== Comment Deletion ====================

func deleteDiscussionComment(commentID, userID int64) error {
	// Verify ownership
	var ownerID int64
	err := db.QueryRow("SELECT user_id FROM discussion_comments WHERE id = ?", commentID).Scan(&ownerID)
	if err != nil {
		return err
	}
	if ownerID != userID {
		return fmt.Errorf("not the comment owner")
	}
	return deleteCommentAndEvents(commentID)
}

func modDeleteDiscussionComment(commentID int64) error {
	// No ownership check — moderator can delete any comment
	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM discussion_comments WHERE id = ?", commentID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("comment not found")
	}
	return deleteCommentAndEvents(commentID)
}

func deleteCommentAndEvents(commentID int64) error {
	// Collect all affected user IDs before deleting
	affectedUsers := map[int64]bool{}

	// The comment author
	var authorID int64
	db.QueryRow("SELECT user_id FROM discussion_comments WHERE id = ?", commentID).Scan(&authorID)
	affectedUsers[authorID] = true

	// Users who have contribution_events referencing this comment
	rows, err := db.Query("SELECT DISTINCT user_id FROM contribution_events WHERE reference_id = ?", commentID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var uid int64
			rows.Scan(&uid)
			affectedUsers[uid] = true
		}
	}

	// Also collect users affected by child comments
	childRows, err := db.Query("SELECT id, user_id FROM discussion_comments WHERE parent_id = ?", commentID)
	if err == nil {
		defer childRows.Close()
		for childRows.Next() {
			var childID, childUserID int64
			childRows.Scan(&childID, &childUserID)
			affectedUsers[childUserID] = true
			// Events referencing child comments
			evRows, err2 := db.Query("SELECT DISTINCT user_id FROM contribution_events WHERE reference_id = ?", childID)
			if err2 == nil {
				for evRows.Next() {
					var uid int64
					evRows.Scan(&uid)
					affectedUsers[uid] = true
				}
				evRows.Close()
			}
		}
	}

	// Delete contribution events for child comments
	db.Exec(`DELETE FROM contribution_events WHERE reference_id IN (SELECT id FROM discussion_comments WHERE parent_id = ?)`, commentID)

	// Delete child comments
	db.Exec("DELETE FROM discussion_comments WHERE parent_id = ?", commentID)

	// Delete contribution events for this comment
	db.Exec("DELETE FROM contribution_events WHERE reference_id = ?", commentID)

	// Delete the comment itself
	db.Exec("DELETE FROM discussion_comments WHERE id = ?", commentID)

	// Recalculate points for all affected users
	for uid := range affectedUsers {
		recalculateContributionPoints(uid)
	}

	return nil
}

// ==================== Moderator ====================

func getInstitutionModerator(instID int64) (int64, error) {
	var userID int64
	err := db.QueryRow(`
		SELECT u.id
		FROM users u
		WHERE u.id IN (
			SELECT user_id FROM ratings WHERE topic_id = ?
			UNION
			SELECT user_id FROM discussion_comments WHERE institution_id = ?
		)
		ORDER BY (
			(SELECT COUNT(*) FROM ratings WHERE user_id = u.id AND topic_id = ?) * 25 +
			(SELECT COUNT(*) FROM discussion_comments WHERE user_id = u.id AND institution_id = ?) * 12
		) DESC
		LIMIT 1
	`, instID, instID, instID, instID).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return userID, err
}

// ==================== Muting ====================

func muteUser(instID, userID, mutedBy int64, minutes int) error {
	_, err := db.Exec(`
		INSERT INTO user_mutes (institution_id, user_id, muted_until, muted_by)
		VALUES (?, ?, datetime('now', ?), ?)
	`, instID, userID, fmt.Sprintf("+%d minutes", minutes), mutedBy)
	return err
}

func isUserMuted(instID, userID int64) (bool, string, error) {
	var mutedUntil string
	err := db.QueryRow(`
		SELECT muted_until FROM user_mutes
		WHERE institution_id = ? AND user_id = ? AND muted_until > datetime('now')
		ORDER BY muted_until DESC LIMIT 1
	`, instID, userID).Scan(&mutedUntil)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, mutedUntil, nil
}

// ==================== Contribution Points ====================

func awardContributionPoints(userID int64, eventType string, points int, referenceID int64) error {
	_, err := db.Exec(`INSERT INTO contribution_events (user_id, event_type, points, reference_id) VALUES (?, ?, ?, ?)`,
		userID, eventType, points, referenceID)
	if err != nil {
		return err
	}
	return recalculateContributionPoints(userID)
}

func recalculateContributionPoints(userID int64) error {
	if err := ensureUserProfile(userID); err != nil {
		return err
	}
	_, err := db.Exec(`
		UPDATE user_profiles SET contribution_points = (
			SELECT COALESCE(SUM(points), 0) FROM contribution_events WHERE user_id = ?
		) WHERE user_id = ?
	`, userID, userID)
	return err
}

// ==================== User Institutions ====================

func getUserInstitutions(userID int64) ([]InstitutionSummary, error) {
	rows, err := db.Query(`
		SELECT DISTINCT t.id, t.title, COALESCE(t.institution_type, ''), COALESCE(t.description, ''),
			u.username,
			COALESCE((SELECT AVG(score) FROM ratings WHERE topic_id = t.id), 0),
			(SELECT COUNT(*) FROM ratings WHERE topic_id = t.id),
			t.created_at,
			COALESCE(t.cover_image, ''), COALESCE(t.email_domain, ''),
			COALESCE(t.city, ''), COALESCE(t.ownership, ''),
			COALESCE(t.tuition_min, 0), COALESCE(t.tuition_max, 0),
			COALESCE(t.programs, '')
		FROM topics t
		JOIN users u ON t.created_by = u.id
		JOIN ratings r ON r.topic_id = t.id AND r.user_id = ?
		ORDER BY t.title
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []InstitutionSummary
	for rows.Next() {
		var t InstitutionSummary
		if err := rows.Scan(&t.ID, &t.Title, &t.InstitutionType, &t.Description, &t.CreatedBy,
			&t.AvgRating, &t.NumRating, &t.CreatedAt, &t.CoverImage, &t.EmailDomain,
			&t.City, &t.Ownership, &t.TuitionMin, &t.TuitionMax, &t.Programs); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	if items == nil {
		items = []InstitutionSummary{}
	}
	return items, nil
}

// ==================== Admin ====================

type AdminUser struct {
	ID                 int64  `json:"id"`
	Username           string `json:"username"`
	ContributionPoints int    `json:"contribution_points"`
	RatingCount        int    `json:"rating_count"`
	CreatedAt          string `json:"created_at"`
	IsBanned           bool   `json:"is_banned"`
}

func getAllUsersWithScores() ([]AdminUser, error) {
	rows, err := db.Query(`
		SELECT u.id, u.username, COALESCE(p.contribution_points, 0),
			(SELECT COUNT(*) FROM ratings WHERE user_id = u.id),
			u.created_at,
			CASE WHEN b.id IS NOT NULL THEN 1 ELSE 0 END AS is_banned
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		LEFT JOIN user_bans b ON b.user_id = u.id
		ORDER BY u.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []AdminUser
	for rows.Next() {
		var u AdminUser
		var isBanned int
		if err := rows.Scan(&u.ID, &u.Username, &u.ContributionPoints, &u.RatingCount, &u.CreatedAt, &isBanned); err != nil {
			return nil, err
		}
		u.IsBanned = isBanned == 1
		users = append(users, u)
	}
	return users, nil
}

// ==================== Education History ====================

type EducationEntry struct {
	ID              int64  `json:"id"`
	UserID          int64  `json:"user_id"`
	InstitutionName string `json:"institution_name"`
	InstitutionID   *int64 `json:"institution_id"`
	StartDate       string `json:"start_date"`
	EndDate         string `json:"end_date"`
	Role            string `json:"role"`
	CreatedAt       string `json:"created_at"`
}

func matchInstitutionByName(name string) *int64 {
	var id int64
	err := db.QueryRow("SELECT id FROM topics WHERE LOWER(title) = LOWER(?)", name).Scan(&id)
	if err != nil {
		return nil
	}
	return &id
}

func addEducationEntry(userID int64, name, startDate, endDate, role string) (int64, error) {
	instID := matchInstitutionByName(name)
	res, err := db.Exec(`
		INSERT INTO education_history (user_id, institution_name, institution_id, start_date, end_date, role)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, name, instID, startDate, endDate, role)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func removeEducationEntry(entryID int64, userID int64) error {
	res, err := db.Exec("DELETE FROM education_history WHERE id = ? AND user_id = ?", entryID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func getUserEducation(userID int64) ([]EducationEntry, error) {
	rows, err := db.Query(`
		SELECT id, user_id, institution_name, institution_id, start_date, end_date, role, created_at
		FROM education_history
		WHERE user_id = ?
		ORDER BY start_date DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []EducationEntry
	for rows.Next() {
		var e EducationEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.InstitutionName, &e.InstitutionID, &e.StartDate, &e.EndDate, &e.Role, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []EducationEntry{}
	}
	return entries, nil
}

// ==================== Verification Requests ====================

type VerificationRequest struct {
	ID              int64   `json:"id"`
	InstitutionID   int64   `json:"institution_id"`
	InstitutionName string  `json:"institution_name"`
	UserID          int64   `json:"user_id"`
	Username        string  `json:"username"`
	ImagePath       string  `json:"image_path"`
	Status          string  `json:"status"`
	ReviewedBy      *int64  `json:"reviewed_by"`
	CreatedAt       string  `json:"created_at"`
	ReviewedAt      *string `json:"reviewed_at"`
}

func createVerificationRequest(instID, userID int64, imagePath string) (int64, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM verification_requests WHERE user_id=? AND institution_id=? AND status='pending'", userID, instID).Scan(&count)
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return 0, fmt.Errorf("you already have a pending verification request for this institution")
	}
	res, err := db.Exec("INSERT INTO verification_requests (institution_id, user_id, image_path) VALUES (?, ?, ?)", instID, userID, imagePath)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func getVerificationRequestsByInstitution(instID int64) ([]VerificationRequest, error) {
	rows, err := db.Query(`
		SELECT vr.id, vr.institution_id, t.title, vr.user_id, u.username, vr.image_path, vr.status, vr.reviewed_by, vr.created_at, vr.reviewed_at
		FROM verification_requests vr
		JOIN users u ON vr.user_id = u.id
		JOIN topics t ON vr.institution_id = t.id
		WHERE vr.institution_id=? AND vr.status='pending'
		ORDER BY vr.created_at ASC
	`, instID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []VerificationRequest
	for rows.Next() {
		var vr VerificationRequest
		if err := rows.Scan(&vr.ID, &vr.InstitutionID, &vr.InstitutionName, &vr.UserID, &vr.Username, &vr.ImagePath, &vr.Status, &vr.ReviewedBy, &vr.CreatedAt, &vr.ReviewedAt); err != nil {
			return nil, err
		}
		result = append(result, vr)
	}
	if result == nil {
		result = []VerificationRequest{}
	}
	return result, nil
}

func getAllPendingVerificationRequests() ([]VerificationRequest, error) {
	rows, err := db.Query(`
		SELECT vr.id, vr.institution_id, t.title, vr.user_id, u.username, vr.image_path, vr.status, vr.reviewed_by, vr.created_at, vr.reviewed_at
		FROM verification_requests vr
		JOIN users u ON vr.user_id = u.id
		JOIN topics t ON vr.institution_id = t.id
		WHERE vr.status='pending'
		ORDER BY vr.created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []VerificationRequest
	for rows.Next() {
		var vr VerificationRequest
		if err := rows.Scan(&vr.ID, &vr.InstitutionID, &vr.InstitutionName, &vr.UserID, &vr.Username, &vr.ImagePath, &vr.Status, &vr.ReviewedBy, &vr.CreatedAt, &vr.ReviewedAt); err != nil {
			return nil, err
		}
		result = append(result, vr)
	}
	if result == nil {
		result = []VerificationRequest{}
	}
	return result, nil
}

func getVerificationRequest(requestID int64) (*VerificationRequest, error) {
	vr := &VerificationRequest{}
	err := db.QueryRow(`
		SELECT vr.id, vr.institution_id, t.title, vr.user_id, u.username, vr.image_path, vr.status, vr.reviewed_by, vr.created_at, vr.reviewed_at
		FROM verification_requests vr
		JOIN users u ON vr.user_id = u.id
		JOIN topics t ON vr.institution_id = t.id
		WHERE vr.id=?
	`, requestID).Scan(&vr.ID, &vr.InstitutionID, &vr.InstitutionName, &vr.UserID, &vr.Username, &vr.ImagePath, &vr.Status, &vr.ReviewedBy, &vr.CreatedAt, &vr.ReviewedAt)
	if err != nil {
		return nil, err
	}
	return vr, nil
}

func reviewVerificationRequest(requestID int64, status string, reviewerID int64) error {
	// First get the request to find userID and instID if approving
	vr, err := getVerificationRequest(requestID)
	if err != nil {
		return err
	}
	_, err = db.Exec("UPDATE verification_requests SET status=?, reviewed_by=?, reviewed_at=CURRENT_TIMESTAMP WHERE id=?", status, reviewerID, requestID)
	if err != nil {
		return err
	}
	if status == "approved" {
		return verifyUserForInstitution(vr.UserID, vr.InstitutionID, "photo-proof")
	}
	return nil
}

// ==================== Bans ====================

func banUser(userID, bannedBy int64, reason string) error {
	_, err := db.Exec("INSERT OR REPLACE INTO user_bans (user_id, banned_by, reason) VALUES (?, ?, ?)", userID, bannedBy, reason)
	return err
}

func unbanUser(userID int64) error {
	_, err := db.Exec("DELETE FROM user_bans WHERE user_id = ?", userID)
	return err
}

func isUserBanned(userID int64) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM user_bans WHERE user_id = ?", userID).Scan(&count)
	return count > 0, err
}

func getAllBannedUsers() ([]map[string]interface{}, error) {
	rows, err := db.Query(`
		SELECT b.user_id, u.username, b.reason, b.created_at, banner.username as banned_by_name
		FROM user_bans b
		JOIN users u ON b.user_id = u.id
		JOIN users banner ON b.banned_by = banner.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]interface{}
	for rows.Next() {
		var userID int64
		var username, reason, createdAt, bannedByName string
		if err := rows.Scan(&userID, &username, &reason, &createdAt, &bannedByName); err != nil {
			return nil, err
		}
		result = append(result, map[string]interface{}{
			"user_id":        userID,
			"username":       username,
			"reason":         reason,
			"created_at":     createdAt,
			"banned_by_name": bannedByName,
		})
	}
	if result == nil {
		result = []map[string]interface{}{}
	}
	return result, nil
}

func getUserActivity(userID int64) (map[string]interface{}, error) {
	// Ratings
	ratingRows, err := db.Query(`
		SELECT r.id, t.title, r.score, r.comment, r.created_at
		FROM ratings r JOIN topics t ON r.topic_id = t.id
		WHERE r.user_id=?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer ratingRows.Close()
	var ratings []map[string]interface{}
	for ratingRows.Next() {
		var id int64
		var title, comment, createdAt string
		var score int
		if err := ratingRows.Scan(&id, &title, &score, &comment, &createdAt); err != nil {
			return nil, err
		}
		ratings = append(ratings, map[string]interface{}{"id": id, "title": title, "score": score, "comment": comment, "created_at": createdAt})
	}
	if ratings == nil {
		ratings = []map[string]interface{}{}
	}

	// Comments
	commentRows, err := db.Query(`
		SELECT dc.id, t.title, dc.content, dc.created_at
		FROM discussion_comments dc JOIN topics t ON dc.institution_id = t.id
		WHERE dc.user_id=?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer commentRows.Close()
	var comments []map[string]interface{}
	for commentRows.Next() {
		var id int64
		var title, content, createdAt string
		if err := commentRows.Scan(&id, &title, &content, &createdAt); err != nil {
			return nil, err
		}
		comments = append(comments, map[string]interface{}{"id": id, "title": title, "content": content, "created_at": createdAt})
	}
	if comments == nil {
		comments = []map[string]interface{}{}
	}

	// Events
	eventRows, err := db.Query(`
		SELECT event_type, points, created_at FROM contribution_events
		WHERE user_id=? ORDER BY created_at DESC LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer eventRows.Close()
	var events []map[string]interface{}
	for eventRows.Next() {
		var eventType, createdAt string
		var points int
		if err := eventRows.Scan(&eventType, &points, &createdAt); err != nil {
			return nil, err
		}
		events = append(events, map[string]interface{}{"event_type": eventType, "points": points, "created_at": createdAt})
	}
	if events == nil {
		events = []map[string]interface{}{}
	}

	return map[string]interface{}{"ratings": ratings, "comments": comments, "events": events}, nil
}

// ==================== Verification ====================

func getInstitutionEmailDomain(instID int64) (string, error) {
	var domain string
	err := db.QueryRow("SELECT COALESCE(email_domain, '') FROM topics WHERE id = ?", instID).Scan(&domain)
	return domain, err
}

func verifyUserForInstitution(userID, instID int64, email string) error {
	_, err := db.Exec(`INSERT INTO user_verifications (user_id, institution_id, verified_email) VALUES (?, ?, ?)
		ON CONFLICT(user_id, institution_id) DO UPDATE SET verified_email=excluded.verified_email`, userID, instID, email)
	return err
}

func isUserVerifiedForInstitution(userID, instID int64) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM user_verifications WHERE user_id = ? AND institution_id = ?", userID, instID).Scan(&count)
	return count > 0, err
}

// ==================== Rank Tiers & Leaderboards ====================

func getRankTier(points int) string {
	switch {
	case points >= 175:
		return "Legend"
	case points >= 150:
		return "Emerald"
	case points >= 125:
		return "Diamond"
	case points >= 100:
		return "Gold"
	case points >= 75:
		return "Silver"
	case points >= 50:
		return "Bronze"
	default:
		return "Iron"
	}
}

type SchoolUserRank struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Points   int    `json:"points"`
	Rank     string `json:"rank"`
}

func getSchoolLeaderboard(instID int64) ([]SchoolUserRank, error) {
	rows, err := db.Query(`
		SELECT u.id, u.username,
			(SELECT COUNT(*) FROM ratings WHERE user_id = u.id AND topic_id = ?) * 25 +
			(SELECT COUNT(*) FROM discussion_comments WHERE user_id = u.id AND institution_id = ?) * 12
			AS total_points
		FROM users u
		WHERE u.id IN (
			SELECT user_id FROM ratings WHERE topic_id = ?
			UNION
			SELECT user_id FROM discussion_comments WHERE institution_id = ?
		)
		ORDER BY total_points DESC
	`, instID, instID, instID, instID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SchoolUserRank
	for rows.Next() {
		var r SchoolUserRank
		if err := rows.Scan(&r.UserID, &r.Username, &r.Points); err != nil {
			return nil, err
		}
		r.Rank = getRankTier(r.Points)
		result = append(result, r)
	}
	if result == nil {
		result = []SchoolUserRank{}
	}
	return result, nil
}

type SchoolRanking struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	InstitutionType string `json:"institution_type"`
	TotalPoints     int    `json:"total_points"`
}

func getTopSchools(institutionType string) ([]SchoolRanking, error) {
	query := `
		SELECT t.id, t.title, COALESCE(t.institution_type, ''),
			(SELECT COUNT(*) FROM ratings WHERE topic_id = t.id) * 25 +
			(SELECT COUNT(*) FROM discussion_comments WHERE institution_id = t.id) * 12
			AS total_points
		FROM topics t
	`
	var rows *sql.Rows
	var err error
	if institutionType != "" {
		query += " WHERE t.institution_type = ? ORDER BY total_points DESC LIMIT 10"
		rows, err = db.Query(query, institutionType)
	} else {
		query += " ORDER BY total_points DESC LIMIT 10"
		rows, err = db.Query(query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SchoolRanking
	for rows.Next() {
		var r SchoolRanking
		if err := rows.Scan(&r.ID, &r.Title, &r.InstitutionType, &r.TotalPoints); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	if result == nil {
		result = []SchoolRanking{}
	}
	return result, nil
}

func setUserContributionPoints(userID int64, points int) error {
	if err := ensureUserProfile(userID); err != nil {
		return err
	}
	// Clear existing events and create a manual adjustment
	_, err := db.Exec(`DELETE FROM contribution_events WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}
	if points != 0 {
		_, err = db.Exec(`INSERT INTO contribution_events (user_id, event_type, points, reference_id) VALUES (?, 'admin_adjustment', ?, 0)`, userID, points)
		if err != nil {
			return err
		}
	}
	_, err = db.Exec(`UPDATE user_profiles SET contribution_points = ? WHERE user_id = ?`, points, userID)
	return err
}
