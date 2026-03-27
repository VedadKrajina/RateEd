package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
)

type InstitutionPageData struct {
	ID          string
	Name        string
	Description string
	Type        string
	City        string
	AvgRating   string
	RatingCount int
	JSONLD      template.HTML
}

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

	// Stats
	r.HandleFunc("/api/stats", handleGetStats).Methods("GET")

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
	r.HandleFunc("/api/institutions/{id}/discussion/{commentId}/vote", requireAuth(handleVoteComment)).Methods("POST")
	r.HandleFunc("/api/institutions/{id}/ratings/{ratingId}/vote", requireAuth(handleVoteRating)).Methods("POST")
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

	// SEO
	r.HandleFunc("/google8866962a8ef7ee77.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "google8866962a8ef7ee77.html")
	})
	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "User-agent: *\nAllow: /\nDisallow: /admin\nDisallow: /api/\n\nSitemap: https://rateed.org/sitemap.xml\n")
	})
	r.HandleFunc("/sitemap.xml", handleSitemap)

	// Static files and uploads
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir))))

	// Page routes
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, "landing.html", nil)
	})
	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, "login.html", nil)
	})
	r.HandleFunc("/home", func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, "index.html", nil)
	})
	r.HandleFunc("/institution/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		var data InstitutionPageData
		data.ID = id

		var avgRating float64
		err := db.QueryRow(`
			SELECT t.title, COALESCE(t.description,''), COALESCE(t.institution_type,''), COALESCE(t.city,''),
			       COALESCE(AVG((r.score_academic+r.score_infrastructure+r.score_student_life+r.score_career+r.score_guidance)/5.0),0.0),
			       COUNT(r.id)
			FROM topics t
			LEFT JOIN ratings r ON r.topic_id = t.id
			WHERE t.id = ?
			GROUP BY t.id
		`, id).Scan(&data.Name, &data.Description, &data.Type, &data.City, &avgRating, &data.RatingCount)
		if err == nil {
			if data.RatingCount > 0 {
				data.AvgRating = fmt.Sprintf("%.1f", avgRating)
			}
			ld := map[string]interface{}{
				"@context": "https://schema.org",
				"@type":    "EducationalOrganization",
				"name":     data.Name,
				"url":      "https://rateed.org/institution/" + id,
			}
			if data.City != "" {
				ld["address"] = map[string]string{"@type": "PostalAddress", "addressLocality": data.City}
			}
			if data.RatingCount > 0 {
				ld["aggregateRating"] = map[string]string{
					"@type": "AggregateRating", "ratingValue": data.AvgRating,
					"bestRating": "5", "worstRating": "1", "ratingCount": strconv.Itoa(data.RatingCount),
				}
			}
			if b, jsonErr := json.MarshalIndent(ld, "", "  "); jsonErr == nil {
				data.JSONLD = template.HTML(b)
			}
		}
		templates.ExecuteTemplate(w, "institution.html", data)
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
