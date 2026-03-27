package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 || len(req.Password) < 4 {
		jsonResponse(w, 400, map[string]string{"error": "username must be 3+ chars, password 4+ chars"})
		return
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	userID, err := createUser(req.Username, hash)
	if err != nil {
		jsonResponse(w, 409, map[string]string{"error": "username already taken"})
		return
	}

	token := createSession(userID, req.Username)
	setSessionCookie(w, token)
	jsonResponse(w, 201, map[string]interface{}{"id": userID, "username": req.Username})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}

	userID, hash, err := getUserByUsername(req.Username)
	if err != nil || !checkPassword(hash, req.Password) {
		jsonResponse(w, 401, map[string]string{"error": "invalid credentials"})
		return
	}

	token := createSession(userID, req.Username)
	setSessionCookie(w, token)
	jsonResponse(w, 200, map[string]interface{}{"id": userID, "username": req.Username})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session"); err == nil {
		deleteSession(cookie.Value)
	}
	clearSessionCookie(w)
	jsonResponse(w, 200, map[string]string{"message": "logged out"})
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	sd, ok := getSessionFromRequest(r)
	if !ok {
		jsonResponse(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	username, err := getUserByID(sd.UserID)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	pic, _ := getProfilePicture(sd.UserID)
	points, _ := getContributionPoints(sd.UserID)
	banned, _ := isUserBanned(sd.UserID)

	jsonResponse(w, 200, map[string]interface{}{
		"id":                  sd.UserID,
		"username":            username,
		"is_admin":            sd.IsAdmin,
		"contribution_points": points,
		"profile_picture":     pic,
		"is_banned":           banned,
	})
}

// ==================== Institution Handlers ====================

func handleListInstitutions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	minRating, _ := strconv.ParseFloat(q.Get("min_rating"), 64)
	minTuition, _ := strconv.Atoi(q.Get("min_tuition"))
	maxTuition, _ := strconv.Atoi(q.Get("max_tuition"))
	filter := InstitutionFilter{
		Query:      q.Get("q"),
		City:       q.Get("city"),
		Ownership:  q.Get("ownership"),
		Programs:   q.Get("programs"),
		MinRating:  minRating,
		MinTuition: minTuition,
		MaxTuition: maxTuition,
	}
	items, err := searchInstitutions(filter)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	if items == nil {
		items = []InstitutionSummary{}
	}
	jsonResponse(w, 200, items)
}

func handleCreateInstitution(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)

	if banned, _ := isUserBanned(userID); banned {
		jsonResponse(w, 403, map[string]string{"error": "your account has been banned"})
		return
	}

	// Check contribution points threshold
	points, _ := getContributionPoints(userID)
	if points < institutionCreateThreshold {
		jsonResponse(w, 403, map[string]string{"error": fmt.Sprintf("need %d contribution points to add institutions (you have %d)", institutionCreateThreshold, points)})
		return
	}

	var req struct {
		Title           string `json:"title"`
		InstitutionType string `json:"institution_type"`
		Description     string `json:"description"`
		EmailDomain     string `json:"email_domain"`
		City            string `json:"city"`
		Ownership       string `json:"ownership"`
		TuitionMin      int    `json:"tuition_min"`
		TuitionMax      int    `json:"tuition_max"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		jsonResponse(w, 400, map[string]string{"error": "title is required"})
		return
	}

	id, err := createInstitution(req.Title, strings.TrimSpace(req.InstitutionType), strings.TrimSpace(req.Description),
		strings.TrimSpace(req.EmailDomain), strings.TrimSpace(req.City), strings.TrimSpace(req.Ownership),
		req.TuitionMin, req.TuitionMax, userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			jsonResponse(w, 409, map[string]string{"error": "institution already exists"})
		} else {
			jsonResponse(w, 500, map[string]string{"error": "server error"})
		}
		return
	}

	jsonResponse(w, 201, map[string]interface{}{"id": id, "title": req.Title})
}

func handleGetInstitution(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}

	viewerID := int64(0)
	isVerified := false
	if sd, ok := getSessionFromRequest(r); ok {
		viewerID = sd.UserID
		isVerified, _ = isUserVerifiedForInstitution(sd.UserID, id)
	}

	inst, err := getInstitutionDetail(id, viewerID)
	if err != nil {
		jsonResponse(w, 404, map[string]string{"error": "institution not found"})
		return
	}

	if inst.Ratings == nil {
		inst.Ratings = []RatingDetail{}
	}

	type InstResponse struct {
		*InstitutionDetail
		IsCurrentUserVerified bool `json:"is_current_user_verified"`
	}
	jsonResponse(w, 200, InstResponse{inst, isVerified})
}

func handleRateInstitution(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	idStr := mux.Vars(r)["id"]
	instID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}

	if banned, _ := isUserBanned(userID); banned {
		jsonResponse(w, 403, map[string]string{"error": "your account has been banned"})
		return
	}

	verified, err := isUserVerifiedForInstitution(userID, instID)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	if !verified {
		jsonResponse(w, 403, map[string]string{"error": "you must be verified for this institution to leave a rating"})
		return
	}

	var req struct {
		ScoreAcademic       int    `json:"score_academic"`
		ScoreInfrastructure int    `json:"score_infrastructure"`
		ScoreStudentLife    int    `json:"score_student_life"`
		ScoreCareer         int    `json:"score_career"`
		ScoreGuidance       int    `json:"score_guidance"`
		Comment             string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}

	scores := []int{req.ScoreAcademic, req.ScoreInfrastructure, req.ScoreStudentLife, req.ScoreCareer, req.ScoreGuidance}
	for _, s := range scores {
		if s < 1 || s > 5 {
			jsonResponse(w, 400, map[string]string{"error": "all category scores must be 1-5"})
			return
		}
	}

	isNew, err := upsertRating(instID, userID, req.ScoreAcademic, req.ScoreInfrastructure, req.ScoreStudentLife, req.ScoreCareer, req.ScoreGuidance, strings.TrimSpace(req.Comment))
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	// Award +25 points for new ratings only
	if isNew {
		awardContributionPoints(userID, "rate_institution", 25, instID)
	}

	jsonResponse(w, 200, map[string]string{"message": "rating saved"})
}

// ==================== Verify Email Handler ====================

func handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	idStr := mux.Vars(r)["id"]
	instID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		jsonResponse(w, 400, map[string]string{"error": "email is required"})
		return
	}

	domain, err := getInstitutionEmailDomain(instID)
	if err != nil {
		jsonResponse(w, 404, map[string]string{"error": "institution not found"})
		return
	}
	if domain == "" {
		jsonResponse(w, 400, map[string]string{"error": "this institution does not have a verification domain"})
		return
	}

	// Check email ends with the domain
	if !strings.HasSuffix(strings.ToLower(req.Email), strings.ToLower(domain)) {
		jsonResponse(w, 400, map[string]string{"error": "email does not match institution domain " + domain})
		return
	}

	if err := verifyUserForInstitution(userID, instID, req.Email); err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	jsonResponse(w, 200, map[string]string{"message": "verification successful"})
}

// ==================== Discussion Handlers ====================

func handleGetDiscussion(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	instID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}

	viewerID := int64(0)
	if sd, ok := getSessionFromRequest(r); ok {
		viewerID = sd.UserID
	}

	comments, err := getInstitutionDiscussion(instID, viewerID)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	jsonResponse(w, 200, comments)
}

func handlePostComment(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	idStr := mux.Vars(r)["id"]
	instID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}

	if banned, _ := isUserBanned(userID); banned {
		jsonResponse(w, 403, map[string]string{"error": "your account has been banned"})
		return
	}

	// Check if user is muted
	muted, mutedUntil, _ := isUserMuted(instID, userID)
	if muted {
		jsonResponse(w, 403, map[string]string{"error": "You are muted until " + mutedUntil})
		return
	}

	var req struct {
		Content  string `json:"content"`
		ParentID *int64 `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}

	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		jsonResponse(w, 400, map[string]string{"error": "content is required"})
		return
	}

	commentID, err := createDiscussionComment(instID, userID, req.ParentID, req.Content)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	// Award +12 points for posting a comment
	awardContributionPoints(userID, "post_comment", 12, commentID)

	// If this is a reply, award +12 points to the parent comment author
	if req.ParentID != nil {
		parentOwner, err := getCommentOwner(*req.ParentID)
		if err == nil && parentOwner != userID {
			awardContributionPoints(parentOwner, "received_reply", 12, commentID)
		}
	}

	jsonResponse(w, 201, map[string]interface{}{"id": commentID, "message": "comment posted"})
}

// ==================== Profile Handlers ====================

func handleGetProfile(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	profile, err := getUserProfile(username)
	if err != nil {
		jsonResponse(w, 404, map[string]string{"error": "user not found"})
		return
	}

	// Get institution affiliations (institutions they've rated)
	institutions, err := getUserInstitutions(profile.UserID)
	if err != nil {
		institutions = []InstitutionSummary{}
	}

	education, err := getUserEducation(profile.UserID)
	if err != nil {
		education = []EducationEntry{}
	}

	jsonResponse(w, 200, map[string]interface{}{
		"user_id":             profile.UserID,
		"username":            profile.Username,
		"profile_picture":     profile.ProfilePicture,
		"contribution_points": profile.ContributionPoints,
		"rating_count":        profile.RatingCount,
		"institutions":        institutions,
		"education":           education,
	})
}

func handleUploadProfilePicture(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "file too large (max 5MB)"})
		return
	}

	file, _, err := r.FormFile("picture")
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "no file provided"})
		return
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	contentType := http.DetectContentType(buf[:n])
	if !allowedImageTypes[contentType] {
		jsonResponse(w, 400, map[string]string{"error": "invalid image type"})
		return
	}

	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}
	filename := hex.EncodeToString(randBytes) + ext
	savePath := filepath.Join(uploadsDir, "profiles", filename)

	out, err := os.Create(savePath)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	defer out.Close()

	out.Write(buf[:n])
	io.Copy(out, file)

	oldPic, _ := getProfilePicture(userID)
	if oldPic != "" {
		os.Remove(oldPic)
	}

	if err := updateProfilePicture(userID, savePath); err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	jsonResponse(w, 200, map[string]string{"path": "/" + savePath})
}

func handleDeleteProfilePicture(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)

	oldPic, _ := getProfilePicture(userID)
	if oldPic != "" {
		os.Remove(oldPic)
	}

	if err := updateProfilePicture(userID, ""); err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	jsonResponse(w, 200, map[string]string{"message": "picture removed"})
}

// ==================== Institution Cover ====================

func handleUploadInstitutionCover(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	idStr := mux.Vars(r)["id"]
	instID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}

	// Check contribution points threshold OR moderator status
	points, _ := getContributionPoints(userID)
	modID, _ := getInstitutionModerator(instID)
	sd, _ := getSessionFromRequest(r)
	if points < institutionCreateThreshold && modID != userID && !sd.IsAdmin {
		jsonResponse(w, 403, map[string]string{"error": fmt.Sprintf("need %d+ contribution points to upload covers", institutionCreateThreshold)})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "file too large (max 5MB)"})
		return
	}

	file, _, err := r.FormFile("cover")
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "no file provided"})
		return
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	contentType := http.DetectContentType(buf[:n])
	if !allowedImageTypes[contentType] {
		jsonResponse(w, 400, map[string]string{"error": "invalid image type"})
		return
	}

	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}
	filename := hex.EncodeToString(randBytes) + ext
	savePath := filepath.Join(uploadsDir, "topics", filename)

	out, err := os.Create(savePath)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	defer out.Close()

	out.Write(buf[:n])
	io.Copy(out, file)

	if err := updateTopicCover(instID, savePath); err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	jsonResponse(w, 200, map[string]string{"path": "/" + savePath})
}

// ==================== Education Handlers ====================

func handleAddEducation(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)

	var req struct {
		InstitutionName string `json:"institution_name"`
		StartDate       string `json:"start_date"`
		EndDate         string `json:"end_date"`
		Role            string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}

	req.InstitutionName = strings.TrimSpace(req.InstitutionName)
	req.StartDate = strings.TrimSpace(req.StartDate)
	req.Role = strings.TrimSpace(req.Role)
	req.EndDate = strings.TrimSpace(req.EndDate)

	if req.InstitutionName == "" || req.StartDate == "" || req.Role == "" {
		jsonResponse(w, 400, map[string]string{"error": "institution_name, start_date, and role are required"})
		return
	}

	id, err := addEducationEntry(userID, req.InstitutionName, req.StartDate, req.EndDate, req.Role)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	jsonResponse(w, 201, map[string]interface{}{"id": id, "message": "education entry added"})
}

func handleDeleteEducation(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	idStr := mux.Vars(r)["id"]
	entryID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid entry id"})
		return
	}

	err = removeEducationEntry(entryID, userID)
	if err == sql.ErrNoRows {
		jsonResponse(w, 404, map[string]string{"error": "entry not found or not owned by you"})
		return
	}
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	jsonResponse(w, 200, map[string]string{"message": "education entry removed"})
}

// ==================== Admin Handlers ====================

func handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := getAllUsersWithScores()
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	if users == nil {
		users = []AdminUser{}
	}
	jsonResponse(w, 200, users)
}

// ==================== Leaderboard & Rankings Handlers ====================

func handleSchoolLeaderboard(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	instID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}

	leaderboard, err := getSchoolLeaderboard(instID)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	modID, _ := getInstitutionModerator(instID)
	jsonResponse(w, 200, map[string]interface{}{
		"entries":      leaderboard,
		"moderator_id": modID,
	})
}

func handleTopSchools(w http.ResponseWriter, r *http.Request) {
	instType := r.URL.Query().Get("type")
	rankings, err := getTopSchools(instType)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	jsonResponse(w, 200, rankings)
}

func handleAdminSetPoints(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid user id"})
		return
	}

	var req struct {
		Points int `json:"points"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}

	if err := setUserContributionPoints(userID, req.Points); err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	jsonResponse(w, 200, map[string]string{"message": "points updated"})
}

// ==================== Comment Deletion Handlers ====================

func handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	vars := mux.Vars(r)
	commentID, err := strconv.ParseInt(vars["commentId"], 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid comment id"})
		return
	}

	if err := deleteDiscussionComment(commentID, userID); err != nil {
		jsonResponse(w, 403, map[string]string{"error": "cannot delete this comment"})
		return
	}

	jsonResponse(w, 200, map[string]string{"message": "comment deleted"})
}

func handleModDeleteComment(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	vars := mux.Vars(r)
	instID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}
	commentID, err := strconv.ParseInt(vars["commentId"], 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid comment id"})
		return
	}

	// Check moderator or admin
	sd, _ := getSessionFromRequest(r)
	modID, _ := getInstitutionModerator(instID)
	if modID != userID && !sd.IsAdmin {
		jsonResponse(w, 403, map[string]string{"error": "only the moderator can do this"})
		return
	}

	if err := modDeleteDiscussionComment(commentID); err != nil {
		jsonResponse(w, 404, map[string]string{"error": "comment not found"})
		return
	}

	jsonResponse(w, 200, map[string]string{"message": "comment deleted by moderator"})
}

// ==================== Institution Meta & Photos ====================

func handleUpdateInstitutionMeta(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	idStr := mux.Vars(r)["id"]
	instID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}

	// Allow creator, moderator, or admin
	sd, _ := getSessionFromRequest(r)
	creatorID, _ := getInstitutionCreator(instID)
	modID, _ := getInstitutionModerator(instID)
	if userID != creatorID && userID != modID && !sd.IsAdmin {
		jsonResponse(w, 403, map[string]string{"error": "forbidden"})
		return
	}

	var req struct {
		Title       string `json:"title"`
		EmailDomain string `json:"email_domain"`
		City        string `json:"city"`
		Ownership   string `json:"ownership"`
		Programs    string `json:"programs"`
		Pros        string `json:"pros"`
		Cons        string `json:"cons"`
		TuitionMin  int    `json:"tuition_min"`
		TuitionMax  int    `json:"tuition_max"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}

	// Fetch current title if not provided
	if strings.TrimSpace(req.Title) == "" {
		inst, err := getInstitutionDetail(instID, 0)
		if err == nil {
			req.Title = inst.Title
		}
	}

	if err := updateInstitutionMeta(instID, strings.TrimSpace(req.Title), strings.TrimSpace(req.EmailDomain),
		strings.TrimSpace(req.City), strings.TrimSpace(req.Ownership),
		strings.TrimSpace(req.Programs), strings.TrimSpace(req.Pros), strings.TrimSpace(req.Cons),
		req.TuitionMin, req.TuitionMax); err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	jsonResponse(w, 200, map[string]string{"message": "updated"})
}

func handleUploadInstitutionPhoto(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	idStr := mux.Vars(r)["id"]
	instID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}

	// Allow creator, moderator, or admin
	sd, _ := getSessionFromRequest(r)
	creatorID, _ := getInstitutionCreator(instID)
	modID, _ := getInstitutionModerator(instID)
	if userID != creatorID && userID != modID && !sd.IsAdmin {
		jsonResponse(w, 403, map[string]string{"error": "forbidden"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "file too large (max 5MB)"})
		return
	}
	file, _, err := r.FormFile("photo")
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "no file provided"})
		return
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	contentType := http.DetectContentType(buf[:n])
	if !allowedImageTypes[contentType] {
		jsonResponse(w, 400, map[string]string{"error": "invalid image type"})
		return
	}

	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}
	filename := hex.EncodeToString(randBytes) + ext
	savePath := filepath.Join(uploadsDir, "institution-photos", filename)

	out, err := os.Create(savePath)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	defer out.Close()
	out.Write(buf[:n])
	io.Copy(out, file)

	id, err := addInstitutionPhoto(instID, userID, savePath)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	jsonResponse(w, 201, map[string]interface{}{"id": id, "path": "/" + savePath})
}

func handleDeleteInstitutionPhoto(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	vars := mux.Vars(r)
	instID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}
	photoID, err := strconv.ParseInt(vars["photoId"], 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid photo id"})
		return
	}

	sd, _ := getSessionFromRequest(r)
	creatorID, _ := getInstitutionCreator(instID)
	modID, _ := getInstitutionModerator(instID)
	if userID != creatorID && userID != modID && !sd.IsAdmin {
		jsonResponse(w, 403, map[string]string{"error": "forbidden"})
		return
	}

	if err := deleteInstitutionPhoto(photoID); err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	jsonResponse(w, 200, map[string]string{"message": "photo deleted"})
}

// ==================== Verification Photo Handlers ====================

func handleUploadVerificationPhoto(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	idStr := mux.Vars(r)["id"]
	instID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}

	already, err := isUserVerifiedForInstitution(userID, instID)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	if already {
		jsonResponse(w, 400, map[string]string{"error": "you are already verified for this institution"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "file too large (max 5MB)"})
		return
	}

	file, _, err := r.FormFile("proof")
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "no file provided"})
		return
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	contentType := http.DetectContentType(buf[:n])
	if !allowedImageTypes[contentType] {
		jsonResponse(w, 400, map[string]string{"error": "invalid image type"})
		return
	}

	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}
	filename := hex.EncodeToString(randBytes) + ext
	savePath := filepath.Join(uploadsDir, "verifications", filename)

	out, err := os.Create(savePath)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	defer out.Close()
	out.Write(buf[:n])
	io.Copy(out, file)

	id, err := createVerificationRequest(instID, userID, savePath)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, 201, map[string]interface{}{"id": id, "message": "verification request submitted, awaiting moderator review"})
}

func handleGetVerificationRequests(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	instID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}

	if !isModeratorOrAdmin(r, instID) {
		jsonResponse(w, 403, map[string]string{"error": "forbidden"})
		return
	}

	requests, err := getVerificationRequestsByInstitution(instID)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	jsonResponse(w, 200, requests)
}

func handleReviewVerificationRequest(w http.ResponseWriter, r *http.Request) {
	sd, ok := getSessionFromRequest(r)
	if !ok {
		jsonResponse(w, 401, map[string]string{"error": "unauthorized"})
		return
	}

	idStr := mux.Vars(r)["id"]
	requestID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request id"})
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}
	if req.Status != "approved" && req.Status != "rejected" {
		jsonResponse(w, 400, map[string]string{"error": "status must be approved or rejected"})
		return
	}

	vr, err := getVerificationRequest(requestID)
	if err != nil {
		jsonResponse(w, 404, map[string]string{"error": "request not found"})
		return
	}

	if !isModeratorOrAdmin(r, vr.InstitutionID) {
		jsonResponse(w, 403, map[string]string{"error": "forbidden"})
		return
	}

	if err := reviewVerificationRequest(requestID, req.Status, sd.UserID); err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	jsonResponse(w, 200, map[string]string{"message": "request " + req.Status})
}

func handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid user id"})
		return
	}

	var username string
	db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if username == adminUsername {
		jsonResponse(w, 403, map[string]string{"error": "cannot delete admin"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	// Detach child comments that reference this user's comments
	tx.Exec("UPDATE discussion_comments SET parent_id = NULL WHERE parent_id IN (SELECT id FROM discussion_comments WHERE user_id = ?)", userID)
	tx.Exec("UPDATE verification_requests SET reviewed_by = NULL WHERE reviewed_by = ?", userID)
	// Delete all user-owned data
	tx.Exec("DELETE FROM rating_votes WHERE user_id = ?", userID)
	tx.Exec("DELETE FROM comment_votes WHERE user_id = ?", userID)
	tx.Exec("DELETE FROM user_mutes WHERE user_id = ? OR muted_by = ?", userID, userID)
	tx.Exec("DELETE FROM user_bans WHERE user_id = ? OR banned_by = ?", userID, userID)
	tx.Exec("DELETE FROM verification_requests WHERE user_id = ?", userID)
	tx.Exec("DELETE FROM user_verifications WHERE user_id = ?", userID)
	tx.Exec("DELETE FROM education_history WHERE user_id = ?", userID)
	tx.Exec("DELETE FROM institution_photos WHERE uploaded_by = ?", userID)
	tx.Exec("DELETE FROM discussion_comments WHERE user_id = ?", userID)
	tx.Exec("DELETE FROM ratings WHERE user_id = ?", userID)
	tx.Exec("DELETE FROM contribution_events WHERE user_id = ?", userID)
	tx.Exec("DELETE FROM user_profiles WHERE user_id = ?", userID)
	// Reassign institutions they created to admin
	var adminID int64
	db.QueryRow("SELECT id FROM users WHERE username = ?", adminUsername).Scan(&adminID)
	if adminID > 0 {
		tx.Exec("UPDATE topics SET created_by = ? WHERE created_by = ?", adminID, userID)
	}

	if _, err = tx.Exec("DELETE FROM users WHERE id = ?", userID); err != nil {
		tx.Rollback()
		jsonResponse(w, 500, map[string]string{"error": "failed to delete user"})
		return
	}
	tx.Commit()

	// Invalidate any active sessions for this user
	sessionsMu.Lock()
	for token, sd := range sessions {
		if sd.UserID == userID {
			delete(sessions, token)
		}
	}
	sessionsMu.Unlock()

	jsonResponse(w, 200, map[string]string{"message": "user deleted"})
}

// ==================== Admin Ban Handlers ====================

func handleBanUser(w http.ResponseWriter, r *http.Request) {
	sd, _ := getSessionFromRequest(r)
	idStr := mux.Vars(r)["id"]
	targetUserID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid user id"})
		return
	}

	if targetUserID == sd.UserID {
		jsonResponse(w, 400, map[string]string{"error": "cannot ban yourself"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}

	if err := banUser(targetUserID, sd.UserID, req.Reason); err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	jsonResponse(w, 200, map[string]string{"message": "user banned"})
}

func handleUnbanUser(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	targetUserID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid user id"})
		return
	}

	if err := unbanUser(targetUserID); err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	jsonResponse(w, 200, map[string]string{"message": "user unbanned"})
}

func handleGetBannedUsers(w http.ResponseWriter, r *http.Request) {
	bans, err := getAllBannedUsers()
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	jsonResponse(w, 200, bans)
}

func handleGetAllPendingVerifications(w http.ResponseWriter, r *http.Request) {
	requests, err := getAllPendingVerificationRequests()
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}
	jsonResponse(w, 200, requests)
}

func handleAdminGetUserActivity(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid user id"})
		return
	}

	activity, err := getUserActivity(userID)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	jsonResponse(w, 200, activity)
}

// ==================== Mute Handler ====================

func handleMuteUser(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	vars := mux.Vars(r)
	instID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid institution id"})
		return
	}

	// Check moderator or admin
	sd, _ := getSessionFromRequest(r)
	modID, _ := getInstitutionModerator(instID)
	if modID != userID && !sd.IsAdmin {
		jsonResponse(w, 403, map[string]string{"error": "only the moderator can mute users"})
		return
	}

	var req struct {
		TargetUserID int64 `json:"user_id"`
		Duration     int   `json:"duration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}

	if req.Duration < 1 || req.Duration > 10080 {
		jsonResponse(w, 400, map[string]string{"error": "duration must be 1-10080 minutes"})
		return
	}

	if req.TargetUserID == userID {
		jsonResponse(w, 400, map[string]string{"error": "cannot mute yourself"})
		return
	}

	if err := muteUser(instID, req.TargetUserID, userID, req.Duration); err != nil {
		jsonResponse(w, 500, map[string]string{"error": "server error"})
		return
	}

	jsonResponse(w, 200, map[string]string{"message": "user muted"})
}

// ==================== Karma Vote Handlers ====================

func handleVoteRating(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	vars := mux.Vars(r)
	ratingID, err := strconv.ParseInt(vars["ratingId"], 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid rating id"})
		return
	}

	var req struct {
		Vote int `json:"vote"` // 1, -1, or 0 to remove
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}
	if req.Vote != 1 && req.Vote != -1 && req.Vote != 0 {
		jsonResponse(w, 400, map[string]string{"error": "vote must be 1, -1, or 0"})
		return
	}

	if err := voteOnRating(ratingID, userID, req.Vote); err != nil {
		if strings.Contains(err.Error(), "own rating") {
			jsonResponse(w, 403, map[string]string{"error": err.Error()})
		} else {
			jsonResponse(w, 500, map[string]string{"error": "server error"})
		}
		return
	}
	jsonResponse(w, 200, map[string]string{"message": "vote recorded"})
}

func handleVoteComment(w http.ResponseWriter, r *http.Request) {
	userID, _ := getUserIDFromRequest(r)
	vars := mux.Vars(r)
	commentID, err := strconv.ParseInt(vars["commentId"], 10, 64)
	if err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid comment id"})
		return
	}

	var req struct {
		Vote int `json:"vote"` // 1, -1, or 0 to remove
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}
	if req.Vote != 1 && req.Vote != -1 && req.Vote != 0 {
		jsonResponse(w, 400, map[string]string{"error": "vote must be 1, -1, or 0"})
		return
	}

	if err := voteOnComment(commentID, userID, req.Vote); err != nil {
		if strings.Contains(err.Error(), "own comment") {
			jsonResponse(w, 403, map[string]string{"error": err.Error()})
		} else {
			jsonResponse(w, 500, map[string]string{"error": "server error"})
		}
		return
	}
	jsonResponse(w, 200, map[string]string{"message": "vote recorded"})
}

// ==================== Stats Handlers ====================

func handleGetStats(w http.ResponseWriter, r *http.Request) {
	var totalRatings, totalInstitutions, totalUsers int

	db.QueryRow("SELECT COUNT(*) FROM ratings").Scan(&totalRatings)
	db.QueryRow("SELECT COUNT(*) FROM topics").Scan(&totalInstitutions)
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)

	jsonResponse(w, 200, map[string]int{
		"total_ratings":      totalRatings,
		"total_institutions": totalInstitutions,
		"total_users":        totalUsers,
	})
}

func handleSitemap(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id FROM topics ORDER BY id")
	if err != nil {
		http.Error(w, "Error generating sitemap", 500)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://rateed.org/</loc>
    <changefreq>weekly</changefreq>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>https://rateed.org/home</loc>
    <changefreq>daily</changefreq>
    <priority>0.8</priority>
  </url>`)

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		fmt.Fprintf(w, `
  <url>
    <loc>https://rateed.org/institution/%d</loc>
    <changefreq>weekly</changefreq>
    <priority>0.8</priority>
  </url>`, id)
	}
	fmt.Fprint(w, "\n</urlset>")
}
