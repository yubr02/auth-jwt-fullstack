package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/mail"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	Port      string
	DBHost    string
	DBPort    string
	DBName    string
	DBUser    string
	DBPass    string
	JWTSecret string
	JWTTTL    time.Duration
}

type User struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at,omitempty"`
}

type Server struct {
	db       *sql.DB
	config   Config
	frontend string
}

func main() {
	config := loadConfig()

	db, err := sql.Open("mysql", dsn(config))
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	server := &Server{
		db:       db,
		config:   config,
		frontend: resolveFrontendDir(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", server.handleHealth)
	mux.HandleFunc("/api/register", server.handleRegister)
	mux.HandleFunc("/api/login", server.handleLogin)
	mux.HandleFunc("/api/profile", server.handleProfile)
	mux.Handle("/", server.handleFrontend())

	log.Printf("Servidor em http://localhost:%s", config.Port)
	if err := http.ListenAndServe(":"+config.Port, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

func loadConfig() Config {
	return Config{
		Port:      env("APP_PORT", "8080"),
		DBHost:    env("DB_HOST", "127.0.0.1"),
		DBPort:    env("DB_PORT", "3306"),
		DBName:    env("DB_NAME", "auth_jwt"),
		DBUser:    env("DB_USER", "root"),
		DBPass:    env("DB_PASS", ""),
		JWTSecret: env("JWT_SECRET", "troque-esta-chave-em-producao"),
		JWTTTL:    time.Duration(envInt("JWT_TTL", 3600)) * time.Second,
	}
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func dsn(config Config) string {
	return config.DBUser + ":" + config.DBPass + "@tcp(" + config.DBHost + ":" + config.DBPort + ")/" +
		config.DBName + "?charset=utf8mb4&parseTime=true"
}

func resolveFrontendDir() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return "frontend"
	}

	candidates := []string{
		filepath.Join(workingDir, "frontend"),
		filepath.Join(workingDir, "..", "frontend"),
	}

	for _, candidate := range candidates {
		info, statErr := os.Stat(candidate)
		if statErr == nil && info.IsDir() {
			absolute, absErr := filepath.Abs(candidate)
			if absErr == nil {
				return absolute
			}

			return candidate
		}
	}

	return "frontend"
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(writer, request)
	})
}

func (server *Server) handleFrontend() http.Handler {
	fileServer := http.FileServer(http.Dir(server.frontend))

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") {
			http.NotFound(writer, request)
			return
		}

		if request.URL.Path == "/" || request.URL.Path == "/frontend" || request.URL.Path == "/frontend/" || request.URL.Path == "/frontend/index.html" {
			http.ServeFile(writer, request, filepath.Join(server.frontend, "index.html"))
			return
		}

		if strings.HasPrefix(request.URL.Path, "/frontend/") {
			request.URL.Path = strings.TrimPrefix(request.URL.Path, "/frontend")
		}

		fileServer.ServeHTTP(writer, request)
	})
}

func (server *Server) handleHealth(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (server *Server) handleRegister(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"message": "Metodo nao permitido."})
		return
	}

	var payload struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"message": "JSON invalido."})
		return
	}

	payload.Name = strings.TrimSpace(payload.Name)
	payload.Email = strings.ToLower(strings.TrimSpace(payload.Email))

	if payload.Name == "" || payload.Email == "" || payload.Password == "" {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"message": "Nome, email e senha sao obrigatorios."})
		return
	}

	if _, err := mail.ParseAddress(payload.Email); err != nil {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"message": "Email invalido."})
		return
	}

	if len(payload.Password) < 6 {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"message": "A senha deve ter pelo menos 6 caracteres."})
		return
	}

	var existingID int64
	err := server.db.QueryRow("SELECT id FROM users WHERE email = ? LIMIT 1", payload.Email).Scan(&existingID)
	if err == nil {
		writeJSON(writer, http.StatusConflict, map[string]string{"message": "Ja existe uma conta com este email."})
		return
	}

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"message": "Erro no banco de dados."})
		return
	}

	hash, err := hashPassword(payload.Password)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"message": "Erro ao processar a senha."})
		return
	}

	result, err := server.db.Exec(
		"INSERT INTO users (name, email, password_hash) VALUES (?, ?, ?)",
		payload.Name,
		payload.Email,
		hash,
	)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"message": "Erro no banco de dados."})
		return
	}

	userID, _ := result.LastInsertId()
	token, err := server.createJWT(userID, payload.Email)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"message": "Erro ao gerar token."})
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]any{
		"message": "Conta criada com sucesso.",
		"token":   token,
		"user": User{
			ID:    userID,
			Name:  payload.Name,
			Email: payload.Email,
		},
	})
}

func (server *Server) handleLogin(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"message": "Metodo nao permitido."})
		return
	}

	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"message": "JSON invalido."})
		return
	}

	payload.Email = strings.ToLower(strings.TrimSpace(payload.Email))

	if payload.Email == "" || payload.Password == "" {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"message": "Email e senha sao obrigatorios."})
		return
	}

	var user User
	var passwordHash string
	var createdAt time.Time
	err := server.db.QueryRow(
		"SELECT id, name, email, password_hash, created_at FROM users WHERE email = ? LIMIT 1",
		payload.Email,
	).Scan(&user.ID, &user.Name, &user.Email, &passwordHash, &createdAt)
	if err != nil {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"message": "Credenciais invalidas."})
		return
	}

	if !checkPassword(payload.Password, passwordHash) {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"message": "Credenciais invalidas."})
		return
	}

	user.CreatedAt = createdAt.Format(time.RFC3339)
	token, err := server.createJWT(user.ID, user.Email)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"message": "Erro ao gerar token."})
		return
	}

	writeJSON(writer, http.StatusOK, map[string]any{
		"message": "Login realizado com sucesso.",
		"token":   token,
		"user":    user,
	})
}

func (server *Server) handleProfile(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"message": "Metodo nao permitido."})
		return
	}

	userID, err := server.userIDFromRequest(request)
	if err != nil {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"message": err.Error()})
		return
	}

	var user User
	var createdAt time.Time
	err = server.db.QueryRow(
		"SELECT id, name, email, created_at FROM users WHERE id = ? LIMIT 1",
		userID,
	).Scan(&user.ID, &user.Name, &user.Email, &createdAt)
	if err != nil {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"message": "Usuario nao encontrado."})
		return
	}

	user.CreatedAt = createdAt.Format(time.RFC3339)
	writeJSON(writer, http.StatusOK, map[string]any{"user": user})
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func checkPassword(password, expected string) bool {
	return bcrypt.CompareHashAndPassword([]byte(expected), []byte(password)) == nil
}

func (server *Server) createJWT(userID int64, email string) (string, error) {
	header, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}

	now := time.Now()
	payload, err := json.Marshal(map[string]any{
		"sub":   userID,
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(server.config.JWTTTL).Unix(),
	})
	if err != nil {
		return "", err
	}

	headerPart := base64.RawURLEncoding.EncodeToString(header)
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := headerPart + "." + payloadPart

	mac := hmac.New(sha256.New, []byte(server.config.JWTSecret))
	_, _ = mac.Write([]byte(signingInput))
	signaturePart := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signaturePart, nil
}

func (server *Server) userIDFromRequest(request *http.Request) (int64, error) {
	header := strings.TrimSpace(request.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return 0, errors.New("Token nao enviado.")
	}

	token := strings.TrimSpace(header[7:])
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, errors.New("Token invalido.")
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(server.config.JWTSecret))
	_, _ = mac.Write([]byte(signingInput))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return 0, errors.New("Assinatura do token invalida.")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, errors.New("Token invalido.")
	}

	var claims struct {
		Sub int64 `json:"sub"`
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return 0, errors.New("Token invalido.")
	}

	if claims.Exp < time.Now().Unix() {
		return 0, errors.New("Token expirado.")
	}

	return claims.Sub, nil
}
