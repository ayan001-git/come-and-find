package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type App struct{ db *sql.DB }

type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type ctxKey string

const userKey ctxKey = "user"

func main() {
	dsn := getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/comeandfind?sslmode=disable")
	port := getenv("PORT", "8080")
	db, err := sql.Open("postgres", dsn)
	must(err)
	defer db.Close()
	maxConnections, err := strconv.Atoi(getenv("DB_MAX_OPEN", "10"))
	must(err)
	if maxConnections < 1 {
		log.Fatal("DB_MAX_OPEN must be positive")
	}
	db.SetMaxOpenConns(maxConnections)
	db.SetMaxIdleConns(maxConnections)
	must(db.Ping())

	schema, err := os.ReadFile("schema.sql")
	must(err)
	_, err = db.Exec(string(schema))
	must(err)

	app := &App{db: db}
	must(app.seed())
	must(app.seedExtras())

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "Database unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "database": "connected"})
	})
	mux.HandleFunc("/api/auth/register", app.register)
	mux.HandleFunc("/api/auth/login", app.login)
	mux.HandleFunc("/api/auth/logout", app.logout)
	mux.Handle("/api/me", app.withUser(http.HandlerFunc(app.me)))
	mux.HandleFunc("/api/categories", app.categories)
	mux.HandleFunc("/api/products", app.products)
	mux.HandleFunc("/api/search/interpret", app.interpretSearch)
	mux.HandleFunc("/api/products/", app.productByID)
	mux.Handle("/api/cart", app.withUser(http.HandlerFunc(app.cart)))
	mux.Handle("/api/cart/", app.withUser(http.HandlerFunc(app.cartItem)))
	mux.Handle("/api/orders", app.withUser(http.HandlerFunc(app.orders)))
	mux.Handle("/api/orders/", app.withUser(http.HandlerFunc(app.orderAction)))
	mux.Handle("/api/supplier/products", app.withUser(app.requireRole("supplier", http.HandlerFunc(app.supplierProducts))))
	mux.Handle("/api/supplier/orders", app.withUser(app.requireRole("supplier", http.HandlerFunc(app.supplierOrders))))
	mux.Handle("/api/supplier/orders/", app.withUser(app.requireRole("supplier", http.HandlerFunc(app.supplierOrderAction))))
	mux.Handle("/api/admin/stats", app.withUser(app.requireRole("admin", http.HandlerFunc(app.adminStats))))
	mux.Handle("/api/admin/products", app.withUser(app.requireRole("admin", http.HandlerFunc(app.adminProducts))))
	mux.Handle("/api/admin/products/", app.withUser(app.requireRole("admin", http.HandlerFunc(app.adminProductAction))))
	mux.Handle("/api/admin/users/", app.withUser(app.requireRole("admin", http.HandlerFunc(app.adminUserAction))))
	mux.Handle("/api/admin/orders/", app.withUser(app.requireRole("admin", http.HandlerFunc(app.adminOrderAction))))
	mux.Handle("/api/admin/users", app.withUser(app.requireRole("admin", http.HandlerFunc(app.adminUsers))))
	mux.Handle("/api/admin/orders", app.withUser(app.requireRole("admin", http.HandlerFunc(app.adminOrders))))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { writeError(w, 404, "API endpoint not found") })
	mux.Handle("/", http.FileServer(http.Dir("web")))

	server := &http.Server{Addr: ":" + port, Handler: logging(sameOrigin(mux)), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	log.Printf("Come and Find is running at http://localhost:%s", port)
	log.Fatal(server.ListenAndServe())
}

func (a *App) seed() error {
	var count int
	if err := a.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	makeUser := func(name, email, password, role string) (int64, error) {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return 0, err
		}
		var id int64
		err = tx.QueryRow(`INSERT INTO users(name,email,password_hash,role) VALUES($1,$2,$3,$4) RETURNING id`, name, email, string(hash), role).Scan(&id)
		return id, err
	}
	adminID, err := makeUser("Come and Find Admin", "admin@comeandfind.kz", "Admin123!", "admin")
	_ = adminID
	if err != nil {
		return err
	}
	supplierUserID, err := makeUser("TechLine Manager", "supplier@techline.kz", "Supplier123!", "supplier")
	if err != nil {
		return err
	}
	_, err = makeUser("Demo Buyer", "buyer@example.kz", "Buyer123!", "buyer")
	if err != nil {
		return err
	}

	var supplierID int64
	err = tx.QueryRow(`INSERT INTO suppliers(user_id,company_name,description,verified) VALUES($1,$2,$3,true) RETURNING id`, supplierUserID, "TechLine Kazakhstan", "Local supplier of computer and office accessories.").Scan(&supplierID)
	if err != nil {
		return err
	}

	cats := []string{"Keyboards", "Mice", "Webcams", "Storage", "Monitors"}
	catIDs := map[string]int64{}
	for _, c := range cats {
		var id int64
		if err := tx.QueryRow(`INSERT INTO categories(name) VALUES($1) RETURNING id`, c).Scan(&id); err != nil {
			return err
		}
		catIDs[c] = id
	}

	products := []struct {
		name, cat, desc, specs, img string
		price                       float64
		stock                       int
	}{
		{"Office Keyboard K120", "Keyboards", "Reliable full-size USB keyboard for study and office work.", `{"Connection":"USB","Layout":"US/RU","Warranty":"12 months"}`, "/assets/keyboard.svg", 7990, 24},
		{"Silent Mouse M220", "Mice", "Compact wireless mouse with quiet clicks.", `{"Connection":"2.4 GHz","Battery":"AA","Warranty":"12 months"}`, "/assets/mouse.svg", 6990, 31},
		{"Full HD Webcam C270", "Webcams", "1080p-ready webcam for meetings, classes and calls.", `{"Resolution":"Full HD","Microphone":"Built-in","Connection":"USB"}`, "/assets/webcam.svg", 14990, 16},
		{"Portable SSD 500 GB", "Storage", "Fast USB-C portable storage for files and project backups.", `{"Capacity":"500 GB","Interface":"USB-C","Type":"SSD"}`, "/assets/ssd.svg", 29990, 12},
		{"24-inch IPS Monitor", "Monitors", "Full HD IPS monitor suitable for coding and office tasks.", `{"Size":"24 inch","Resolution":"1920x1080","Panel":"IPS"}`, "/assets/monitor.svg", 64990, 8},
	}
	for _, p := range products {
		_, err := tx.Exec(`INSERT INTO products(supplier_id,category_id,name,description,specs,price_kzt,stock,image_url,approved) VALUES($1,$2,$3,$4,$5::jsonb,$6,$7,$8,true)`, supplierID, catIDs[p.cat], p.name, p.desc, p.specs, p.price, p.stock, p.img)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct{ Name, Email, Password, Role, Company string }
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)
	parsedEmail, emailErr := mail.ParseAddress(in.Email)
	if in.Name == "" || len(in.Name) > 120 || emailErr != nil || (emailErr == nil && parsedEmail.Address != in.Email) || len(in.Email) > 254 || !strings.Contains(in.Email, "@") || len(in.Password) < 8 || len(in.Password) > 72 || (in.Role != "buyer" && in.Role != "supplier") {
		writeError(w, http.StatusBadRequest, "Provide name, valid email, password of at least 8 characters, and role buyer or supplier.")
		return
	}
	hash, hashErr := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if hashErr != nil {
		writeError(w, 400, "Password must contain 8-72 bytes")
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRow(`INSERT INTO users(name,email,password_hash,role) VALUES($1,$2,$3,$4) RETURNING id`, in.Name, in.Email, string(hash), in.Role).Scan(&id)
	if err != nil {
		writeError(w, 409, "Email already exists")
		return
	}
	if in.Role == "supplier" {
		company := strings.TrimSpace(in.Company)
		if company == "" {
			company = in.Name + " Store"
		}
		if _, err := tx.Exec(`INSERT INTO suppliers(user_id,company_name,description,verified) VALUES($1,$2,'New supplier awaiting verification',false)`, id, company); err != nil {
			writeError(w, 500, "Could not create supplier profile")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, 500, "Database error")
		return
	}
	user := User{ID: id, Name: in.Name, Email: in.Email, Role: in.Role}
	if !a.createSession(w, user.ID) {
		return
	}
	writeJSON(w, 201, user)
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct{ Email, Password string }
	if !decodeJSON(w, r, &in) {
		return
	}
	var u User
	var hash string
	var active bool
	err := a.db.QueryRow(`SELECT id,name,email,role,password_hash,active FROM users WHERE lower(email)=lower($1)`, strings.TrimSpace(in.Email)).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &hash, &active)
	if err != nil || !active || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		writeError(w, 401, "Invalid email or password")
		return
	}
	if !a.createSession(w, u.ID) {
		return
	}
	writeJSON(w, 200, u)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if c, err := r.Cookie("cf_session"); err == nil {
		if _, err := a.db.Exec(`DELETE FROM sessions WHERE token=$1`, c.Value); err != nil {
			writeError(w, 500, "Logout failed; retry")
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: "cf_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: getenv("COOKIE_SECURE", "false") == "true"})
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *App) me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, 200, currentUser(r))
}

func (a *App) categories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	rows, err := a.db.Query(`SELECT id,name FROM categories ORDER BY name`)
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			writeError(w, 500, "Database read error")
			return
		}
		out = append(out, map[string]any{"id": id, "name": name})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "Database read error")
		return
	}
	writeJSON(w, 200, out)
}

func (a *App) products(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	intent, err := parseSearch(r.URL.Query().Get("search"))
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	cat := r.URL.Query().Get("category")
	query := `SELECT p.id,p.name,p.description,p.price_kzt,p.stock,p.image_url,c.name,s.company_name,s.id
              FROM products p JOIN categories c ON c.id=p.category_id JOIN suppliers s ON s.id=p.supplier_id
              WHERE p.approved=true AND EXISTS(SELECT 1 FROM users su WHERE su.id=s.user_id AND su.active=true)`
	args := []any{}
	if intent.Category != "" {
		args = append(args, intent.Category)
		query += fmt.Sprintf(" AND c.name=$%d", len(args))
	}
	for _, term := range intent.Terms {
		args = append(args, "%"+term+"%")
		n := len(args)
		query += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.description ILIKE $%d OR p.specs::text ILIKE $%d OR c.name ILIKE $%d)", n, n, n, n)
	}
	if cat != "" {
		cid, err := strconv.ParseInt(cat, 10, 64)
		if err != nil || cid < 1 {
			writeError(w, 400, "Invalid category")
			return
		}
		args = append(args, cid)
		query += fmt.Sprintf(" AND c.id=$%d", len(args))
	}
	var bounds [2]float64
	var present [2]bool
	for i, key := range []string{"min_price", "max_price"} {
		raw := r.URL.Query().Get(key)
		if raw == "" {
			v := intent.Min
			if i == 1 {
				v = intent.Max
			}
			if v != nil {
				raw = strconv.FormatFloat(*v, 'f', -1, 64)
			}
		}
		if raw != "" {
			v, err := strconv.ParseFloat(raw, 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
				writeError(w, 400, "Invalid price range")
				return
			}
			bounds[i] = v
			present[i] = true
			op := ">="
			if i == 1 {
				op = "<="
			}
			args = append(args, v)
			query += fmt.Sprintf(" AND p.price_kzt%s$%d", op, len(args))
		}
	}
	if present[0] && present[1] && bounds[0] > bounds[1] {
		writeError(w, 400, "Minimum price exceeds maximum")
		return
	}
	query += ` ORDER BY p.id`
	rows, err := a.db.Query(query, args...)
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, sid int64
		var name, desc, img, cn, sn string
		var price float64
		var stock int
		if err := rows.Scan(&id, &name, &desc, &price, &stock, &img, &cn, &sn, &sid); err != nil {
			writeError(w, 500, "Database read error")
			return
		}
		out = append(out, map[string]any{"id": id, "name": name, "description": desc, "price_kzt": price, "stock": stock, "image_url": img, "category": cn, "supplier": sn, "supplier_id": sid})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "Database read error")
		return
	}
	writeJSON(w, 200, out)
}

func (a *App) productByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	id, err := pathID(r.URL.Path, "/api/products/")
	if err != nil {
		writeError(w, 400, "Bad product id")
		return
	}
	var name, desc, img, cat, supplier, sdesc string
	var specs []byte
	var price float64
	var stock int
	var verified bool
	var sid int64
	err = a.db.QueryRow(`SELECT p.name,p.description,p.specs,p.price_kzt,p.stock,p.image_url,c.name,s.company_name,s.description,s.verified,s.id FROM products p JOIN categories c ON c.id=p.category_id JOIN suppliers s ON s.id=p.supplier_id WHERE p.id=$1 AND p.approved=true AND EXISTS(SELECT 1 FROM users su WHERE su.id=s.user_id AND su.active=true)`, id).Scan(&name, &desc, &specs, &price, &stock, &img, &cat, &supplier, &sdesc, &verified, &sid)
	if err == sql.ErrNoRows {
		writeError(w, 404, "Product not found")
		return
	}
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	var specsMap map[string]any
	json.Unmarshal(specs, &specsMap)
	writeJSON(w, 200, map[string]any{"id": id, "name": name, "description": desc, "specs": specsMap, "price_kzt": price, "stock": stock, "image_url": img, "category": cat, "supplier": map[string]any{"id": sid, "company_name": supplier, "description": sdesc, "verified": verified}})
}

func (a *App) cart(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u.Role != "buyer" {
		writeError(w, 403, "Buyer account required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := a.db.Query(`SELECT p.id,p.name,p.price_kzt,p.image_url,ci.quantity,p.stock FROM cart_items ci JOIN products p ON p.id=ci.product_id WHERE ci.user_id=$1 ORDER BY p.name`, u.ID)
		if err != nil {
			writeError(w, 500, "Database error")
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		total := 0.0
		for rows.Next() {
			var id int64
			var name, img string
			var price float64
			var qty, stock int
			if err := rows.Scan(&id, &name, &price, &img, &qty, &stock); err != nil {
				writeError(w, 500, "Database read error")
				return
			}
			total += price * float64(qty)
			out = append(out, map[string]any{"id": id, "name": name, "price_kzt": price, "image_url": img, "quantity": qty, "stock": stock})
		}
		if err := rows.Err(); err != nil {
			writeError(w, 500, "Database read error")
			return
		}
		writeJSON(w, 200, map[string]any{"items": out, "total_kzt": total})
	case http.MethodPost, http.MethodPatch:
		var in struct {
			ProductID int64 `json:"product_id"`
			Quantity  int   `json:"quantity"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		if in.ProductID < 1 || in.Quantity < 1 || in.Quantity > 10000 {
			writeError(w, 400, "Quantity must be 1-10000")
			return
		}
		tx, err := a.db.BeginTx(r.Context(), nil)
		if err != nil {
			writeError(w, 500, "Database error")
			return
		}
		defer tx.Rollback()
		var uid int64
		if err = tx.QueryRow(`SELECT id FROM users WHERE id=$1 FOR UPDATE`, u.ID).Scan(&uid); err != nil {
			writeError(w, 500, "Database error")
			return
		}
		var stock int
		var approved, active bool
		var sid int64
		err = tx.QueryRow(`SELECT p.stock,p.approved,p.supplier_id,su.active FROM products p JOIN suppliers s ON s.id=p.supplier_id JOIN users su ON su.id=s.user_id WHERE p.id=$1 FOR UPDATE OF p`, in.ProductID).Scan(&stock, &approved, &sid, &active)
		if err == sql.ErrNoRows || !approved || !active {
			writeError(w, 404, "Product unavailable")
			return
		}
		if err != nil {
			writeError(w, 500, "Database error")
			return
		}
		var mixed bool
		if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM cart_items ci JOIN products p ON p.id=ci.product_id WHERE ci.user_id=$1 AND p.supplier_id<>$2)`, u.ID, sid).Scan(&mixed); err != nil {
			writeError(w, 500, "Database error")
			return
		}
		if mixed {
			writeError(w, 409, "One supplier per cart. Complete or clear your current cart first.")
			return
		}
		qty := in.Quantity
		if r.Method == http.MethodPost {
			var existing int
			err = tx.QueryRow(`SELECT quantity FROM cart_items WHERE user_id=$1 AND product_id=$2`, u.ID, in.ProductID).Scan(&existing)
			if err != nil && err != sql.ErrNoRows {
				writeError(w, 500, "Database error")
				return
			}
			qty += existing
		}
		if qty > stock {
			writeError(w, 400, "Not enough stock")
			return
		}
		if _, err = tx.Exec(`INSERT INTO cart_items(user_id,product_id,quantity) VALUES($1,$2,$3) ON CONFLICT(user_id,product_id) DO UPDATE SET quantity=EXCLUDED.quantity`, u.ID, in.ProductID, qty); err != nil {
			writeError(w, 500, "Database error")
			return
		}
		if err = tx.Commit(); err != nil {
			writeError(w, 409, "Cart changed; retry")
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	default:
		methodNotAllowed(w)
	}
}

func (a *App) cartItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	u := currentUser(r)
	id, err := pathID(r.URL.Path, "/api/cart/")
	if err != nil {
		writeError(w, 400, "Bad product id")
		return
	}
	if u.Role != "buyer" {
		writeError(w, 403, "Buyer account required")
		return
	}
	if _, err := a.db.Exec(`DELETE FROM cart_items WHERE user_id=$1 AND product_id=$2`, u.ID, id); err != nil {
		writeError(w, 500, "Database error")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *App) orders(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u.Role != "buyer" {
		writeError(w, 403, "Buyer account required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := a.db.Query(`SELECT id,status,payment_status,total_kzt,created_at FROM orders WHERE buyer_id=$1 ORDER BY id DESC`, u.ID)
		if err != nil {
			writeError(w, 500, "Database error")
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id int64
			var status, pay string
			var total float64
			var created time.Time
			if err := rows.Scan(&id, &status, &pay, &total, &created); err != nil {
				writeError(w, 500, "Database read error")
				return
			}
			out = append(out, map[string]any{"id": id, "status": status, "payment_status": pay, "total_kzt": total, "created_at": created})
		}
		if err := rows.Err(); err != nil {
			writeError(w, 500, "Database read error")
			return
		}
		writeJSON(w, 200, out)
	case http.MethodPost:
		tx, err := a.db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			writeError(w, 500, "Database error")
			return
		}
		defer tx.Rollback()
		var lockedUser int64
		if err := tx.QueryRow(`SELECT id FROM users WHERE id=$1 FOR UPDATE`, u.ID).Scan(&lockedUser); err != nil {
			writeError(w, 409, "Cart changed; retry")
			return
		}
		rows, err := tx.Query(`SELECT p.id,p.supplier_id,p.name,p.price_kzt,p.stock,ci.quantity,p.approved,su.active FROM cart_items ci JOIN products p ON p.id=ci.product_id JOIN suppliers s ON s.id=p.supplier_id JOIN users su ON su.id=s.user_id WHERE ci.user_id=$1 ORDER BY p.id FOR UPDATE OF p`, u.ID)
		if err != nil {
			writeError(w, 500, "Database error")
			return
		}
		type item struct {
			id, supplier int64
			name         string
			price        float64
			stock, qty   int
		}
		var items []item
		total := 0.0
		for rows.Next() {
			var it item
			var approved, active bool
			if err := rows.Scan(&it.id, &it.supplier, &it.name, &it.price, &it.stock, &it.qty, &approved, &active); err != nil {
				rows.Close()
				writeError(w, 500, "Database read error")
				return
			}
			if !approved || !active || it.qty < 1 || it.qty > it.stock {
				rows.Close()
				writeError(w, 400, "One item no longer has enough stock")
				return
			}
			if len(items) > 0 && items[0].supplier != it.supplier {
				rows.Close()
				writeError(w, 409, "One supplier per order; clear mixed cart")
				return
			}
			items = append(items, it)
			total += it.price * float64(it.qty)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			writeError(w, 500, "Database read error")
			return
		}
		rows.Close()
		if len(items) == 0 {
			writeError(w, 400, "Cart is empty")
			return
		}
		var oid int64
		if err := tx.QueryRow(`INSERT INTO orders(buyer_id,total_kzt) VALUES($1,$2) RETURNING id`, u.ID, total).Scan(&oid); err != nil {
			writeError(w, 500, "Database error")
			return
		}
		for _, it := range items {
			if _, err := tx.Exec(`INSERT INTO order_items(order_id,product_id,supplier_id,product_name,unit_price_kzt,quantity) VALUES($1,$2,$3,$4,$5,$6)`, oid, it.id, it.supplier, it.name, it.price, it.qty); err != nil {
				writeError(w, 500, "Database error")
				return
			}
			if _, err := tx.Exec(`UPDATE products SET stock=stock-$1 WHERE id=$2`, it.qty, it.id); err != nil {
				writeError(w, 500, "Database error")
				return
			}
		}
		if _, err := tx.Exec(`DELETE FROM cart_items WHERE user_id=$1`, u.ID); err != nil {
			writeError(w, 500, "Database error")
			return
		}
		if err := tx.Commit(); err != nil {
			writeError(w, 409, "Order changed; try again")
			return
		}
		writeJSON(w, 201, map[string]any{"id": oid, "total_kzt": total, "status": "created"})
	default:
		methodNotAllowed(w)
	}
}

func (a *App) orderAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	u := currentUser(r)
	if u.Role != "buyer" {
		writeError(w, 403, "Buyer account required")
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/orders/"), "/"), "/")
	if len(parts) != 2 || parts[1] != "pay" {
		writeError(w, 404, "Not found")
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, 400, "Bad order id")
		return
	}
	res, err := a.db.Exec(`UPDATE orders SET payment_status='simulated_paid',status='paid' WHERE id=$1 AND buyer_id=$2 AND status='created'`, id, u.ID)
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 404, "Order not found or already paid")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "message": "Simulated payment completed"})
}

func (a *App) supplierProducts(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var sid int64
	if err := a.db.QueryRow(`SELECT id FROM suppliers WHERE user_id=$1`, u.ID).Scan(&sid); err != nil {
		writeError(w, 403, "Supplier profile missing")
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := a.db.Query(`SELECT p.id,p.name,c.name,p.price_kzt,p.stock,p.approved FROM products p JOIN categories c ON c.id=p.category_id WHERE p.supplier_id=$1 ORDER BY p.id DESC`, sid)
		if err != nil {
			writeError(w, 500, "Database error")
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id int64
			var name, cat string
			var price float64
			var stock int
			var approved bool
			if err := rows.Scan(&id, &name, &cat, &price, &stock, &approved); err != nil {
				writeError(w, 500, "Database read error")
				return
			}
			out = append(out, map[string]any{"id": id, "name": name, "category": cat, "price_kzt": price, "stock": stock, "approved": approved})
		}
		if err := rows.Err(); err != nil {
			writeError(w, 500, "Database read error")
			return
		}
		writeJSON(w, 200, out)
	case http.MethodPost:
		var in struct {
			Name        string            `json:"name"`
			Description string            `json:"description"`
			ImageURL    string            `json:"image_url"`
			CategoryID  int64             `json:"category_id"`
			Price       float64           `json:"price_kzt"`
			Stock       int               `json:"stock"`
			Specs       map[string]string `json:"specs"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		if strings.TrimSpace(in.Name) == "" || len([]rune(in.Name)) > 160 || len([]rune(in.Description)) > 2000 || len(in.ImageURL) > 255 || len(in.Specs) > 30 || in.CategoryID == 0 || in.Price < 0 || in.Price > 9999999999.99 || math.IsNaN(in.Price) || math.IsInf(in.Price, 0) || in.Stock < 0 || in.Stock > 1000000 {
			writeError(w, 400, "Invalid product")
			return
		}
		var categoryExists bool
		if err := a.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM categories WHERE id=$1)`, in.CategoryID).Scan(&categoryExists); err != nil {
			writeError(w, 500, "Database error")
			return
		}
		if !categoryExists {
			writeError(w, 400, "Unknown category")
			return
		}
		if in.ImageURL != "" && (!strings.HasPrefix(in.ImageURL, "/assets/") || strings.Contains(in.ImageURL, "..")) {
			writeError(w, 400, "Use a local /assets/ image")
			return
		}
		if in.Specs == nil {
			in.Specs = map[string]string{}
		}
		if in.ImageURL == "" {
			in.ImageURL = "/assets/product.svg"
		}
		specs, _ := json.Marshal(in.Specs)
		var id int64
		err := a.db.QueryRow(`INSERT INTO products(supplier_id,category_id,name,description,specs,price_kzt,stock,image_url,approved) VALUES($1,$2,$3,$4,$5::jsonb,$6,$7,$8,false) RETURNING id`, sid, in.CategoryID, in.Name, in.Description, specs, in.Price, in.Stock, in.ImageURL).Scan(&id)
		if err != nil {
			writeError(w, 500, "Database error")
			return
		}
		writeJSON(w, 201, map[string]any{"id": id, "approved": false})
	default:
		methodNotAllowed(w)
	}
}

func (a *App) supplierOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	u := currentUser(r)
	var sid int64
	if err := a.db.QueryRow(`SELECT id FROM suppliers WHERE user_id=$1`, u.ID).Scan(&sid); err != nil {
		writeError(w, 403, "Supplier profile missing")
		return
	}
	rows, err := a.db.Query(`SELECT DISTINCT o.id,o.status,o.payment_status,o.total_kzt,o.created_at,u.name FROM orders o JOIN order_items oi ON oi.order_id=o.id JOIN users u ON u.id=o.buyer_id WHERE oi.supplier_id=$1 ORDER BY o.id DESC`, sid)
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var status, pay, buyer string
		var total float64
		var created time.Time
		if err := rows.Scan(&id, &status, &pay, &total, &created, &buyer); err != nil {
			writeError(w, 500, "Database read error")
			return
		}
		out = append(out, map[string]any{"id": id, "status": status, "payment_status": pay, "total_kzt": total, "created_at": created, "buyer": buyer})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "Database read error")
		return
	}
	writeJSON(w, 200, out)
}

func (a *App) supplierOrderAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	u := currentUser(r)
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/supplier/orders/"), "/"), "/")
	if len(parts) != 2 || parts[1] != "status" {
		writeError(w, 404, "Not found")
		return
	}
	oid, err := strconv.ParseInt(parts[0], 10, 64)
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
	allowed := map[string]bool{"processing": true, "shipped": true, "delivered": true}
	if !allowed[in.Status] {
		writeError(w, 400, "Invalid status")
		return
	}
	var sid int64
	if err := a.db.QueryRow(`SELECT id FROM suppliers WHERE user_id=$1`, u.ID).Scan(&sid); err != nil {
		writeError(w, 403, "Supplier profile missing")
		return
	}
	res, err := a.db.Exec(`UPDATE orders SET status=$1 WHERE id=$2 AND EXISTS(SELECT 1 FROM order_items WHERE order_id=$2 AND supplier_id=$3) AND NOT EXISTS(SELECT 1 FROM order_items WHERE order_id=$2 AND supplier_id<>$3) AND payment_status='simulated_paid' AND status=$4`, in.Status, oid, sid, map[string]string{"processing": "paid", "shipped": "processing", "delivered": "shipped"}[in.Status])
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 404, "Order unavailable or invalid status transition")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *App) adminStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	var users, products, pending, orders int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
		writeError(w, 500, "Database error")
		return
	}
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&products); err != nil {
		writeError(w, 500, "Database error")
		return
	}
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM products WHERE approved=false`).Scan(&pending); err != nil {
		writeError(w, 500, "Database error")
		return
	}
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders); err != nil {
		writeError(w, 500, "Database error")
		return
	}
	writeJSON(w, 200, map[string]any{"users": users, "products": products, "pending_products": pending, "orders": orders})
}
func (a *App) adminProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	rows, err := a.db.Query(`SELECT p.id,p.name,s.company_name,c.name,p.price_kzt,p.stock,p.approved FROM products p JOIN suppliers s ON s.id=p.supplier_id JOIN categories c ON c.id=p.category_id ORDER BY p.approved,p.id DESC`)
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var name, sup, cat string
		var price float64
		var stock int
		var approved bool
		if err := rows.Scan(&id, &name, &sup, &cat, &price, &stock, &approved); err != nil {
			writeError(w, 500, "Database read error")
			return
		}
		out = append(out, map[string]any{"id": id, "name": name, "supplier": sup, "category": cat, "price_kzt": price, "stock": stock, "approved": approved})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "Database read error")
		return
	}
	writeJSON(w, 200, out)
}
func (a *App) adminProductAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/products/"), "/"), "/")
	if len(parts) != 2 || (parts[1] != "approve" && parts[1] != "withdraw" && parts[1] != "reject") {
		writeError(w, 404, "Not found")
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, 400, "Bad id")
		return
	}
	res, err := a.db.Exec(`UPDATE products SET approved=$2 WHERE id=$1`, id, parts[1] == "approve")
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 404, "Not found")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
func (a *App) adminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	rows, err := a.db.Query(`SELECT id,name,email,role,active,created_at FROM users ORDER BY id`)
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var name, email, role string
		var active bool
		var created time.Time
		if err := rows.Scan(&id, &name, &email, &role, &active, &created); err != nil {
			writeError(w, 500, "Database read error")
			return
		}
		out = append(out, map[string]any{"id": id, "name": name, "email": email, "role": role, "active": active, "created_at": created})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "Database read error")
		return
	}
	writeJSON(w, 200, out)
}

func (a *App) adminOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	rows, err := a.db.Query(`SELECT o.id,u.name,o.total_kzt,o.payment_status,o.status,o.created_at FROM orders o JOIN users u ON u.id=o.buyer_id ORDER BY o.id DESC`)
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var buyer, pay, status string
		var total float64
		var created time.Time
		if err := rows.Scan(&id, &buyer, &total, &pay, &status, &created); err != nil {
			writeError(w, 500, "Database read error")
			return
		}
		out = append(out, map[string]any{"id": id, "buyer": buyer, "total_kzt": total, "payment_status": pay, "status": status, "created_at": created})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "Database read error")
		return
	}
	writeJSON(w, 200, out)
}

func (a *App) withUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("cf_session")
		if err != nil {
			writeError(w, 401, "Login required")
			return
		}
		var u User
		err = a.db.QueryRow(`SELECT u.id,u.name,u.email,u.role FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token=$1 AND s.expires_at>NOW() AND u.active=true`, c.Value).Scan(&u.ID, &u.Name, &u.Email, &u.Role)
		if err != nil {
			writeError(w, 401, "Session expired")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	})
}
func (a *App) requireRole(role string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if currentUser(r).Role != role {
			writeError(w, 403, "Access denied")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func currentUser(r *http.Request) User { u, _ := r.Context().Value(userKey).(User); return u }
func (a *App) createSession(w http.ResponseWriter, userID int64) bool {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		writeError(w, 500, "Session generation failed")
		return false
	}
	token := hex.EncodeToString(buf)
	expires := time.Now().Add(7 * 24 * time.Hour)
	if _, err := a.db.Exec(`INSERT INTO sessions(token,user_id,expires_at) VALUES($1,$2,$3)`, token, userID, expires); err != nil {
		writeError(w, 500, "Session creation failed")
		return false
	}
	http.SetCookie(w, &http.Cookie{Name: "cf_session", Value: token, Path: "/", Expires: expires, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: getenv("COOKIE_SECURE", "false") == "true"})
	return true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		writeError(w, 400, "Invalid JSON")
		return false
	}
	if err := d.Decode(new(any)); err != io.EOF {
		writeError(w, 400, "Only one JSON object is allowed")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
func methodNotAllowed(w http.ResponseWriter) { writeError(w, 405, "Method not allowed") }
func pathID(path, prefix string) (int64, error) {
	s := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if strings.Contains(s, "/") {
		return 0, errors.New("bad path")
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("bad id")
	}
	return id, nil
}
func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
