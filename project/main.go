package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

var templates *template.Template

func main() {
	// DATA_DIR is where the DB and uploads live (set to a persistent volume on Railway)
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "."
	}
	uploadsDir = dataDir + "/uploads"

	initDB(dataDir)
	defer db.Close()

	os.MkdirAll(uploadsDir+"/profiles", 0755)
	os.MkdirAll(uploadsDir+"/topics", 0755)
	os.MkdirAll(uploadsDir+"/verifications", 0755)
	os.MkdirAll(uploadsDir+"/institution-photos", 0755)

	templates = template.Must(template.ParseGlob("templates/*.html"))

	r := mux.NewRouter()

	// Auth
	r.HandleFunc("/api/register", handleRegister).Methods("POST")
	r.HandleFunc("/api/login", handleLogin).Methods("POST")
	r.HandleFunc("/api/logout", handleLogout).Methods("POST")
	r.HandleFunc("/api/me", handleMe).Methods("GET")

	// Institutions
	r.HandleFunc("/api/institutions", handleListInstitutions).Methods("GET")
	r.HandleFunc("/api/institutions", requireAuth(handleCreateInstitution)).Methods("POST")
	r.HandleFunc("/api/institutions/{id}", handleGetInstitution).Methods("GET")
	r.HandleFunc("/api/institutions/{id}/rate", requireAuth(handleRateInstitution)).Methods("POST")
	r.HandleFunc("/api/institutions/{id}/verify", requireAuth(handleVerifyEmail)).Methods("POST")
	r.HandleFunc("/api/institutions/{id}/cover", requireAuth(handleUploadInstitutionCover)).Methods("POST")
	r.HandleFunc("/api/institutions/{id}/meta", requireAuth(handleUpdateInstitutionMeta)).Methods("PATCH")
	r.HandleFunc("/api/institutions/{id}/photos", requireAuth(handleUploadInstitutionPhoto)).Methods("POST")
	r.HandleFunc("/api/institutions/{id}/photos/{photoId}", requireAuth(handleDeleteInstitutionPhoto)).Methods("DELETE")
	r.HandleFunc("/api/institutions/{id}/leaderboard", handleSchoolLeaderboard).Methods("GET")
	r.HandleFunc("/api/institutions/{id}/discussion", handleGetDiscussion).Methods("GET")
	r.HandleFunc("/api/institutions/{id}/discussion", requireAuth(handlePostComment)).Methods("POST")
	r.HandleFunc("/api/institutions/{id}/discussion/{commentId}", requireAuth(handleDeleteComment)).Methods("DELETE")
	r.HandleFunc("/api/institutions/{id}/discussion/{commentId}/mod", requireAuth(handleModDeleteComment)).Methods("DELETE")
	r.HandleFunc("/api/institutions/{id}/mute", requireAuth(handleMuteUser)).Methods("POST")

	// School rankings
	r.HandleFunc("/api/schools/rankings", handleTopSchools).Methods("GET")

	// Profiles
	r.HandleFunc("/api/users/{username}", handleGetProfile).Methods("GET")
	r.HandleFunc("/api/profile/picture", requireAuth(handleUploadProfilePicture)).Methods("POST")
	r.HandleFunc("/api/profile/picture", requireAuth(handleDeleteProfilePicture)).Methods("DELETE")
	r.HandleFunc("/api/profile/education", requireAuth(handleAddEducation)).Methods("POST")
	r.HandleFunc("/api/profile/education/{id}", requireAuth(handleDeleteEducation)).Methods("DELETE")

	// Verification photo
	r.HandleFunc("/api/institutions/{id}/verify-photo", requireAuth(handleUploadVerificationPhoto)).Methods("POST")
	r.HandleFunc("/api/institutions/{id}/verification-requests", requireAuth(handleGetVerificationRequests)).Methods("GET")
	r.HandleFunc("/api/verification-requests/{id}", requireAuth(handleReviewVerificationRequest)).Methods("PUT")

	// Admin
	r.HandleFunc("/api/admin/users", requireAdmin(handleAdminListUsers)).Methods("GET")
	r.HandleFunc("/api/admin/users/{id}/points", requireAdmin(handleAdminSetPoints)).Methods("PUT")
	r.HandleFunc("/api/admin/verification-requests", requireAdmin(handleGetAllPendingVerifications)).Methods("GET")
	r.HandleFunc("/api/admin/users/{id}/activity", requireAdmin(handleAdminGetUserActivity)).Methods("GET")
	r.HandleFunc("/api/admin/users/{id}/ban", requireAdmin(handleBanUser)).Methods("POST")
	r.HandleFunc("/api/admin/users/{id}/ban", requireAdmin(handleUnbanUser)).Methods("DELETE")
	r.HandleFunc("/api/admin/bans", requireAdmin(handleGetBannedUsers)).Methods("GET")

	// Static files and uploads
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir))))

	// Page routes
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, "login.html", nil)
	})
	r.HandleFunc("/home", func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, "index.html", nil)
	})
	r.HandleFunc("/institution/{id}", func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, "institution.html", nil)
	})
	r.HandleFunc("/profile/{username}", func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, "profile.html", nil)
	})
	r.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, "admin.html", nil)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3141"
	}
	log.Println("RateEd server starting on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
