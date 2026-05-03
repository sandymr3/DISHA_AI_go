// DISHA AI Backend — Entry point
// Wires all dependencies, registers routes, and starts the HTTP server with graceful shutdown.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"disha-backend/config"
	staticdata "disha-backend/data"
	"disha-backend/internal/ai"
	"disha-backend/internal/applications"
	"disha-backend/internal/data"
	"disha-backend/internal/ecp"
	"disha-backend/internal/handlers"
	"disha-backend/internal/ingestion"
	"disha-backend/internal/loans"
	"disha-backend/internal/matching"
	"disha-backend/internal/middleware"
	"disha-backend/internal/students"
)

const version = "1.0.0"

func main() {
	cfg := config.Load()

	// Configure structured logging
	if cfg.IsDevelopment() {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
	} else {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	}

	// Root context for background goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Load embedded data ---
	var universities []matching.University
	if err := json.Unmarshal(staticdata.UniversitiesJSON, &universities); err != nil {
		log.Fatalf("FATAL: failed to parse universities.json: %v", err)
	}
	slog.Info("loaded universities", "count", len(universities))

	var loanOffers []loans.LoanOffer
	if err := json.Unmarshal(staticdata.LoanOffersJSON, &loanOffers); err != nil {
		log.Fatalf("FATAL: failed to parse loan_offers.json: %v", err)
	}
	slog.Info("loaded loan offers", "count", len(loanOffers))

	// --- Initialize services ---
	matcher := matching.NewMatcher(universities)
	studentStore := students.NewStore(ctx, 24*time.Hour)
	appStore := applications.NewStore()
	rateLimiter := ai.NewRateLimiter(ctx, cfg.RateLimitRequests, cfg.RateLimitWindowHours)

	// --- Firebase / Firestore setup ---
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "disha-ai-d9394"
	}
	credsJSON := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON")

	// Firestore (for dynamic university DB + student persistence)
	firestoreClient, err := data.NewFirestoreClient(ctx, projectID, credsJSON)
	if err != nil {
		slog.Warn("Failed to initialize Firestore (continuing without Firestore persistence)", "error", err)
	}

	// Firebase Auth middleware (for verifying client ID tokens)
	var fbAuthMiddleware *middleware.FirebaseAuth
	fbAuthMiddleware, err = middleware.NewFirebaseAuth(ctx, projectID, credsJSON)
	if err != nil {
		slog.Warn("Failed to initialize Firebase Auth middleware (auth verification disabled)", "error", err)
	} else {
		slog.Info("Firebase Auth middleware initialized", "projectID", projectID)
	}

	// Firestore-backed student store (used when auth is available)
	var fsStudentStore *students.FirestoreStore
	if firestoreClient != nil {
		fsStudentStore = students.NewFirestoreStore(firestoreClient)
		slog.Info("Firestore student store active — profiles will persist across restarts")
	} else {
		slog.Warn("Firestore student store unavailable — using in-memory store only")
	}

	// --- Serper.dev ---
	serperAPIKey := os.Getenv("SERPER_API_KEY")
	var serperClient *ingestion.SerperClient
	if serperAPIKey != "" {
		serperClient = ingestion.NewSerperClient(serperAPIKey)
	} else {
		slog.Warn("SERPER_API_KEY not set - dynamic search/ingestion will fail")
	}

	// --- Gemini AI client (optional) ---
	aiClient, err := ai.NewClient(ctx, cfg.GoogleAPIKey)
	if err != nil {
		slog.Error("failed to create Gemini client, chat will use fallbacks", "error", err)
	}
	if !cfg.HasGeminiKey() {
		slog.Warn("GOOGLE_API_KEY not set — chat endpoint will use fallback responses")
	}

	// --- Pre-load demo students into in-memory store ---
	loadDemoStudents(ctx, studentStore)

	// --- Create handlers ---
	ecpHandler := handlers.NewECPHandler()
	studentsHandler := handlers.NewStudentsHandlerWithFirestore(studentStore, fsStudentStore)
	uniHandler := handlers.NewUniversitiesHandler(matcher, studentStore)
	loansHandler := handlers.NewLoansHandler(loanOffers, studentStore, appStore)
	chatHandler := handlers.NewChatHandler(aiClient, studentStore, matcher, rateLimiter)
	healthHandler := handlers.NewHealthHandler(studentStore, appStore, version)
	passportHandler := handlers.NewPassportHandler(studentStore, matcher)
	dreamGapHandler := handlers.NewDreamGapHandler(studentStore, matcher)
	roiHandler := handlers.NewROIHandler(studentStore, matcher)
	ingestHandler := handlers.NewIngestionHandler(serperClient, aiClient, firestoreClient)
	searchHandler := handlers.NewSearchHandler(firestoreClient)

	// --- Setup router ---
	r := chi.NewRouter()

	// Global middleware stack
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(middleware.Logging())

	// authMiddleware wraps a route if Firebase Auth is available; otherwise passes through.
	optionalAuth := func(next http.Handler) http.Handler {
		if fbAuthMiddleware != nil {
			return fbAuthMiddleware.Middleware(next)
		}
		return next // pass-through — no auth verification in dev without creds
	}

	r.Route("/api/v1", func(r chi.Router) {
		// ─── Public routes (no auth required) ───────────────────────────────
		r.Get("/health", healthHandler.Handle)
		r.Post("/ecp/calculate", ecpHandler.Calculate)
		r.Post("/ecp/simulate", ecpHandler.Simulate)
		r.Get("/search/autocomplete", searchHandler.HandleAutocomplete)

		// ─── Optional-auth ingestion ─────────────────────────────────────────
		r.Post("/ingest", ingestHandler.HandleIngest)

		// ─── Auth-protected routes ───────────────────────────────────────────
		// Students — auth injects UID so Firestore uses it as doc ID
		r.With(optionalAuth).Post("/students", studentsHandler.Create)
		r.With(optionalAuth).Get("/students/{studentId}", studentsHandler.Get)
		r.With(optionalAuth).Put("/students/{studentId}", studentsHandler.Update)

		// Universities
		r.With(optionalAuth).Get("/universities", uniHandler.GetAll)
		r.With(optionalAuth).Get("/universities/match", uniHandler.Match)
		r.With(optionalAuth).Get("/universities/{universityId}", uniHandler.GetByID)

		// Loans
		r.With(optionalAuth).Get("/loans/offers", loansHandler.GetOffers)
		r.With(optionalAuth).Post("/loans/emi", loansHandler.CalculateEMI)

		// ROI
		r.With(optionalAuth).Post("/roi/calculate", roiHandler.Calculate)

		// Chat
		r.With(optionalAuth).Post("/chat", chatHandler.Handle)

		// Applications
		r.With(optionalAuth).Post("/applications", loansHandler.CreateApplication)
		r.With(optionalAuth).Get("/applications/{applicationId}", loansHandler.GetApplication)

		// Funding Passport
		r.With(optionalAuth).Get("/funding-passport/{studentId}", passportHandler.Handle)

		// Dream Gap
		r.With(optionalAuth).Get("/dream-gap", dreamGapHandler.Handle)
	})

	// --- Start server ---
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // 60s for SSE streaming
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigChan
		slog.Info("received signal, shutting down", "signal", sig)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		cancel() // Cancel root context to stop background goroutines

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown error", "error", err)
		}
		slog.Info("shutdown complete")
	}()

	slog.Info(fmt.Sprintf("DISHA API server listening on :%s", cfg.Port),
		"version", version,
		"env", cfg.Env,
		"gemini", cfg.HasGeminiKey(),
		"firestore", firestoreClient != nil,
		"firebase_auth", fbAuthMiddleware != nil,
	)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("FATAL: server failed: %v", err)
	}
}

func loadDemoStudents(ctx context.Context, store *students.Store) {
	demos := []struct {
		id      string
		profile ecp.StudentProfile
	}{
		{
			id: "STU-PRIYA001",
			profile: ecp.StudentProfile{
				Name: "Priya Sharma", CGPA: 7.8, GREScore: 312,
				FamilyIncome: ecp.Income8To20L, HasCoApplicant: true, CoApplicantIncome: ecp.Income8To20L,
				TargetCountry: ecp.CountryUSA, TargetProgram: ecp.ProgramMS, Intake: ecp.IntakeSep2026,
			},
		},
		{
			id: "STU-ROHAN002",
			profile: ecp.StudentProfile{
				Name: "Rohan Mehta", CGPA: 8.1, GREScore: 318,
				FamilyIncome: ecp.Income8To20L, HasCoApplicant: true, CoApplicantIncome: ecp.Income8To20L,
				TargetCountry: ecp.CountryUSA, TargetProgram: ecp.ProgramMBA, Intake: ecp.IntakeJan2026,
			},
		},
		{
			id: "STU-ANJALI003",
			profile: ecp.StudentProfile{
				Name: "Anjali Nair", CGPA: 8.5, GREScore: 0,
				FamilyIncome: ecp.Income8To20L, HasCoApplicant: false,
				TargetCountry: ecp.CountryCanada, TargetProgram: ecp.ProgramMiM, Intake: ecp.IntakeSep2026,
			},
		},
	}

	for _, d := range demos {
		record, err := store.CreateWithID(ctx, d.id, d.profile)
		if err != nil {
			slog.Error("failed to load demo student", "id", d.id, "error", err)
			continue
		}
		slog.Info("loaded demo student", "id", d.id, "ecp_score", record.ECPResult.Score, "tier", record.ECPResult.Tier)
	}
}
