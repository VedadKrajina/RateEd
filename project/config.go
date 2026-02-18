package main

var (
	maxUploadSize int64 = 5 << 20 // 5MB
	allowedImageTypes   = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	institutionCreateThreshold int = 5
	adminUsername               string = "vk15"
	uploadsDir                  string = "uploads"
)
