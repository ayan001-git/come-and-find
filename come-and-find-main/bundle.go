package main

import (
	"database/sql"
	"net/http"
	"sort"
)

// Add one unit of each quoted item. Checkout still handles the stock deduction.
func (a *App) addBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var selection struct {
		IDs []int64 `json:"ids"`
		Total int64 `json:"total_tiyn"`
	}
	if !decodeJSON(w, r, &selection) { return }
	if len(selection.IDs) < 2 || len(selection.IDs) > 9 || selection.Total < 0 {
		writeError(w, 400, "Invalid bundle")
		return
	}
	sort.Slice(selection.IDs, func(i,j int) bool { return selection.IDs[i] < selection.IDs[j] })
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil { writeError(w,500,"Database error"); return }
	defer tx.Rollback()
	uid := currentUser(r).ID
	var buyerID, seller, total int64
	if err := tx.QueryRow("SELECT id FROM users WHERE id=$1 FOR UPDATE",uid).Scan(&buyerID); err != nil {
		writeError(w,500,"Database error"); return
	}
	categories := map[int64]bool{}
	for _, id := range selection.IDs {
		var supplier, category, price int64
		var stock, quantity int
		err := tx.QueryRow(`SELECT p.supplier_id,p.category_id,(p.price_kzt*100)::bigint,p.stock,
			COALESCE((SELECT quantity FROM cart_items WHERE user_id=$2 AND product_id=p.id),0)
			FROM products p JOIN suppliers s ON s.id=p.supplier_id JOIN users u ON u.id=s.user_id
			WHERE p.id=$1 AND p.approved AND u.active FOR UPDATE OF p`,id,uid).
			Scan(&supplier,&category,&price,&stock,&quantity)
		if err == sql.ErrNoRows { writeError(w,409,"Bundle changed"); return }
		if err != nil { writeError(w,500,"Database error"); return }
		if quantity >= stock || categories[category] || (seller != 0 && seller != supplier) {
			writeError(w,409,"Bundle changed"); return
		}
		seller = supplier
		categories[category] = true
		total += price
	}
	var mixed bool
	err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM cart_items ci JOIN products p ON p.id=ci.product_id
		WHERE ci.user_id=$1 AND p.supplier_id<>$2)`,uid,seller).Scan(&mixed)
	if err != nil { writeError(w,500,"Database error"); return }
	if mixed { writeError(w,409,"Different cart supplier"); return }
	if total != selection.Total { writeError(w,409,"Bundle changed"); return }
	for _, id := range selection.IDs {
		_,err := tx.Exec(`INSERT INTO cart_items(user_id,product_id,quantity) VALUES($1,$2,1)
			ON CONFLICT(user_id,product_id) DO UPDATE SET quantity=cart_items.quantity+1`,uid,id)
		if err != nil { writeError(w,500,"Database error"); return }
	}
	if err := tx.Commit(); err != nil { writeError(w,409,"Bundle changed"); return }
	writeJSON(w,200,map[string]bool{"ok":true})
}
