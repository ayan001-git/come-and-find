const $ = s => document.querySelector(s);
const fmt = n => new Intl.NumberFormat('ru-RU', {maximumFractionDigits:0}).format(Number(n)) + ' ₸';
let me = null, categories = [];
let allProducts = [], visibleProductCount = 0;
const PAGE_SIZE = 9;
const categoryArtwork = {Keyboards:'/assets/keyboard.svg',Mice:'/assets/mouse.svg',Webcams:'/assets/webcam.svg',Storage:'/assets/ssd.svg',Monitors:'/assets/monitor.svg'};
function productImage(p){return !p.image_url || p.image_url === '/assets/product.svg' ? (categoryArtwork[p.category] || '/assets/product.svg') : p.image_url;}
const demoAccounts = {buyer:['buyer@example.kz','Buyer123!'],supplier:['supplier@techline.kz','Supplier123!'],admin:['admin@comeandfind.kz','Admin123!']};

async function api(url, options={}) {
  const cfg = {credentials:'same-origin',headers:{'Content-Type':'application/json'}, ...options};
  const res = await fetch(url, cfg);
  const data = await res.json().catch(()=>({}));
  if(res.status===401 && me){me=null;renderAuth();$('#dashboard').classList.add('hidden');}
  if (!res.ok) throw new Error(data.error || 'Request failed');
  return data;
}
let toastTimer; function toast(msg){const t=$('#toast');t.textContent=msg;t.classList.remove('hidden');clearTimeout(toastTimer);toastTimer=setTimeout(()=>t.classList.add('hidden'),3400)}
function esc(s=''){return String(s).replace(/[&<>'"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c]))}

async function init(){
  try{me=await api('/api/me')}catch{me=null}
  categories=await api('/api/categories');
  $('#categorySelect').innerHTML='<option value="">All categories</option>'+categories.map(c=>`<option value="${c.id}">${esc(c.name)}</option>`).join('');
  bind(); renderAuth(); await loadProducts();
}
function bind(){
  $('#loadMoreBtn').onclick=renderProducts;
  document.querySelectorAll('[data-demo-role]').forEach(b=>b.addEventListener('click',()=>{const [email,password]=demoAccounts[b.dataset.demoRole];$('#loginEmail').value=email;$('#loginPassword').value=password;$('#authMessage').textContent='Demo '+b.dataset.demoRole+' selected. Click Sign in.';$('#loginPassword').focus();}));
  $('#searchBtn').onclick=loadProducts; $('#categorySelect').onchange=loadProducts; $('#clearFilters').onclick=()=>{for(const id of ['searchInput','minPrice','maxPrice'])$('#'+id).value='';$('#categorySelect').value='';return loadProducts()};
  $('#authBtn').onclick=openAuth; $('#heroLogin').onclick=openAuth; $('#authClose').onclick=()=>$('#authDialog').close(); $('#productClose').onclick=()=>$('#productDialog').close(); $('#cartClose').onclick=()=>$('#cartDialog').close();
  $('#loginTab').onclick=()=>switchAuth('login'); $('#registerTab').onclick=()=>switchAuth('register'); $('#regRole').onchange=()=>$('#companyLabel').classList.toggle('hidden',$('#regRole').value!=='supplier');
  $('#loginSubmit').onclick=login; $('#registerSubmit').onclick=register; $('#logoutBtn').onclick=logout; $('#cartBtn').onclick=openCart; $('#dashboardBtn').onclick=openDashboard;
}
function renderAuth(){
 if(!me){$('#whoami').textContent='';$('#cartCount').textContent='0';$('#dashboardContent').innerHTML='';}
 $('#heroLogin').textContent=me?'Open dashboard':'Sign in';$('#heroLogin').onclick=me?openDashboard:openAuth;
  $('#authBtn').classList.toggle('hidden',!!me); $('#logoutBtn').classList.toggle('hidden',!me); $('#dashboardBtn').classList.toggle('hidden',!me); $('#cartBtn').classList.toggle('hidden',!me||me.role!=='buyer');
  if(me){$('#whoami').textContent=`${me.name} • ${me.role}`; if(me.role==='buyer')refreshCartCount();}
}
function openAuth(){for(const id of ['productDialog','cartDialog'])$('#'+id).close();switchAuth('login');$('#authMessage').textContent='';$('#authDialog').showModal();$('#loginEmail').focus()}
function switchAuth(mode){const login=mode==='login';$('#loginPane').classList.toggle('hidden',!login);$('#registerPane').classList.toggle('hidden',login);$('#loginTab').classList.toggle('active',login);$('#registerTab').classList.toggle('active',!login)}
async function login(){try{me=await api('/api/auth/login',{method:'POST',body:JSON.stringify({email:$('#loginEmail').value,password:$('#loginPassword').value})});$('#authDialog').close();$('#loginPassword').value='';renderAuth();toast(`Signed in as ${me.role}`)}catch(e){msg(e.message)}}
async function register(){try{me=await api('/api/auth/register',{method:'POST',body:JSON.stringify({name:$('#regName').value,email:$('#regEmail').value,password:$('#regPassword').value,role:$('#regRole').value,company:$('#regCompany').value})});$('#authDialog').close();$('#regPassword').value='';renderAuth();toast('Account created')}catch(e){msg(e.message)}}
async function logout(){await api('/api/auth/logout',{method:'POST'});me=null;renderAuth();$('#dashboard').classList.add('hidden');toast('Logged out')}
function msg(s){const m=$('#authMessage');m.textContent=s;m.className='message error'}

let catalogueRequest=0;
async function loadProducts(){
 const request=++catalogueRequest;
  const qs=new URLSearchParams(); if($('#searchInput').value)qs.set('search',$('#searchInput').value);if($('#categorySelect').value)qs.set('category',$('#categorySelect').value);
  for(const [id,key] of [['minPrice','min_price'],['maxPrice','max_price']])if($('#'+id).value!=='')qs.set(key,$('#'+id).value);
  const [products,intent]=await Promise.all([api('/api/products?'+qs.toString()),api('/api/search/interpret?q='+encodeURIComponent($('#searchInput').value))]); if(request!==catalogueRequest)return;
  const hints=[];if(intent.category)hints.push('Category: '+intent.category);if(intent.terms.length)hints.push('Terms: '+intent.terms.join(', '));
  const min=$('#minPrice').value!==''?Number($('#minPrice').value):intent.min_price;const max=$('#maxPrice').value!==''?Number($('#maxPrice').value):intent.max_price;
  if(min!==undefined&&min!==null)hints.push('Min: '+fmt(min));if(max!==undefined&&max!==null)hints.push('Max: '+fmt(max));if($('#categorySelect').value)hints.push('Selected category also applies');
  $('#searchHint').textContent=hints.length?'Search understood as: '+hints.join(' · '):'Try “mouse under 10 thousand” or “мышь до 10 тысяч”.';
  $('#catalogMeta').textContent=`${products.length} product${products.length===1?'':'s'}`;
  $('#resultSummary').textContent=products.length ? `Showing ${products.length} matching products` : 'No matching products';
  allProducts = products;
  visibleProductCount = 0;
  renderProducts();
}

function renderProducts(){
  const grid = $('#productGrid');
  if(!allProducts.length){
    grid.innerHTML='<div class="empty-state"><span class="empty-icon">⌕</span><h3>No products found</h3><p>Try a different name, category or price range.</p><button class="ghost" id="emptyReset">Clear filters</button></div>';
    $('#emptyReset').onclick=()=>$('#clearFilters').click();
    $('#loadMoreBtn').classList.add('hidden');
    return;
  }
  const previous = visibleProductCount;
  visibleProductCount = Math.min(visibleProductCount + PAGE_SIZE, allProducts.length);
  const cards = allProducts.slice(previous, visibleProductCount).map(p=>`<article class="product-card"><div class="product-media"><img src="${esc(productImage(p))}" alt="${esc(p.name)}" loading="lazy"><span class="media-tag">${esc(p.category)}</span></div><div class="product-body"><div class="product-top"><h3>${esc(p.name)}</h3><span class="stock ${p.stock<5?'low-stock':''}">${p.stock>0?p.stock+' available':'Out of stock'}</span></div><p class="product-description">${esc(p.description)}</p><p class="supplier-line"><span aria-hidden="true">◈</span> ${esc(p.supplier)}</p><div class="card-bottom"><div class="price">${fmt(p.price_kzt)}</div><div class="product-actions"><button class="ghost" onclick="viewProduct(${p.id})" aria-label="View ${esc(p.name)} details">Details</button><button class="btn" ${p.stock<1?'disabled':''} onclick="addToCart(${p.id})">${p.stock<1?'Sold out':'+ Add'}</button></div></div></div></article>`).join('');
  if(previous===0)grid.innerHTML=cards; else grid.insertAdjacentHTML('beforeend',cards);
  $('#loadMoreBtn').classList.toggle('hidden',visibleProductCount>=allProducts.length);
  $('#loadMoreBtn').textContent=`Show more products (${allProducts.length-visibleProductCount} remaining) ↓`;
}

window.viewProduct=async id=>{const p=await api('/api/products/'+id);const specs=Object.entries(p.specs||{}).map(([k,v])=>`<div class="spec"><b>${esc(k)}</b><br><span class="muted">${esc(v)}</span></div>`).join('');$('#productDetail').innerHTML=`<div class="detail"><img src="${esc(productImage(p))}" alt="${esc(p.name)}"><div><span class="chip">${esc(p.category)}</span><h2>${esc(p.name)}</h2><p class="muted">${esc(p.description)}</p><div class="price">${fmt(p.price_kzt)}</div><p class="stock">${p.stock} units available</p><div class="specs">${specs}</div><div class="supplier-box"><b>${esc(p.supplier.company_name)} ${p.supplier.verified?'✓':''}</b><p>${esc(p.supplier.description)}</p></div><button class="btn" ${p.stock<1?'disabled':''} onclick="addToCart(${p.id})">${p.stock<1?'Out of stock':'Add to cart'}</button></div></div>`;$('#productDialog').showModal()}
window.addToCart=async id=>{if(!me){openAuth();return}if(me.role!=='buyer'){toast('Use a buyer account to shop');return}try{await api('/api/cart',{method:'POST',body:JSON.stringify({product_id:id,quantity:1})});toast('Added to cart');await refreshCartCount()}catch(e){toast(e.message)}}
async function refreshCartCount(){try{const c=await api('/api/cart');$('#cartCount').textContent=c.items.reduce((a,x)=>a+x.quantity,0)}catch{}}
async function openCart(){const c=await api('/api/cart');$('#cartContent').innerHTML=c.items.length?c.items.map(i=>`<div class="cart-row"><img src="${esc(i.image_url)}"><div><b>${esc(i.name)}</b><br><span class="quantity"><button class="ghost" aria-label="Decrease quantity" ${i.quantity<=1?'disabled':''} onclick="setQuantity(${i.id},${i.quantity-1})">−</button><span>Qty: ${i.quantity}</span><button class="ghost" aria-label="Increase quantity" ${i.quantity>=i.stock?'disabled':''} onclick="setQuantity(${i.id},${i.quantity+1})">+</button></span></div><div class="price">${fmt(i.price_kzt*i.quantity)}</div><button class="ghost" onclick="removeCart(${i.id})">Remove</button></div>`).join('')+`<div class="cart-total"><span>Total</span><span>${fmt(c.total_kzt)}</span></div><button class="btn full" onclick="createOrder()">Create order</button>`:'<p class="muted">Your cart is empty.</p>';if(!$('#cartDialog').open)$('#cartDialog').showModal()}
window.setQuantity=async(id,quantity)=>{await api('/api/cart',{method:'PATCH',body:JSON.stringify({product_id:id,quantity})});await refreshCartCount();await openCart()};
window.removeCart=async id=>{await api('/api/cart/'+id,{method:'DELETE'});await refreshCartCount();await openCart()}
window.createOrder=async()=>{try{const o=await api('/api/orders',{method:'POST'});$('#cartDialog').close();await refreshCartCount();toast(`Order #${o.id} created`);await loadProducts();await openDashboard()}catch(e){toast(e.message)}}

async function openDashboard(){if(!me)return;$('#dashboard').classList.remove('hidden');$('#dashboard').scrollIntoView({behavior:'smooth'});$('#dashboardTitle').textContent=me.role==='buyer'?'My orders':me.role==='supplier'?'Supplier dashboard':'Admin dashboard';if(me.role==='buyer')await buyerDashboard();if(me.role==='supplier')await supplierDashboard();if(me.role==='admin')await adminDashboard()}
async function buyerDashboard(){const orders=await api('/api/orders');$('#dashboardContent').innerHTML=`<div class="panel"><h3>Orders</h3>${orders.length?`<table class="data-table"><thead><tr><th>ID</th><th>Total</th><th>Payment</th><th>Status</th><th>Created</th><th>Action</th></tr></thead><tbody>${orders.map(o=>`<tr><td>#${o.id}</td><td>${fmt(o.total_kzt)}</td><td>${esc(o.payment_status)}</td><td><span class="status ${o.status==='delivered'?'ok':'wait'}">${esc(o.status)}</span></td><td>${new Date(o.created_at).toLocaleString()}</td><td>${o.status==='created'?`<button class="btn small" onclick="payOrder(${o.id})">Simulate payment</button>`:'—'}</td></tr>`).join('')}</tbody></table>`:'<p class="muted">No orders yet. Add a product to your cart first.</p>'}</div>`}
window.payOrder=async id=>{try{await api(`/api/orders/${id}/pay`,{method:'POST'});toast('Payment simulated successfully');await buyerDashboard()}catch(e){toast(e.message)}}

async function supplierDashboard(){const [products,orders]=await Promise.all([api('/api/supplier/products'),api('/api/supplier/orders')]);$('#dashboardContent').innerHTML=`<div class="panel"><h3>Add product</h3><div class="form-grid"><input id="spName" class="field" placeholder="Product name"><select id="spCat" class="field">${categories.map(c=>`<option value="${c.id}">${esc(c.name)}</option>`).join('')}</select><input id="spPrice" class="field" type="number" min="0" placeholder="Price KZT"><input id="spStock" class="field" type="number" min="0" placeholder="Stock"><input id="spImage" class="field fullrow" placeholder="Optional image: /assets/product.svg"><input id="spSpecs" class="field fullrow" placeholder='Specifications JSON: {"Connection":"USB"}'><input id="spDesc" class="field fullrow" placeholder="Short description"><button class="btn fullrow" onclick="supplierAddProduct()">Submit for approval</button></div></div><div class="panel"><h3>My products</h3><table class="data-table"><thead><tr><th>ID</th><th>Product</th><th>Category</th><th>Price</th><th>Stock</th><th>Approval</th></tr></thead><tbody>${products.map(p=>`<tr><td>${p.id}</td><td>${esc(p.name)}</td><td>${esc(p.category)}</td><td>${fmt(p.price_kzt)}</td><td>${p.stock}</td><td><span class="status ${p.approved?'ok':'wait'}">${p.approved?'Approved':'Pending'}</span></td></tr>`).join('')}</tbody></table></div><div class="panel"><h3>Related orders</h3>${orders.length?`<table class="data-table"><thead><tr><th>ID</th><th>Buyer</th><th>Total</th><th>Payment</th><th>Status</th><th>Next</th></tr></thead><tbody>${orders.map(o=>`<tr><td>#${o.id}</td><td>${esc(o.buyer)}</td><td>${fmt(o.total_kzt)}</td><td>${esc(o.payment_status)}</td><td>${esc(o.status)}</td><td>${supplierNextButton(o)}</td></tr>`).join('')}</tbody></table>`:'<p class="muted">No supplier orders yet.</p>'}</div>`}
function supplierNextButton(o){if(o.payment_status!=='simulated_paid')return 'Waiting payment';const map={paid:'processing',processing:'shipped',shipped:'delivered'};const next=map[o.status];return next?`<button class="btn small" onclick="supplierStatus(${o.id},'${next}')">Mark ${next}</button>`:'—'}
window.supplierAddProduct=async()=>{try{await api('/api/supplier/products',{method:'POST',body:JSON.stringify({name:$('#spName').value,category_id:Number($('#spCat').value),price_kzt:Number($('#spPrice').value),stock:Number($('#spStock').value),description:$('#spDesc').value,image_url:$('#spImage').value.trim(),specs:$('#spSpecs').value.trim()?JSON.parse($('#spSpecs').value):{}})});toast('Product submitted for admin approval');await supplierDashboard()}catch(e){toast(e.message)}}
window.supplierStatus=async(id,status)=>{try{await api(`/api/supplier/orders/${id}/status`,{method:'POST',body:JSON.stringify({status})});toast('Order status updated');await supplierDashboard()}catch(e){toast(e.message)}}

async function adminDashboard(){const [stats,products,users,orders]=await Promise.all([api('/api/admin/stats'),api('/api/admin/products'),api('/api/admin/users'),api('/api/admin/orders')]);$('#dashboardContent').innerHTML=`<div class="dashboard-grid"><div class="metric"><span class="muted">Users</span><b>${stats.users}</b></div><div class="metric"><span class="muted">Products</span><b>${stats.products}</b></div><div class="metric"><span class="muted">Pending</span><b>${stats.pending_products}</b></div><div class="metric"><span class="muted">Orders</span><b>${stats.orders}</b></div></div><div class="panel"><h3>Product approval</h3><table class="data-table"><thead><tr><th>ID</th><th>Product</th><th>Supplier</th><th>Category</th><th>Price</th><th>Status</th><th>Action</th></tr></thead><tbody>${products.map(p=>`<tr><td>${p.id}</td><td>${esc(p.name)}</td><td>${esc(p.supplier)}</td><td>${esc(p.category)}</td><td>${fmt(p.price_kzt)}</td><td><span class="status ${p.approved?'ok':'wait'}">${p.approved?'Approved':'Pending'}</span></td><td>${p.approved?`<button class="ghost" onclick="moderateProduct(${p.id},'withdraw')">Withdraw</button>`:`<button class="btn small" onclick="approveProduct(${p.id})">Approve</button><button class="ghost" onclick="moderateProduct(${p.id},'reject')">Reject / keep hidden</button>`}</td></tr>`).join('')}</tbody></table></div><div class="panel"><h3>All orders</h3>${orders.length?`<table class="data-table"><thead><tr><th>ID</th><th>Buyer</th><th>Total</th><th>Payment</th><th>Status</th><th>Created</th></tr></thead><tbody>${orders.map(o=>`<tr><td>#${o.id}</td><td>${esc(o.buyer)}</td><td>${fmt(o.total_kzt)}</td><td>${esc(o.payment_status)}</td><td>${esc(o.status)} ${adminNextButton(o)}</td><td>${new Date(o.created_at).toLocaleString()}</td></tr>`).join('')}</tbody></table>`:'<p class="muted">No orders yet.</p>'}</div><div class="panel"><h3>Users</h3><table class="data-table"><thead><tr><th>ID</th><th>Name</th><th>Email</th><th>Role</th><th>Active</th><th>Action</th></tr></thead><tbody>${users.map(u=>`<tr><td>${u.id}</td><td>${esc(u.name)}</td><td>${esc(u.email)}</td><td>${esc(u.role)}</td><td>${u.active?'Yes':'No'}</td><td>${u.role==='admin'?'Protected':`<button class="ghost" onclick="toggleUser(${u.id},${!u.active})">${u.active?'Deactivate':'Activate'}</button>`}</td></tr>`).join('')}</tbody></table></div>`}
window.approveProduct=async id=>{try{await api(`/api/admin/products/${id}/approve`,{method:'POST'});toast('Product approved');await adminDashboard();await loadProducts()}catch(e){toast(e.message)}}

function adminNextButton(o){const next={paid:'processing',processing:'shipped',shipped:'delivered'}[o.status];return next&&o.payment_status==='simulated_paid'?`<button class="ghost" onclick="adminStatus(${o.id},'${next}')">${next}</button>`:''}
window.adminStatus=async(id,status)=>{await api('/api/admin/orders/'+id,{method:'POST',body:JSON.stringify({status})});await adminDashboard()};
window.moderateProduct=async(id,action)=>{await api('/api/admin/products/'+id+'/'+action,{method:'POST',body:'{}'});await adminDashboard();await loadProducts()};
window.toggleUser=async(id,active)=>{await api('/api/admin/users/'+id,{method:'POST',body:JSON.stringify({active})});await adminDashboard();await loadProducts()};
// Catch async UI failures and block duplicate clicks while a mutation is pending.
function safeAction(fn){return async function(...args){if(safeAction.busy)return;safeAction.busy=true;const buttons=[...document.querySelectorAll('button:not(#voiceBtn)')];const previous=buttons.map(b=>b.disabled);buttons.forEach(b=>b.disabled=true);try{return await fn.apply(this,args)}catch(e){toast(e.message||'Operation failed');if(fn===loadProducts){$('#searchHint').textContent='Search failed: '+e.message;$('#productGrid').innerHTML='';$('#catalogMeta').textContent=''}}finally{safeAction.busy=false;buttons.forEach((b,i)=>{if(b.isConnected)b.disabled=previous[i]})}}}
for(const name of ['viewProduct','addToCart','removeCart','setQuantity','createOrder','payOrder','supplierAddProduct','supplierStatus','approveProduct','moderateProduct','toggleUser','adminStatus'])window[name]=safeAction(window[name]);
const originalBind=bind;bind=function(){originalBind();for(const id of ['searchBtn','clearFilters','loginSubmit','registerSubmit','logoutBtn','cartBtn','dashboardBtn','heroLogin']){const e=$('#'+id);if(e.onclick)e.onclick=safeAction(e.onclick)}$('#categorySelect').onchange=safeAction(loadProducts);$('#searchInput').onkeydown=e=>{if(e.key==='Enter'){e.preventDefault();safeAction(loadProducts)()}}};
init().catch(e=>{console.error(e);toast('Server error: '+e.message)});
