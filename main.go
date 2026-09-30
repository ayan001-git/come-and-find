package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
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
	must(db.Ping())

	schema, err := os.ReadFile("schema.sql")
	must(err)
	_, err = db.Exec(string(schema))
	must(err)

	app := &App{db: db}
	must(app.seed())

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/register", app.register)
	mux.HandleFunc("/api/auth/login", app.login)
	mux.HandleFunc("/api/auth/logout", app.logout)
	mux.Handle("/api/me", app.withUser(http.HandlerFunc(app.me)))
	mux.HandleFunc("/api/categories", app.categories)
	mux.HandleFunc("/api/products", app.products)
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
	mux.Handle("/api/admin/users", app.withUser(app.requireRole("admin", http.HandlerFunc(app.adminUsers))))
	mux.Handle("/api/admin/orders", app.withUser(app.requireRole("admin", http.HandlerFunc(app.adminOrders))))
	mux.Handle("/", http.FileServer(http.Dir("web")))

	server := &http.Server{Addr: ":" + port, Handler: logging(mux), ReadHeaderTimeout: 5 * time.Second}
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
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		var id int64
		err := tx.QueryRow(`INSERT INTO users(name,email,password_hash,role) VALUES($1,$2,$3,$4) RETURNING id`, name, email, string(hash), role).Scan(&id)
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
	if in.Name == "" || !strings.Contains(in.Email, "@") || len(in.Password) < 8 || (in.Role != "buyer" && in.Role != "supplier") {
		writeError(w, http.StatusBadRequest, "Provide name, valid email, password of at least 8 characters, and role buyer or supplier.")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
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
	a.createSession(w, user.ID)
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
	a.createSession(w, u.ID)
	writeJSON(w, 200, u)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if c, err := r.Cookie("cf_session"); err == nil {
		a.db.Exec(`DELETE FROM sessions WHERE token=$1`, c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "cf_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *App) me(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, currentUser(r)) }

func (a *App) categories(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(`SELECT id,name FROM categories ORDER BY name`)
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int64
		var name string
		rows.Scan(&id, &name)
		out = append(out, map[string]any{"id": id, "name": name})
	}
	writeJSON(w, 200, out)
}

func (a *App) products(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	q := `%` + strings.TrimSpace(r.URL.Query().Get("search")) + `%`
	cat := r.URL.Query().Get("category")
	query := `SELECT p.id,p.name,p.description,p.price_kzt,p.stock,p.image_url,c.name,s.company_name,s.id
              FROM products p JOIN categories c ON c.id=p.category_id JOIN suppliers s ON s.id=p.supplier_id
              WHERE p.approved=true AND (p.name ILIKE $1 OR p.description ILIKE $1)`
	args := []any{q}
	if cat != "" {
		query += ` AND c.id=$2`
		args = append(args, cat)
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
		rows.Scan(&id, &name, &desc, &price, &stock, &img, &cn, &sn, &sid)
		out = append(out, map[string]any{"id": id, "name": name, "description": desc, "price_kzt": price, "stock": stock, "image_url": img, "category": cn, "supplier": sn, "supplier_id": sid})
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
	err = a.db.QueryRow(`SELECT p.name,p.description,p.specs,p.price_kzt,p.stock,p.image_url,c.name,s.company_name,s.description,s.verified,s.id FROM products p JOIN categories c ON c.id=p.category_id JOIN suppliers s ON s.id=p.supplier_id WHERE p.id=$1 AND p.approved=true`, id).Scan(&name, &desc, &specs, &price, &stock, &img, &cat, &supplier, &sdesc, &verified, &sid)
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
			rows.Scan(&id, &name, &price, &img, &qty, &stock)
			total += price * float64(qty)
			out = append(out, map[string]any{"id": id, "name": name, "price_kzt": price, "image_url": img, "quantity": qty, "stock": stock})
		}
		writeJSON(w, 200, map[string]any{"items": out, "total_kzt": total})
	case http.MethodPost:
		var in struct {
			ProductID int64 `json:"product_id"`
			Quantity  int   `json:"quantity"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		if in.Quantity < 1 {
			in.Quantity = 1
		}
		var stock int
		var approved bool
		if err := a.db.QueryRow(`SELECT stock,approved FROM products WHERE id=$1`, in.ProductID).Scan(&stock, &approved); err != nil || !approved {
			writeError(w, 404, "Product not found")
			return
		}
		if in.Quantity > stock {
			writeError(w, 400, "Not enough stock")
			return
		}
		_, err := a.db.Exec(`INSERT INTO cart_items(user_id,product_id,quantity) VALUES($1,$2,$3) ON CONFLICT(user_id,product_id) DO UPDATE SET quantity=LEAST($3,(SELECT stock FROM products WHERE id=$2))`, u.ID, in.ProductID, in.Quantity)
		if err != nil {
			writeError(w, 500, "Database error")
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
	a.db.Exec(`DELETE FROM cart_items WHERE user_id=$1 AND product_id=$2`, u.ID, id)
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
			rows.Scan(&id, &status, &pay, &total, &created)
			out = append(out, map[string]any{"id": id, "status": status, "payment_status": pay, "total_kzt": total, "created_at": created})
		}
		writeJSON(w, 200, out)
	case http.MethodPost:
		tx, err := a.db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			writeError(w, 500, "Database error")
			return
		}
		defer tx.Rollback()
		rows, err := tx.Query(`SELECT p.id,p.supplier_id,p.name,p.price_kzt,p.stock,ci.quantity FROM cart_items ci JOIN products p ON p.id=ci.product_id WHERE ci.user_id=$1 AND p.approved=true FOR UPDATE OF p`, u.ID)
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
			rows.Scan(&it.id, &it.supplier, &it.name, &it.price, &it.stock, &it.qty)
			if it.qty > it.stock {
				rows.Close()
				writeError(w, 400, "One item no longer has enough stock")
				return
			}
			items = append(items, it)
			total += it.price * float64(it.qty)
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
		tx.Exec(`DELETE FROM cart_items WHERE user_id=$1`, u.ID)
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
			rows.Scan(&id, &name, &cat, &price, &stock, &approved)
			out = append(out, map[string]any{"id": id, "name": name, "category": cat, "price_kzt": price, "stock": stock, "approved": approved})
		}
		writeJSON(w, 200, out)
	case http.MethodPost:
		var in struct {
			Name, Description, ImageURL string
			CategoryID                  int64             `json:"category_id"`
			Price                       float64           `json:"price_kzt"`
			Stock                       int               `json:"stock"`
			Specs                       map[string]string `json:"specs"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		if strings.TrimSpace(in.Name) == "" || in.CategoryID == 0 || in.Price < 0 || in.Stock < 0 {
			writeError(w, 400, "Invalid product")
			return
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
		rows.Scan(&id, &status, &pay, &total, &created, &buyer)
		out = append(out, map[string]any{"id": id, "status": status, "payment_status": pay, "total_kzt": total, "created_at": created, "buyer": buyer})
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
	a.db.QueryRow(`SELECT id FROM suppliers WHERE user_id=$1`, u.ID).Scan(&sid)
	res, err := a.db.Exec(`UPDATE orders SET status=$1 WHERE id=$2 AND EXISTS(SELECT 1 FROM order_items WHERE order_id=$2 AND supplier_id=$3) AND payment_status='simulated_paid'`, in.Status, oid, sid)
	if err != nil {
		writeError(w, 500, "Database error")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 404, "Order not found or not paid")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *App) adminStats(w http.ResponseWriter, r *http.Request) {
	var users, products, pending, orders int
	a.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users)
	a.db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&products)
	a.db.QueryRow(`SELECT COUNT(*) FROM products WHERE approved=false`).Scan(&pending)
	a.db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders)
	writeJSON(w, 200, map[string]any{"users": users, "products": products, "pending_products": pending, "orders": orders})
}
func (a *App) adminProducts(w http.ResponseWriter, r *http.Request) {
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
		rows.Scan(&id, &name, &sup, &cat, &price, &stock, &approved)
		out = append(out, map[string]any{"id": id, "name": name, "supplier": sup, "category": cat, "price_kzt": price, "stock": stock, "approved": approved})
	}
	writeJSON(w, 200, out)
}
func (a *App) adminProductAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/products/"), "/"), "/")
	if len(parts) != 2 || parts[1] != "approve" {
		writeError(w, 404, "Not found")
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, 400, "Bad id")
		return
	}
	res, err := a.db.Exec(`UPDATE products SET approved=true WHERE id=$1`, id)
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
		rows.Scan(&id, &name, &email, &role, &active, &created)
		out = append(out, map[string]any{"id": id, "name": name, "email": email, "role": role, "active": active, "created_at": created})
	}
	writeJSON(w, 200, out)
}

func (a *App) adminOrders(w http.ResponseWriter, r *http.Request) {
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
		rows.Scan(&id, &buyer, &total, &pay, &status, &created)
		out = append(out, map[string]any{"id": id, "buyer": buyer, "total_kzt": total, "payment_status": pay, "status": status, "created_at": created})
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
func (a *App) createSession(w http.ResponseWriter, userID int64) {
	buf := make([]byte, 32)
	rand.Read(buf)
	token := hex.EncodeToString(buf)
	expires := time.Now().Add(7 * 24 * time.Hour)
	a.db.Exec(`INSERT INTO sessions(token,user_id,expires_at) VALUES($1,$2,$3)`, token, userID, expires)
	http.SetCookie(w, &http.Cookie{Name: "cf_session", Value: token, Path: "/", Expires: expires, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := d.Decode(v); err != nil {
		writeError(w, 400, "Invalid JSON")
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
	return strconv.ParseInt(s, 10, 64)
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
