package main

import (
	"database/sql"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Browser state changes must originate from this application.
func sameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=(self)")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				writeError(w, 403, "Cross-site request blocked")
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Host != r.Host || !(u.Scheme == "http" || u.Scheme == "https") {
					writeError(w, 403, "Origin not allowed")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) adminUserAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	id, err := pathID(r.URL.Path, "/api/admin/users/")
	if err != nil {
		writeError(w, 400, "Bad user id")
		return
	}
	var in struct {
		Active *bool `json:"active"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Active == nil {
		writeError(w, 400, "Provide active boolean")
		return
	}
	res, err := a.db.Exec(`UPDATE users SET active=$1 WHERE id=$2 AND role<>'admin'`, *in.Active, id)
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 404, "User not found or protected admin account")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *App) adminOrderAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	id, err := pathID(r.URL.Path, "/api/admin/orders/")
	if err != nil {
		writeError(w, 400, "Bad order id")
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	previous := map[string]string{"processing": "paid", "shipped": "processing", "delivered": "shipped"}[in.Status]
	if previous == "" {
		writeError(w, 400, "Invalid status")
		return
	}
	res, err := a.db.Exec(`UPDATE orders SET status=$1 WHERE id=$2 AND status=$3 AND payment_status='simulated_paid'`, in.Status, id, previous)
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 409, "Invalid order transition")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// Add missing synthetic examples without resetting existing orders or stock.
func (a *App) seedExtras() error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	type demo struct {
		email, company, name, category string
		price                          int
	}
	demos := []demo{
		{"supplier@techline.kz", "TechLine Kazakhstan", "Compact Keyboard", "Keyboards", 9990},
		{"supplier@techline.kz", "TechLine Kazakhstan", "Office Mouse", "Mice", 3990},
		{"supplier@officehub.kz", "OfficeHub Demo", "Office Keyboard", "Keyboards", 8990},
		{"supplier@officehub.kz", "OfficeHub Demo", "Office Webcam", "Webcams", 17990},
		{"supplier@officehub.kz", "OfficeHub Demo", "Office Monitor", "Monitors", 74990},
		{"supplier@datapoint.kz", "DataPoint Demo", "Portable SSD 1 TB", "Storage", 45990},
		{"supplier@datapoint.kz", "DataPoint Demo", "Wireless Mouse", "Mice", 7990},
	}
	for _, d := range demos {
		var uid int64
		err = tx.QueryRow(`SELECT id FROM users WHERE email=$1`, d.email).Scan(&uid)
		if err == sql.ErrNoRows {
			hash, e := bcrypt.GenerateFromPassword([]byte("Supplier123!"), bcrypt.DefaultCost)
			if e != nil {
				return e
			}
			err = tx.QueryRow(`INSERT INTO users(name,email,password_hash,role) VALUES($1,$2,$3,'supplier') RETURNING id`, d.company, d.email, string(hash)).Scan(&uid)
		}
		if err != nil {
			return err
		}
		var sid int64
		err = tx.QueryRow(`SELECT id FROM suppliers WHERE user_id=$1`, uid).Scan(&sid)
		if err == sql.ErrNoRows {
			err = tx.QueryRow(`INSERT INTO suppliers(user_id,company_name,description) VALUES($1,$2,'Synthetic course demonstration supplier') RETURNING id`, uid, d.company).Scan(&sid)
		}
		if err != nil {
			return err
		}
		var exists bool
		if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM products WHERE supplier_id=$1 AND name=$2)`, sid, d.name).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		var cid int64
		if err = tx.QueryRow(`SELECT id FROM categories WHERE name=$1`, d.category).Scan(&cid); err != nil {
			return err
		}
		imageByCategory := map[string]string{"Keyboards": "keyboard.svg", "Mice": "mouse.svg", "Webcams": "webcam.svg", "Storage": "ssd.svg", "Monitors": "monitor.svg"}
		imageURL := "/assets/" + imageByCategory[d.category]
		if _, err = tx.Exec(`INSERT INTO products(supplier_id,category_id,name,description,specs,price_kzt,stock,image_url,approved) VALUES($1,$2,$3,'Synthetic accessory for the course demo','{"Use":"Office"}'::jsonb,$4,10,$5,true)`, sid, cid, d.name, strconv.Itoa(d.price), imageURL); err != nil {
			return err
		}
	}
	return tx.Commit()
}
