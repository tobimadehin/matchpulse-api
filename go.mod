module github.com/tobimadehin/matchpulse-api

go 1.21

require (
	github.com/gorilla/mux v1.8.0
	github.com/rs/cors v1.10.1
)

// Why these specific dependencies?
//
// github.com/gorilla/mux - Provides powerful HTTP routing with URL parameter extraction
// This is more feature-rich than the standard library's ServeMux, allowing us to
// create clean RESTful routes like /api/v1/matches/{id} with type validation
//
// github.com/rs/cors - Handles Cross-Origin Resource Sharing automatically
// Essential for a testing API that needs to work with any frontend application
// The standard library doesn't provide CORS handling out of the box
