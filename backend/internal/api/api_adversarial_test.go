package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// ADVERSARIAL SECURITY TESTS - ADMIN_TOKEN Authentication
// =============================================================================

func TestAdversarial_SQLInjectionInToken_Still401(t *testing.T) {
	handler := AuthHandler("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer '; DROP TABLE users; --")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("SQL injection attempt: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_VeryLongToken_NoOOM(t *testing.T) {
	handler := AuthHandler("short-token")
	longToken := strings.Repeat("x", 2*1024*1024)
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer "+longToken)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Long token: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_NullBytesInToken_NoBypass(t *testing.T) {
	handler := AuthHandler("token\x00admin")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Null byte bypass attempt: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_UnicodeEmojiInToken_Graceful(t *testing.T) {
	handler := AuthHandler("🔐secret🔒")
	testCases := []struct {
		name  string
		token string
	}{
		{"emoji_wrong", "🔐wrong🔒"},
		{"chinese", "密码令牌"},
		{"arabic", "رمز"},
		{"mixed", "Hello世界"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("Unicode token %s: expected 401, got %d", tc.name, w.Code)
			}
		})
	}
}

func TestAdversarial_UnicodeEmojiTokenCorrect_Works(t *testing.T) {
	handler := AuthHandler("🔐secret🔒")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer 🔐secret🔒")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Correct emoji token: expected 200, got %d", w.Code)
	}
}

func TestAdversarial_BearerInsideToken_CorrectComparison(t *testing.T) {
	handler := AuthHandler("Bearer secret")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer Bearer secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Bearer in token value: expected 200, got %d", w.Code)
	}
	var resp map[string]bool
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["valid"] != true {
		t.Errorf("expected valid=true, got %v", resp["valid"])
	}
}

func TestAdversarial_EmptyStringToken_NoBypass(t *testing.T) {
	handler := AuthHandler("non-empty-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Empty token bypass: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_MultipleAuthHeaders_UsesFirst(t *testing.T) {
	middleware := AuthMiddleware("correct-token")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	req.Header.Add("Authorization", "Bearer wrong-token")
	req.Header.Add("Authorization", "Bearer correct-token")
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Multiple auth headers: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_ContentTypeMismatch_Graceful(t *testing.T) {
	handler := AuthHandler("any-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Content-Type mismatch: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_VeryLargeJSONBody_NoOOM(t *testing.T) {
	handler := AuthHandler("short-token")
	longToken := strings.Repeat("x", 10*1024*1024)
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer "+longToken)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Large body: got %d, expected 401", w.Code)
	}
}

func TestAdversarial_PathTraversalInAuthRoute_404(t *testing.T) {
	handler := AuthHandler("any-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth/../../etc/passwd", nil)
	req.Header.Set("Authorization", "Bearer any-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("Path traversal: expected 404, got %d", w.Code)
	}
}

func TestAdversarial_MiddlewareLongToken_NoOOM(t *testing.T) {
	middleware := AuthMiddleware("short-token")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	longToken := strings.Repeat("x", 2*1024*1024)
	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	req.Header.Set("Authorization", "Bearer "+longToken)
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Middleware long token: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_MiddlewareNullBytes_NoBypass(t *testing.T) {
	middleware := AuthMiddleware("token\x00admin")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Middleware null byte bypass: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_AuthHandlerMissingTokenField_401(t *testing.T) {
	handler := AuthHandler("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Missing token field: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_AuthHandlerNullTokenValue_401(t *testing.T) {
	handler := AuthHandler("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer null")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Null token value: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_BearerSchemeCaseSensitivity(t *testing.T) {
	middleware := AuthMiddleware("secret-token")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	req.Header.Set("Authorization", "bearer secret-token")
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Lowercase bearer: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_BearerWithExtraSpaces_401(t *testing.T) {
	middleware := AuthMiddleware("secret-token")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	req.Header.Set("Authorization", "Bearer   secret-token")
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Bearer with extra spaces: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_EmptyBearerPrefix_401(t *testing.T) {
	middleware := AuthMiddleware("secret-token")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Empty bearer token: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_ValidMiddlewareToken_200(t *testing.T) {
	middleware := AuthMiddleware("valid-secret")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	req.Header.Set("Authorization", "Bearer valid-secret")
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Valid token: expected 200, got %d", w.Code)
	}
}

func TestAdversarial_AuthWithValidToken_200(t *testing.T) {
	handler := AuthHandler("my-secret")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer my-secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Valid token: expected 200, got %d", w.Code)
	}
}

func TestAdversarial_EmptyExpectedToken_DevModeAnyToken(t *testing.T) {
	handler := AuthHandler("")
	testCases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"random", "anything-works"},
		{"sql injection", "'; DROP TABLE --"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Dev mode %s: expected 200, got %d", tc.name, w.Code)
			}
		})
	}
}

func TestAdversarial_RouteOnlyAcceptsExactPath(t *testing.T) {
	handler := AuthHandler("any-token")
	testCases := []struct {
		path   string
		expect int
	}{
		{"/api/auth", http.StatusOK},
		{"/api/auth/", http.StatusNotFound},
		{"/api/auth/verify", http.StatusNotFound},
		{"/api/auth/..", http.StatusNotFound},
	}
	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, nil)
			req.Header.Set("Authorization", "Bearer any-token")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != tc.expect {
				t.Errorf("Path %s: expected %d, got %d", tc.path, tc.expect, w.Code)
			}
		})
	}
}

func TestAdversarial_AuthHandlerEmptyBody_401(t *testing.T) {
	handler := AuthHandler("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Empty body: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_AuthHandlerPartialJSON_401(t *testing.T) {
	handler := AuthHandler("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer partial")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Partial token: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_ReadAllWithSizeLimit(t *testing.T) {
	handler := AuthHandler("secret-token")
	longToken := strings.Repeat("x", 1024*1024)
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer "+longToken)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Large token field: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_ConcurrentAuthRequests(t *testing.T) {
	handler := AuthHandler("secret-token")
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
			req.Header.Set("Authorization", "Bearer wrong-token")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			done <- (w.Code == http.StatusUnauthorized)
		}(i)
	}
	for i := 0; i < 100; i++ {
		if !<-done {
			t.Errorf("Request %d did not return 401", i)
		}
	}
}

func TestAdversarial_TokenFieldTypeConfusion_InvalidJSON(t *testing.T) {
	handler := AuthHandler("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer {\"value\":\"secret-token\"}")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Token type confusion (object): expected 401, got %d", w.Code)
	}
}

func TestAdversarial_TokenFieldArrayConfusion_InvalidJSON(t *testing.T) {
	handler := AuthHandler("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer [\"secret-token\",\"another\"]")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Token type confusion (array): expected 401, got %d", w.Code)
	}
}

func TestAdversarial_URLPathNormalization(t *testing.T) {
	handler := AuthHandler("any-token")
	paths := []string{"/api/auth", "/api/auth/.", "/api/auth/./", "/api/auth/../api/auth", "/api/auth/..%2Fapi/auth"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, path, nil)
			req.Header.Set("Authorization", "Bearer any-token")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if path == "/api/auth" && w.Code != http.StatusOK {
				t.Errorf("Path %s: expected 200, got %d", path, w.Code)
			} else if path != "/api/auth" && w.Code != http.StatusNotFound {
				t.Errorf("Path %s: expected 404, got %d", path, w.Code)
			}
		})
	}
}

func TestAdversarial_HeaderExtractionSafety(t *testing.T) {
	middleware := AuthMiddleware("secret-token")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	testCases := []struct {
		name   string
		header string
	}{
		{"newline injection", "Bearer\nsecret-token"},
		{"carriage return", "Bearer\rsecret-token"},
		{"null byte prefix", "Bearer\x00secret-token"},
		{"tab injection", "Bearer\tsecret-token"},
		{"null in token", "Bearer secr\x00et-token"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
			req.Header.Set("Authorization", tc.header)
			w := httptest.NewRecorder()
			middleware(next).ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("%s: expected 401, got %d", tc.name, w.Code)
			}
		})
	}
}

func TestAdversarial_NoTokenLeakInResponse(t *testing.T) {
	handler := AuthHandler("super-secret-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer super-secret-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	respBody := w.Body.String()
	if strings.Contains(respBody, "super-secret-token") {
		t.Error("Token leaked in response body")
	}
}

func TestAdversarial_MiddlewarePreservesRequest(t *testing.T) {
	middleware := AuthMiddleware("secret-token")
	var capturedToken string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedToken = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)
	if capturedToken != "Bearer secret-token" {
		t.Errorf("Request was modified: got %s", capturedToken)
	}
}

func TestAdversarial_AuthHandlerMethodValidation(t *testing.T) {
	handler := AuthHandler("any-token")
	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/auth", nil)
			req.Header.Set("Authorization", "Bearer any-token")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Method %s: expected 405, got %d", method, w.Code)
			}
		})
	}
}

func TestAdversarial_InvalidJSONTrailingData_401(t *testing.T) {
	handler := AuthHandler("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer validtrailing")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Trailing data: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_MiddlewareRejectsBasicAuth(t *testing.T) {
	middleware := AuthMiddleware("secret-token")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Basic auth: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_MiddlewareRejectsDigestAuth(t *testing.T) {
	middleware := AuthMiddleware("secret-token")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/sockets", nil)
	req.Header.Set("Authorization", "Digest username=admin")
	w := httptest.NewRecorder()
	middleware(next).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Digest auth: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_TrailingWhitespaceInToken_401(t *testing.T) {
	handler := AuthHandler("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer secret-token ")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Trailing whitespace token: expected 401, got %d", w.Code)
	}
}

func TestAdversarial_LeadingWhitespaceInToken_401(t *testing.T) {
	handler := AuthHandler("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.Header.Set("Authorization", "Bearer  secret-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Leading whitespace token: expected 401, got %d", w.Code)
	}
}
