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

// Add a realistic, repeatable demo catalogue without resetting existing orders or stock.
func (a *App) seedExtras() error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	type supplier struct{ email, company, description string }
	suppliers := map[string]supplier{
		"techline": {"supplier@techline.kz", "TechLine Kazakhstan", "Демонстрационный продавец компьютерной периферии и оборудования."},
		"officehub": {"supplier@officehub.kz", "OfficeHub Demo", "Демонстрационный магазин для офиса, учёбы и видеосвязи."},
		"datapoint": {"supplier@datapoint.kz", "DataPoint Demo", "Демонстрационный продавец накопителей и сетевого оборудования."},
		"gadgethouse": {"supplier@gadgethouse.kz", "GadgetHouse Demo", "Демонстрационный магазин аксессуаров и аудиотехники."},
	}
	categories := []string{"Keyboards", "Mice", "Webcams", "Storage", "Monitors", "Headsets", "Networking", "Accessories", "Power"}
	categoryIDs := make(map[string]int64, len(categories))
	for _, category := range categories {
		if _, err = tx.Exec(`INSERT INTO categories(name) VALUES($1) ON CONFLICT(name) DO NOTHING`, category); err != nil { return err }
		if err = tx.QueryRow(`SELECT id FROM categories WHERE name=$1`, category).Scan(&categoryIDs[category]); err != nil { return err }
	}

	legacyPhotos := map[string]string{
		"Office Keyboard K120": "https://www.jbhifi.com.au/cdn/shop/products/394698-Product-0-I_92f5e9a0-1bff-47f3-a08d-5b92c3fdbbe4_1024x1024.png",
		"Silent Mouse M220": "https://biuropaczka.pl/media/catalog/product/6/f/6f65b020-9501-499d-be0a-6230a07e9210.jpg",
		"Full HD Webcam C270": "https://mercury.vtexassets.com/arquivos/ids/21723812/-SERT206012101.jpg?v=638937214185100000",
		"Portable SSD 500 GB": "https://thumb.pccomponentes.com/w-530-530/articles/75/754958/1160-hp-p500-ssd-externo-500gb-usb-c-31-negro.jpg",
		"24-inch IPS Monitor": "https://neom.kz/upload/iblock/e53/rjpdatre81gfknm3y2y12fafqtlfhn99.jpg",
		"Compact Keyboard": "https://resource.logitech.com/e_trim/w_700%2Ch_600%2Cc_pad%2Cq_auto%2Cf_auto%2Cdpr_1.0/b_white/content/dam/logitech/en/products/keyboards/pebble-keys-2-k380s/gallery/pebble-keys-2-k380s-top-tonal-graphite-gallery-us.png",
		"Office Mouse": "https://bizweb.dktcdn.net/100/497/222/products/6988-chu-t-logitech-m185-wireless-a5498d7f-d936-49ce-94bf-262ce6c9953c.jpg?v=1701145770907",
		"Office Keyboard": "https://www.jbhifi.com.au/cdn/shop/products/394698-Product-0-I_92f5e9a0-1bff-47f3-a08d-5b92c3fdbbe4_1024x1024.png",
		"Office Webcam": "https://www.cecomsa.com/web/image/product.template/24905/image_1024?unique=3637dfc",
		"Office Monitor": "https://neom.kz/upload/iblock/e53/rjpdatre81gfknm3y2y12fafqtlfhn99.jpg",
		"Portable SSD 1 TB": "https://paksell.pk/cdn/shop/products/samsung-t7-portable-ssd-1tb-usb32-gen-2-type-c-external-ssd-438520.jpg?v=1691492418&width=800",
		"Wireless Mouse": "https://bizweb.dktcdn.net/100/497/222/products/6988-chu-t-logitech-m185-wireless-a5498d7f-d936-49ce-94bf-262ce6c9953c.jpg?v=1701145770907",
	}
	for name, imageURL := range legacyPhotos {
		if _, err = tx.Exec(`UPDATE products SET image_url=$1 WHERE name=$2 AND image_url LIKE '/assets/%'`, imageURL, name); err != nil { return err }
	}

	// Values are illustrative KZT prices and stock counts for coursework, not live offers.
	type product struct { supplier, name, category, description, specs, image string; price, stock int }
	products := []product{
		{"techline", "Logitech K120", "Keyboards", "Полноразмерная проводная клавиатура для дома и офиса. Русско-английская раскладка, защита от случайных брызг.", `{"Brand":"Logitech","Connection":"USB-A","Layout":"RU/EN","Layout type":"Full-size","Warranty":"12 months"}`, "https://www.jbhifi.com.au/cdn/shop/products/394698-Product-0-I_92f5e9a0-1bff-47f3-a08d-5b92c3fdbbe4_1024x1024.png", 8990, 18},
		{"techline", "Logitech K380s Pebble Keys 2", "Keyboards", "Компактная Bluetooth-клавиатура с подключением к нескольким устройствам и тихими клавишами.", `{"Brand":"Logitech","Connection":"Bluetooth LE","Devices":"Up to 3","Power":"2 AAA batteries","Warranty":"12 months"}`, "https://resource.logitech.com/e_trim/w_700%2Ch_600%2Cc_pad%2Cq_auto%2Cf_auto%2Cdpr_1.0/b_white/content/dam/logitech/en/products/keyboards/pebble-keys-2-k380s/gallery/pebble-keys-2-k380s-top-tonal-graphite-gallery-us.png", 24990, 9},
		{"gadgethouse", "Redragon K552 Kumara", "Keyboards", "Компактная механическая клавиатура с красными переключателями и подсветкой для игр и набора текста.", `{"Brand":"Redragon","Connection":"USB-A","Switches":"Red mechanical","Layout":"87 keys","Backlight":"Red"}`, "https://vivadigital.com.co/img/products/Teclado_Mecnico_KUMARA_K552_Redragon_Black/Teclado_Mecnico_KUMARA_K552_Redragon_Black16504021321.jpg", 23990, 7},
		{"techline", "Logitech M185", "Mice", "Компактная беспроводная мышь с USB-приёмником, подходящая для ноутбука и повседневной работы.", `{"Brand":"Logitech","Connection":"2.4 GHz USB receiver","DPI":"1000","Power":"1 AA battery","Warranty":"12 months"}`, "https://bizweb.dktcdn.net/100/497/222/products/6988-chu-t-logitech-m185-wireless-a5498d7f-d936-49ce-94bf-262ce6c9953c.jpg?v=1701145770907", 6990, 25},
		{"gadgethouse", "Logitech G102 LIGHTSYNC", "Mice", "Игровая проводная мышь с регулируемым сенсором и настраиваемой RGB-подсветкой.", `{"Brand":"Logitech G","Connection":"USB-A","DPI":"200–8000","Buttons":"6 programmable","Lighting":"LIGHTSYNC RGB"}`, "https://img.gkbcdn.com/s3/d/202004/770eadbb-b0b2-4033-a1b5-114287439435.jpg", 17990, 11},
		{"officehub", "Logitech C920 HD Pro", "Webcams", "Веб-камера Full HD с двумя микрофонами для онлайн-занятий, встреч и видеозвонков.", `{"Brand":"Logitech","Resolution":"1080p at 30 fps","Microphones":"Dual stereo","Connection":"USB-A","Lens":"78° field of view"}`, "https://www.cecomsa.com/web/image/product.template/24905/image_1024?unique=3637dfc", 42990, 6},
		{"officehub", "A4Tech PK-910H", "Webcams", "Веб-камера Full HD с автоматической коррекцией изображения для видеосвязи.", `{"Brand":"A4Tech","Resolution":"1080p","Microphone":"Built-in","Connection":"USB-A","Focus":"Fixed"}`, "https://object.pscloud.io/cms/cms/Photo/img_0_342_109_3_1.jpg", 18990, 10},
		{"datapoint", "Samsung T7 Portable SSD 1 TB", "Storage", "Портативный SSD с интерфейсом USB 3.2 Gen 2 для резервных копий и переноса больших файлов.", `{"Brand":"Samsung","Capacity":"1 TB","Interface":"USB 3.2 Gen 2 Type-C","Read speed":"Up to 1,050 MB/s","Warranty":"12 months"}`, "https://paksell.pk/cdn/shop/products/samsung-t7-portable-ssd-1tb-usb32-gen-2-type-c-external-ssd-438520.jpg?v=1691492418&width=800", 58990, 8},
		{"datapoint", "Kingston XS1000 Portable SSD 1 TB", "Storage", "Компактный внешний SSD для учёбы, проектов и резервного копирования данных.", `{"Brand":"Kingston","Capacity":"1 TB","Interface":"USB 3.2 Gen 2","Read speed":"Up to 1,050 MB/s","Warranty":"12 months"}`, "https://oechsle.vteximg.com.br/arquivos/ids/23817662-1000-1000/imageUrl_1.jpg?v=639040537531100000", 49990, 10},
		{"datapoint", "WD Blue SN580 NVMe SSD 1 TB", "Storage", "Внутренний NVMe-накопитель формата M.2 для совместимого ноутбука или настольного компьютера.", `{"Brand":"Western Digital","Capacity":"1 TB","Interface":"PCIe Gen 4 x4 NVMe","Form factor":"M.2 2280","Warranty":"36 months"}`, "https://cdn.ballicom.co.uk/?r=peyJpbWciOiJcL1wvaW1hZ2VzXC9jZG5cL2ZkXC9jNlwvZmRjNmFhMTMtZmJkZS00ZjI4LTk4OGQtOWRmNzAwMWQ3ZTQyLmpwZyIsInNpemUiOjg1MCwiZXh0ZW5zaW9uIjoianBnIn0%3Db", 37990, 5},
		{"officehub", "Samsung Essential Monitor S3 24 inch", "Monitors", "24-дюймовый IPS-монитор Full HD с тонкими рамками для работы, учёбы и дома.", `{"Brand":"Samsung","Size":"24 inch","Resolution":"1920 × 1080","Panel":"IPS, 100 Hz","Ports":"HDMI, D-Sub"}`, "https://neom.kz/upload/iblock/e53/rjpdatre81gfknm3y2y12fafqtlfhn99.jpg", 69990, 5},
		{"officehub", "Xiaomi Monitor A27i 27 inch", "Monitors", "27-дюймовый IPS-монитор Full HD с частотой 100 Гц и тонким корпусом.", `{"Brand":"Xiaomi","Size":"27 inch","Resolution":"1920 × 1080","Panel":"IPS, 100 Hz","Ports":"HDMI, DisplayPort"}`, "https://www.proshop.dk/Images/915x900/3238265_54995d01098b.jpg", 79990, 4},
		{"gadgethouse", "AOC 24B2XH 23.8 inch", "Monitors", "Компактный IPS-монитор Full HD с матовым экраном для офисной работы.", `{"Brand":"AOC","Size":"23.8 inch","Resolution":"1920 × 1080","Panel":"IPS, 75 Hz","Ports":"HDMI, VGA"}`, "https://desktop.bg/system/images/272049/double/aoc_24b2xheu.png", 59990, 7},
		{"gadgethouse", "HyperX Cloud Stinger 2 Core", "Headsets", "Лёгкая игровая гарнитура с поворотным микрофоном и мягкими амбушюрами.", `{"Brand":"HyperX","Connection":"3.5 mm","Microphone":"Swivel-to-mute","Audio":"Stereo","Platform":"PC and console"}`, "https://cdn.cs.1worldsync.com/syndication/mediaserverredirect/8244d46464d238a3a7d28fddcf21e025/width%283000%29.webp", 22990, 8},
		{"gadgethouse", "JBL Quantum 100", "Headsets", "Проводная игровая гарнитура с отсоединяемым микрофоном и объёмным звуком JBL QuantumSOUND.", `{"Brand":"JBL","Connection":"3.5 mm","Microphone":"Detachable boom mic","Audio":"JBL QuantumSOUND","Cable":"1.2 m"}`, "https://www.jbl.com/on/demandware.static/-/Sites-masterCatalog_Harman/default/dw9fa4cceb/JBL_Quantum_100_Product%20Image_Hero_Black.png", 19990, 12},
		{"datapoint", "TP-Link Archer C6 AC1200", "Networking", "Двухдиапазонный Wi‑Fi-роутер с гигабитными портами для квартиры и небольшого офиса.", `{"Brand":"TP-Link","Wi-Fi":"AC1200 dual-band","Ports":"4 × Gigabit LAN, 1 × Gigabit WAN","Antennas":"4 external","Security":"WPA/WPA2"}`, "https://www.dateks.lv/images/pic/2400/2400/475/1062.jpg", 24990, 6},
		{"datapoint", "TP-Link Archer T3U Plus AC1300", "Networking", "USB Wi‑Fi адаптер с внешней антенной для подключения настольного компьютера к беспроводной сети.", `{"Brand":"TP-Link","Interface":"USB 3.0","Wi-Fi":"AC1300 dual-band","Antenna":"High-gain external","Security":"WPA/WPA2"}`, "https://files.pccasegear.com/images/1587082608-ARCHER-T3U-PLUS-thb.jpg", 10990, 14},
		{"gadgethouse", "UGREEN 6-in-1 USB-C Hub", "Accessories", "Многофункциональный USB-C адаптер с HDMI, USB-A и чтением карт памяти.", `{"Brand":"UGREEN","Host connection":"USB-C","Ports":"HDMI, 3 × USB-A, SD, microSD","HDMI output":"Up to 4K at 30 Hz","Power":"USB-C PD pass-through"}`, "https://down-br.img.susercontent.com/file/br-11134207-7r98p-lllb4dtuavlm2b", 18990, 10},
		{"gadgethouse", "Baseus GaN5 Pro 65 W Charger", "Power", "Компактное сетевое зарядное устройство GaN с портами USB-C и USB-A для ноутбука и телефона.", `{"Brand":"Baseus","Max output":"65 W","Ports":"2 × USB-C, 1 × USB-A","Technology":"GaN","Protection":"Over-current and over-temperature"}`, "https://www.mirao.com.br/media/catalog/product/cache/3e2c2cbb89ac2ad7f8b7f37e77054bc6/c/a/carregador-rapido-baseus-gan5-pro-2c-u-65w-preto-ccgp120201-1.jpg", 17990, 9},
		{"officehub", "APC Back-UPS 650VA", "Power", "Источник бесперебойного питания для роутера, компьютера и периферийных устройств.", `{"Brand":"APC","Capacity":"650 VA / 360 W","Outlets":"4","Protection":"Surge and battery backup","Warranty":"12 months"}`, "https://noventiqstore.kz/files/resized/products/00bb71a6-436c-4b77-8372-25a6b0193355default.1920x1080.png", 44990, 3},
		{"officehub", "Logitech Brio 300", "Webcams", "Компактная Full HD веб-камера с автоэкспозицией, крышкой объектива и встроенным микрофоном.", `{"Brand":"Logitech","Resolution":"1080p at 30 fps","Connection":"USB-C","Privacy":"Integrated shutter","Microphone":"Mono with noise reduction"}`, "https://www.logitech.com/content/dam/logitech/en/products/webcams/brio-300/gallery/brio-300-gallery-graphite-4.png", 32990, 6},
	}
	for _, p := range products {
		s := suppliers[p.supplier]
		var uid int64
		err = tx.QueryRow(`SELECT id FROM users WHERE email=$1`, s.email).Scan(&uid)
		if err == sql.ErrNoRows {
			hash, e := bcrypt.GenerateFromPassword([]byte("Supplier123!"), bcrypt.DefaultCost)
			if e != nil { return e }
			err = tx.QueryRow(`INSERT INTO users(name,email,password_hash,role) VALUES($1,$2,$3,'supplier') RETURNING id`, s.company, s.email, string(hash)).Scan(&uid)
		}
		if err != nil { return err }
		var sid int64
		err = tx.QueryRow(`SELECT id FROM suppliers WHERE user_id=$1`, uid).Scan(&sid)
		if err == sql.ErrNoRows {
			err = tx.QueryRow(`INSERT INTO suppliers(user_id,company_name,description,verified) VALUES($1,$2,$3,true) RETURNING id`, uid, s.company, s.description).Scan(&sid)
		} else if err == nil {
			_, err = tx.Exec(`UPDATE suppliers SET description=$1,verified=true WHERE id=$2`, s.description, sid)
		}
		if err != nil { return err }
		var exists bool
		if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM products WHERE supplier_id=$1 AND name=$2)`, sid, p.name).Scan(&exists); err != nil { return err }
		if exists { continue }
		imageURL := p.image
		if !strings.HasPrefix(imageURL, "https://") {
			imageURL = "/assets/" + imageURL
		}
		if _, err = tx.Exec(`INSERT INTO products(supplier_id,category_id,name,description,specs,price_kzt,stock,image_url,approved) VALUES($1,$2,$3,$4,$5::jsonb,$6,$7,$8,true)`, sid, categoryIDs[p.category], p.name, p.description, p.specs, strconv.Itoa(p.price), p.stock, imageURL); err != nil { return err }
	}
	return tx.Commit()
}
