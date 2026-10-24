package consentnotification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDecideAppointmentNotification(t *testing.T) {
	tests := []struct {
		name string
		body string
		want Decision
	}{
		{"granted", `{"ok":true,"data":{"granted":true}}`, Decision{Send: true, Reason: "consent is granted"}},
		{"revoked", `{"ok":true,"data":{"granted":false}}`, Decision{Reason: "consent is not granted"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v1/auth/consent/check/user-7/appointment" {
					t.Fatalf("request = %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-key" {
					t.Fatal("missing authorization")
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			c := &Client{BaseURL: srv.URL, APIKey: "test-key", HTTPClient: srv.Client()}
			got, err := Decide(context.Background(), c, Appointment{UserID: "user-7", Category: "appointment", Message: "Your visit is tomorrow."})
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("decision = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestEnvelopeIsReadBeforeStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"CONSENT_REQUIRED","message":"grant required"}}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, APIKey: "test-key", HTTPClient: srv.Client()}
	_, err := c.CheckConsent(context.Background(), "user-7", "appointment")
	if err == nil || err.Error() != "CONSENT_REQUIRED: grant required" {
		t.Fatalf("error = %v", err)
	}
}
