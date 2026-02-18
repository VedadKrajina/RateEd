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
	initDB()
	defer db.Close()

	os.MkdirAll("uploads/profiles", 0755)
	os.MkdirAll("uploads/topics", 0755)

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

	// Admin
	r.HandleFunc("/api/admin/users", requireAdmin(handleAdminListUsers)).Methods("GET")
	r.HandleFunc("/api/admin/users/{id}/points", requireAdmin(handleAdminSetPoints)).Methods("PUT")

	// Static files and uploads
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

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

	log.Println("RateEd server starting on http://localhost:3141")
	log.Fatal(http.ListenAndServe(":3141", r))
}
